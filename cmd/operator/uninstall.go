package operator

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	kubectl "k8s.io/kubectl/pkg/cmd"
	"os"
)

func NewCmdUninstall() *cobra.Command {

	// Create a new default kubectl command and retrieve its flags
	kubectlCmd := kubectl.NewDefaultKubectlCommand()
	kubectlFlags := kubectlCmd.Flags()

	cmd := &cobra.Command{
		Use:     "operator",
		Aliases: []string{"opr"},
		Short:   "Uninstall operator components",
		RunE: func(cmd *cobra.Command, args []string) error {
			kctl := newKubectlCmd()
			namespace := cmd.Flag("namespace").Value.String()

			yaml, err := prepareUninstallManifest(namespace)
			if err != nil {
				return err
			}

			kctl.SetArgs([]string{"delete", "-f", yaml})

			err = kctl.Execute()
			if err != nil {
				return err
			}
			defer os.RemoveAll(yaml)

			cmd.Printf("Calyptia Operator uninstalled successfully.\n")
			return nil
		},
	}

	kubectlFlags.VisitAll(func(flag *pflag.Flag) {
		if flag.Name == "log-flush-frequency" || flag.Name == "version" {
			return
		}
		cmd.PersistentFlags().AddFlag(flag)
	})
	return cmd
}

func prepareUninstallManifest(namespace string) (string, error) {
	file, err := manifest.ReadFile("manifest.yaml")
	if err != nil {
		return "", err
	}

	fullFile := string(file)

	solveNamespace := solveNamespaceCreation(false, fullFile, namespace)

	withNamespace := injectNamespace(solveNamespace, namespace)

	dir, err := os.MkdirTemp("", "calyptia-operator")
	if err != nil {
		return "", err
	}

	sysFile, err := os.CreateTemp(dir, "operator_*.yaml")
	if err != nil {
		return "", err
	}
	defer sysFile.Close()

	_, err = sysFile.WriteString(withNamespace)
	if err != nil {
		return "", err
	}

	return sysFile.Name(), nil
}
