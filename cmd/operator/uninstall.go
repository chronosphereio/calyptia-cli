package operator

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/calyptia/cli/k8s"
)

func NewCmdUninstall() *cobra.Command {
	// Create a new default kubectl command and retrieve its flags
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	cmd := &cobra.Command{
		Use:     "operator",
		Aliases: []string{"opr"},
		Short:   "Uninstall operator components",
		RunE: func(cmd *cobra.Command, args []string) error {
			kctl := newKubectlCmd()
			namespace := cmd.Flag("kube-namespace").Value.String()
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

			exists, err := k.CheckOperatorInstalled(cmd.Context(), namespace)
			if !exists || err != nil {
				return fmt.Errorf("operator not installed in the namespace %s %s", namespace, err.Error())
			}

			// remove all pipelines
			kctl.SetArgs([]string{"delete", "pipeline", "-A", "--all"})
			err = kctl.Execute()
			if err != nil {
				return err
			}

			// remove all ingestchecks
			kctl.SetArgs([]string{"delete", "ingestcheck", "-A", "--all"})
			err = kctl.Execute()
			if err != nil {
				return err
			}

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

			// when each Pipeline is create we ensure that there are appropriate RBAC setting for each. 
			// This will ensure that the respective ClusterRole, ClusterRoleBinding and ServiceAccount get wiped 
			if err := k.PurgeLeftoverRBAC(cmd.Context()); err != nil {
				return err
			}

			cmd.Printf("Calyptia Operator uninstalled successfully.\n")
			return nil
		},
	}
	fs := cmd.Flags()
	clientcmd.BindOverrideFlags(configOverrides, fs, clientcmd.RecommendedConfigOverrideFlags("kube-"))
	return cmd
}

func prepareUninstallManifest(namespace string) (string, error) {
	file, err := f.ReadFile(manifestFile)
	if err != nil {
		return "", err
	}

	fullFile := string(file)
	var isNamespace bool
	if namespace != "" {
		isNamespace = true
	}

	solveNamespace := solveNamespaceCreation(isNamespace, fullFile, namespace)
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
