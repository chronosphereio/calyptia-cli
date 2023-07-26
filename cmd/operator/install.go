package operator

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/calyptia/cli/cmd/utils"
	operatormanifest "github.com/calyptia/cli/operator-manifest"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"os"
	"regexp"
	"strings"

	"github.com/calyptia/cli/k8s"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/component-base/logs"
	kubectl "k8s.io/kubectl/pkg/cmd"
)

func NewCmdInstall() *cobra.Command {
	var coreInstanceVersion string
	var coreDockerImage string
	var waitReady bool

	// Create a new default kubectl command and retrieve its flags
	kubectlCmd := kubectl.NewDefaultKubectlCommand()
	kubectlFlags := kubectlCmd.Flags()

	cmd := &cobra.Command{
		Use:     "operator",
		Aliases: []string{"opr"},
		Short:   "Setup a new core operator instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			kctl := newKubectlCmd()
			namespace := cmd.Flag("namespace").Value.String()
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

			loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
			configOverrides := &clientcmd.ConfigOverrides{}
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
			}
			_, err = k.GetNamespace(context.Background(), namespace)
			if err != nil && !k8serrors.IsNotFound(err) {
				return err
			}
			yaml, err := prepareInstallManifest(coreDockerImage, coreInstanceVersion, namespace, k8serrors.IsNotFound(err))
			if err != nil {
				return err
			}
			
			kctl.SetArgs([]string{"apply", "-k", yaml})
			//get original flags from kubectl

			err = kctl.Execute()
			if err != nil {
				return err
			}
			defer os.RemoveAll(yaml)

			cmd.Printf("Core operator manager successfully installed.\n")
			return nil
		},
	}

	kubectlFlags.VisitAll(func(flag *pflag.Flag) {
		if flag.Name == "log-flush-frequency" || flag.Name == "version" {
			return
		}
		cmd.PersistentFlags().AddFlag(flag)
	})

	fs := cmd.Flags()

	fs.BoolVar(&waitReady, "wait", false, "Wait for the core instance to be ready before returning")
	fs.StringVar(&coreInstanceVersion, "version", utils.DefaultCoreOperatorDockerImageTag, "Core instance version")
	fs.StringVar(&coreDockerImage, "image", utils.DefaultCoreOperatorDockerImage, "Calyptia core manager docker image to use (fully composed docker image).")
	_ = cmd.Flags().MarkHidden("image")
	return cmd
}

func prepareInstallManifest(coreDockerImage, coreInstanceVersion, namespace string, createNamespace bool) (string, error) {
	tmpdir := os.TempDir()

	manifestNames := operatormanifest.AssetNames()
	for _, name := range manifestNames {
		dir := filepath.Dir(name)
		if dir != "." {
			if err := os.MkdirAll(filepath.Join(tmpdir, dir), 0700); err != nil {
				return "", err
			}
		}

		f, err := os.Create(filepath.Join(tmpdir, name))
		if err != nil {
			return "", nil
		}

		content, _ := operatormanifest.Asset(name)
		if _, err := f.Write(content); err != nil {
			return "", nil
		}

	}
	return filepath.Join(tmpdir, "config", "default"), nil
}

func solveNamespaceCreation(createNamespace bool, fullFile string, namespace string) string {
	if !createNamespace {
		splitFile := strings.Split(fullFile, "---\n")
		return strings.Join(splitFile[1:], "---\n")
	}
	if _, err := strconv.Atoi(namespace); err == nil {
		namespace = fmt.Sprintf(`"%s"`, namespace)
	}
	return strings.ReplaceAll(fullFile, "name: calyptia-core", fmt.Sprintf("name: %s", namespace))
}

func addImage(coreDockerImage, coreInstanceVersion, file string) (string, error) {
	const pattern string = `image:\s*ghcr.io/calyptia/core-operator:[^\n\r]*`
	reImagePattern := regexp.MustCompile(pattern)
	match := reImagePattern.FindString(file)
	if match == "" {
		return "", errors.New("could not find image in manifest")
	}
	updatedMatch := fmt.Sprintf("image: %s:%s", coreDockerImage, coreInstanceVersion) // Remove '\n' at the end
	return reImagePattern.ReplaceAllString(file, updatedMatch), nil
}

func injectNamespace(s string, namespace string) string {
	if _, err := strconv.Atoi(namespace); err == nil {
		namespace = fmt.Sprintf(`"%s"`, namespace)
	}
	return strings.ReplaceAll(s, "namespace: calyptia-core", fmt.Sprintf("namespace: %s", namespace))
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
