package operator

import (
	"context"
	"github.com/calyptia/cli/k8s"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	kubectl "k8s.io/kubectl/pkg/cmd"
	"os"
	"path/filepath"
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

			version, err := k.CheckOperatorVersion(context.Background())
			if err != nil {
				return err
			}

			yaml, err := prepareUninstallManifest(version, namespace)
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

func prepareUninstallManifest(version string, namespace string) (string, error) {
	file, err := k8s.GetOperatorManifest(version)
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

	fileLocation := filepath.Join(dir, "operator.yaml")
	err = os.WriteFile(fileLocation, []byte(withNamespace), 0644)
	return fileLocation, err
}
