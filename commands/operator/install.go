package operator

import (
	"context"
	"embed"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
	apiv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/logs"
	kubectl "k8s.io/kubectl/pkg/cmd"

	"github.com/calyptia/cli/confirm"
	"github.com/calyptia/cli/coreversions"
	"github.com/calyptia/cli/k8s"
)

//go:embed manifest.yaml
var f embed.FS

const manifestFile = "manifest.yaml"
const EnableExternalTrafficPolicyLocal = "-enable-external-traffic-policy-local=true"

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
						confirmed, err := confirm.Read(cmd.InOrStdin())
						if err != nil {
							return err
						}

						if !confirmed {
							cmd.Println("Aborted")
							return nil
						}
					}
				}
			}

			_, err = k.GetNamespace(context.Background(), namespace)
			if err != nil && !k8serrors.IsNotFound(err) {
				return err
			}

			manifest, err := installManifest(namespace, coreDockerImage, coreInstanceVersion, k8serrors.IsNotFound(err), externalTrafficPolicyLocal)
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
	fs.StringVar(&coreDockerImage, "image", coreversions.DefaultCoreOperatorDockerImage, "Calyptia core manager docker image to use (fully composed docker image).")
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

func prepareInstallManifest(coreDockerImage, coreInstanceVersion, namespace string, createNamespace bool, externalTrafficPolicyLocal bool) (string, error) {
	file, err := f.ReadFile(manifestFile)
	if err != nil {
		return "", err
	}
	fullFile := string(file)
	solveNamespace := solveNamespaceCreation(createNamespace, fullFile, namespace)
	withNamespace := injectNamespace(solveNamespace, namespace)

	withImage, err := addImage(coreDockerImage, coreInstanceVersion, withNamespace)
	if err != nil {
		return "", err
	}
	fullManifest := injectArguments(withImage, externalTrafficPolicyLocal)

	dir, err := os.MkdirTemp("", "calyptia-operator")
	if err != nil {
		return "", err
	}

	temp, err := os.CreateTemp(dir, "operator_*.yaml")
	if err != nil {
		return "", err
	}

	_, err = temp.WriteString(fullManifest)
	if err != nil {
		return "", err
	}

	return temp.Name(), err
}

func solveNamespaceCreation(createNamespace bool, fullFile string, namespace string) string {
	if !createNamespace {
		splitFile := strings.Split(fullFile, "---\n")
		return strings.Join(splitFile[1:], "---\n")
	} else {
		splitFile := strings.Split(fullFile, "---\n")
		if strings.Contains(splitFile[0], "kind: Namespace") {
			splitFile[0] = strings.ReplaceAll(splitFile[0], "name: calyptia-core", fmt.Sprintf("name: %s", namespace))
		}
		fullFile = strings.Join(splitFile, "---\n")
	}
	if _, err := strconv.Atoi(namespace); err == nil {
		namespace = fmt.Sprintf(`"%s"`, namespace)
	}

	out := strings.ReplaceAll(fullFile, "namespace: calyptia-core", fmt.Sprintf("namespace: %s", namespace))
	return out
}

func solveNamespaceCreationForDelete(fullFile string, namespace string) string {
	if namespace == "" {
		splitFile := strings.Split(fullFile, "---\n")
		return strings.Join(splitFile[1:], "---\n")
	}
	if _, err := strconv.Atoi(namespace); err == nil {
		namespace = fmt.Sprintf(`"%s"`, namespace)
	}
	temp := strings.ReplaceAll(fullFile, "serviceAccountName: calyptia-core", fmt.Sprintf("serviceAccountName: %s", namespace))
	return strings.ReplaceAll(temp, "name: calyptia-core", fmt.Sprintf("name: %s", namespace))
}

func addImage(coreDockerImage, coreInstanceVersion, file string) (string, error) {
	if coreInstanceVersion != "" {
		const pattern string = `image:\s*ghcr\.io/calyptia/core-operator:[^\n\r]*`
		reImagePattern := regexp.MustCompile(pattern)
		match := reImagePattern.FindString(file)
		if match == "" {
			return "", errors.New("could not find image in manifest")
		}
		updatedMatch := fmt.Sprintf("image: %s:%s", coreDockerImage, coreInstanceVersion) // Remove '\n' at the end
		return reImagePattern.ReplaceAllString(file, updatedMatch), nil
	}
	return file, nil
}

func injectNamespace(s string, namespace string) string {
	if namespace == "" {
		namespace = "default"
	}
	if _, err := strconv.Atoi(namespace); err == nil {
		namespace = fmt.Sprintf(`"%s"`, namespace)
	}
	return strings.ReplaceAll(s, "namespace: calyptia-core", fmt.Sprintf("namespace: %s", namespace))
}

func injectArguments(s string, externalTrafficPolicyLocal bool) string {
	if externalTrafficPolicyLocal {
		fmt.Println("Enabling traffic policy LOCAL: ", EnableExternalTrafficPolicyLocal)
		return strings.ReplaceAll(s, "args: []", "args: ['"+EnableExternalTrafficPolicyLocal+"']")
	}
	return s
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

func installManifest(namespace, coreDockerImage, coreInstanceVersion string, createNamespace bool, externalTrafficPolicyLocal bool) (string, error) {
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
