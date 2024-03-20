package operator

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	sigyaml "sigs.k8s.io/yaml"

	"github.com/calyptia/cli/cmd/utils"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/component-base/logs"
	kubectl "k8s.io/kubectl/pkg/cmd"

	"github.com/calyptia/cli/k8s"
)

//go:embed manifest.yaml
var f embed.FS

const manifestFile = "manifest.yaml"

func NewCmdInstall() *cobra.Command {
	var (
		coreInstanceVersion        string
		coreDockerImage            string
		isNonInteractive           bool
		waitReady                  bool
		waitTimeout                time.Duration
		confirmed                  bool
		externalTrafficPolicyLocal bool
	)

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	cmd := &cobra.Command{
		Use:     "operator",
		Aliases: []string{"opr"},
		Short:   "Setup a new core operator instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			var namespace string

			kubeNamespaceFlag := cmd.Flag("kube-namespace")
			if kubeNamespaceFlag != nil {
				namespace = kubeNamespaceFlag.Value.String()
			}

			if namespace == "" {
				namespace = apiv1.NamespaceDefault
			}

			n, err := k8s.GetCurrentContextNamespace()
			if err != nil {
				if errors.Is(err, k8s.ErrNoContext) {
					cmd.Printf("No context is currently set. Using default namespace.\n")
				} else {
					return err
				}
			}
			if n != "" {
				namespace = n
			}

			kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
			kubeClientConfig, err := kubeConfig.ClientConfig()
			if err != nil {
				return err
			}

			clientSet, err := kubernetes.NewForConfig(kubeClientConfig)
			if err != nil {
				return err
			}
			k := &k8s.Client{
				Interface: clientSet,
				Config:    kubeClientConfig,
			}
			if !confirmed {
				isInstalled, err := k.IsOperatorInstalled(cmd.Context())
				if isInstalled {
					var e *k8s.OperatorIncompleteError
					if errors.As(err, &e) {
						cmd.Printf("Previous operator installation components found:\n%s\n", e.Error())
						cmd.Printf("Are you sure you want to proceed? (y/N) ")
						var answer string
						_, err := fmt.Scanln(&answer)
						if err != nil && err.Error() == "unexpected newline" {
							err = nil
						}

						if err != nil {
							return fmt.Errorf("could not to read answer: %v", err)
						}

						answer = strings.TrimSpace(strings.ToLower(answer))
						if answer != "y" && answer != "yes" {
							return nil
						}
					}
				}
			}

			_, err = k.GetNamespace(context.Background(), namespace)
			if err != nil && !k8serrors.IsNotFound(err) {
				return err
			}

			manifest, err := installManifest(namespace, coreDockerImage,
				coreInstanceVersion, k8serrors.IsNotFound(err), externalTrafficPolicyLocal)
			if err != nil {
				return err
			}
			defer os.RemoveAll(manifest)

			if waitReady {
				deployment, err := extractDeployment(manifest)
				if err != nil {
					return err
				}
				start := time.Now()
				fmt.Println("Waiting for core operator manager to be ready...")
				err = k.WaitReady(context.Background(), namespace, deployment, false, waitTimeout)
				if err != nil {
					return err
				}
				fmt.Printf("Core operator manager is ready. Took %s\n", time.Since(start))
			}

			cmd.Printf("Core operator manager successfully installed.\n")
			return nil
		},
	}

	fs := cmd.Flags()

	fs.BoolVarP(&confirmed, "yes", "y", isNonInteractive, "Confirm install")
	fs.BoolVar(&waitReady, "wait", false, "Wait for the core instance to be ready before returning")
	fs.DurationVar(&waitTimeout, "timeout", time.Second*30, "Wait timeout")
	fs.StringVar(&coreInstanceVersion, "version", "", "Core instance version")
	fs.StringVar(&coreDockerImage, "image", utils.DefaultCoreOperatorDockerImage, "Calyptia core manager docker image to use (fully composed docker image).")
	fs.BoolVar(&externalTrafficPolicyLocal, "external-traffic-policy-local", false, "Set ExternalTrafficPolicy to local for all services used by core operator pipelines.")
	_ = cmd.Flags().MarkHidden("image")
	clientcmd.BindOverrideFlags(configOverrides, fs, clientcmd.RecommendedConfigOverrideFlags("kube-"))

	return cmd
}

