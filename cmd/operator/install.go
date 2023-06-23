package operator

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/calyptia/cli/cmd/utils"
	"github.com/calyptia/cli/k8s"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/component-base/logs"
	kubectl "k8s.io/kubectl/pkg/cmd"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

			yaml, err := prepareManifest(coreDockerImage, coreInstanceVersion, namespace)
			if err != nil {
				return err
			}

			kctl.SetArgs([]string{"apply", "-f", yaml})
			//get original flags from kubectl

			err = kctl.Execute()
			if err != nil {
				return err
			}
			os.RemoveAll(yaml)

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

func prepareManifest(coreDockerImage, coreInstanceVersion, namespace string) (string, error) {
	file, err := k8s.GetOperatorManifest(coreInstanceVersion)
	if err != nil {
		return "", err
	}

	fullFile := string(file)

	splitFile := strings.Split(fullFile, "---\n")
	withoutNamespaceCreation := strings.Join(splitFile[1:], "---\n")

	withNamespace := addNamespace(withoutNamespaceCreation, namespace)

	withImage, err := addImage(coreDockerImage, coreInstanceVersion, withNamespace)
	if err != nil {
		return "", err
	}

	dir, err := os.MkdirTemp("", "calyptia-operator")
	if err != nil {
		return "", err
	}

	fileLocation := filepath.Join(dir, "operator.yaml")
	err = os.WriteFile(fileLocation, []byte(withImage), 0644)
	return fileLocation, err
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

func addNamespace(s string, namespace string) string {
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