// extractDeployment extracts the name of the deployment from the yaml file
// provided. It assumes that the last yaml document is the deployment.
// This is a temporary solution until we have a better way to do this.
// Possibly we will strip it out when we change the way we install the
// operator.
func extractDeployment(yml string) (string, error) {
	file, err := os.ReadFile(yml)
	if err != nil {
		return "", err
	}
	splitFile := strings.Split(string(file), "---\n")
	deployment := splitFile[len(splitFile)-1]
	var deploymentConfig struct {
		Metadata struct {
			Name string `yaml:"name"`
		}
	}
	err = yaml.Unmarshal([]byte(deployment), &deploymentConfig)
	if err != nil {
		return "", err
	}
	deployName := deploymentConfig.Metadata.Name
	return deployName, nil
}

func enableExternalTrafficPolicyLocal(manifests manifests) manifests {
	for idx, manifest := range manifests {
		if manifest.Kind == "Deployment" {
			deployment, ok := manifest.Descriptor.(appsv1.Deployment)
			if !ok {
				return manifests
			}

			for cidx, container := range deployment.Spec.Template.Spec.Containers {
				if container.Command[0] == "/manager" {
					container.Args = append(container.Args,
						"enable-external-traffic-policy-local")
				}
				deployment.Spec.Template.Spec.Containers[cidx] = container
			}
			manifest.Descriptor = deployment
		}
		manifests[idx] = manifest
		break
	}

	return manifests
}

func prepareInstallManifest(coreDockerImage, coreInstanceVersion, namespace string, createNamespace, externalTrafficPolicyLocal bool) (string, error) {
	manifests, err := parseManifest(manifestFile)
	if err != nil {
		return "", err
	}

	*manifests = solveNamespaceCreation(createNamespace, *manifests, namespace)
	*manifests = injectNamespace(*manifests, namespace)

	if externalTrafficPolicyLocal {
		*manifests = enableExternalTrafficPolicyLocal(*manifests)
	}

	*manifests, err = addImage(*manifests, coreDockerImage, coreInstanceVersion)
	if err != nil {
		return "", err
	}

	dir, err := os.MkdirTemp("", "calyptia-operator")
	if err != nil {
		return "", err
	}

	temp, err := os.CreateTemp(dir, "operator_*.yaml")
	if err != nil {
		return "", err
	}

	enc := yaml.NewEncoder(temp)

	for _, manifest := range *manifests {
		enc.Encode(manifest)
	}

	return temp.Name(), err
}

func solveNamespaceCreation(createNamespace bool, manifests manifests, namespace string) manifests {
	for idx, manifest := range manifests {
		if !createNamespace {
			if manifest.Kind == "Namespace" {
				manifests = append(manifests[:idx], manifests[idx+1:]...)
			}
		} else {
			manifests[idx].Metadata["name"] = namespace
		}

		break
	}

	return manifests
}

func addImage(manifests manifests, coreDockerImage, coreInstanceVersion string) (manifests, error) {
	if coreInstanceVersion == "" {
		return manifests, nil
	}
	for idx := range manifests {
		if manifests[idx].Kind == "Deployment" {
			deployment, ok := manifests[idx].Descriptor.(appsv1.Deployment)
			if !ok {
				return manifests, fmt.Errorf("unable to decipher deployment")
			}
			deployment.Spec.Template.Spec.Containers[0].Image =
				fmt.Sprintf("%s:%s", coreDockerImage, coreInstanceVersion)
			break
		}
	}
	return manifests, nil
}

func injectNamespace(manifests manifests, namespace string) manifests {
	if namespace == "" {
		namespace = "default"
	}
	for idx := range manifests {
		manifests[idx].Metadata["namespace"] = namespace
	}
	return manifests
}

func newKubectlCmd() *cobra.Command {
	_ = pflag.CommandLine.MarkHidden("log-flush-frequency")
	_ = pflag.CommandLine.MarkHidden("version")

	args := kubectl.KubectlOptions{
		IOStreams: genericclioptions.IOStreams{
			In:     os.Stdin,
			Out:    os.Stdout,
			ErrOut: os.Stderr,
		},
		Arguments: os.Args,
	}

	cmd := kubectl.NewKubectlCommand(args)

	cmd.Aliases = []string{"kc"}
	cmd.Hidden = true
	// Get handle on the original kubectl prerun so we can call it later
	originalPreRunE := cmd.PersistentPreRunE
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Call parents pre-run if exists, cobra does not do this automatically
		// See: https://github.com/spf13/cobra/issues/216
		if parent := cmd.Parent(); parent != nil {
			if parent.PersistentPreRun != nil {
				parent.PersistentPreRun(parent, args)
			}
			if parent.PersistentPreRunE != nil {
				err := parent.PersistentPreRunE(parent, args)
				if err != nil {
					return err
				}
			}
		}
		return originalPreRunE(cmd, args)
	}

	originalRun := cmd.Run
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		originalRun(cmd, args)
		return nil
	}

	logs.AddFlags(cmd.PersistentFlags())
	return cmd
}

func installManifest(namespace, coreDockerImage, coreInstanceVersion string, createNamespace, externalTrafficPolicyLocal bool) (string, error) {
	kctl := newKubectlCmd()

	manifest, err := prepareInstallManifest(coreDockerImage, coreInstanceVersion, namespace, createNamespace, externalTrafficPolicyLocal)
	if err != nil {
		return "", err
	}

	kctl.SetArgs([]string{"apply", "-f", manifest})
	err = kctl.Execute()
	if err != nil {
		return "", err
	}

	return manifest, err
}

type manifest struct {
	APIVersion string         `yaml:"apiVersion"`
	Metadata   map[string]any `yaml:"metadata"`
	Kind       string         `yaml:"kind"`
	Node       *yaml.Node     `yaml:"-"`
	Descriptor any            `yaml:"-"`
}

func (m *manifest) UnmarshalYAML(value *yaml.Node) error {
	*m = manifest{Node: value}

	for i := 0; i < len(value.Content); i += 2 {
		prop := value.Content[i]
		val := value.Content[i+1]
		switch prop.Value {
		case "apiVersion":
			m.APIVersion = val.Value
		case "kind":
			m.Kind = val.Value
		case "metadata":
			y, err := yaml.Marshal(val)
			if err != nil {
				return err
			}
			sigyaml.Unmarshal(y, &m.Metadata)
		}
	}

	switch m.Kind {
	case "Namespace":
		ns := apiv1.Namespace{}
		value.Decode(&ns)
		m.Descriptor = ns
	case "CustomResourceDefinition":
		m.Descriptor = value
	case "ServiceAccount":
		sa := apiv1.ServiceAccount{}
		value.Decode(&sa)
		m.Descriptor = sa
	case "ClusterRole":
		cr := rbacv1.ClusterRole{}
		value.Decode(&cr)
		m.Descriptor = cr
	case "ClusterRoleBinding":
		crb := rbacv1.ClusterRoleBinding{}
		value.Decode(&crb)
		m.Descriptor = crb
	case "Service":
		svc := apiv1.Service{}
		value.Decode(&svc)
		m.Descriptor = svc
	case "Deployment":
		var deploy appsv1.Deployment
		y, err := yaml.Marshal(value)
		if err != nil {
			return err
		}
		sigyaml.Unmarshal(y, &deploy)
		m.Descriptor = deploy
	}
	return nil
}

func (m manifest) MarshalYAML() (interface{}, error) {
	base := map[string]any{
		"apiVersion": m.APIVersion,
		"kind":       m.Kind,
		"metadata":   m.Metadata,
	}

	var desc []byte

	if m.Kind == "CustomResourceDefinition" {
		desc, _ = yaml.Marshal(m.Descriptor)
	} else {
		desc, _ = sigyaml.Marshal(m.Descriptor)
	}

	spec := make(map[string]interface{})
	sigyaml.Unmarshal(desc, &spec)

	for k, v := range spec {
		if _, ok := base[k]; !ok {
			base[k] = v
		}
	}

	return base, nil
}

type manifests []manifest

func parseManifest(filename string) (*manifests, error) {
	manifests := make(manifests, 0)

	data, err := f.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var doc manifest
		if dec.Decode(&doc) != nil {
			break
		}
		manifests = append(manifests, doc)
	}

	return &manifests, nil
}
