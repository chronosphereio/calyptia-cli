package operator

import (
	"errors"
	"fmt"

	"github.com/calyptia/cli/cmd/utils"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/calyptia/cli/k8s"
	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
)

func NewCmdUpdate() *cobra.Command {
	var coreOperatorVersion string
	var waitReady bool

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	cmd := &cobra.Command{
		Use:     "operator",
		Aliases: []string{"opr"},
		Short:   "Update core operator",
		RunE: func(cmd *cobra.Command, args []string) error {
			var namespace string
			if configOverrides.Context.Namespace == "" {
				configOverrides.Context.Namespace = apiv1.NamespaceDefault
			}

			kubeNamespaceFlag := cmd.Flag("kube-namespace")
			if kubeNamespaceFlag != nil {
				namespace = kubeNamespaceFlag.Value.String()
			}

			// if namespace == "" {
			// 	namespace = apiv1.NamespaceDefault
			// }

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
				Namespace: configOverrides.Context.Namespace,
			}
			_, err = k.GetNamespace(cmd.Context(), namespace)
			if err != nil && !k8serrors.IsNotFound(err) {
				return err
			}

			label := fmt.Sprintf("%s=%s,%s=%s,%s=%s", k8s.LabelComponent, "manager", k8s.LabelCreatedBy, "operator", k8s.LabelInstance, "controller-manager")
			if err := k.UpdateOperatorDeploymentByLabel(cmd.Context(), label, fmt.Sprintf("%s:%s", utils.DefaultCoreOperatorDockerImage, coreOperatorVersion)); err != nil {
				return fmt.Errorf("could not update kubernetes deployment: %w", err)
			}

			cmd.Printf("Core operator manager successfully installed.\n")
			return nil
		},
	}

	fs := cmd.Flags()

	fs.BoolVar(&waitReady, "wait", false, "Wait for the core instance to be ready before returning")
	fs.StringVar(&coreOperatorVersion, "version", utils.DefaultCoreOperatorDockerImageTag, "Core instance version")
	_ = cmd.Flags().MarkHidden("image")
	clientcmd.BindOverrideFlags(configOverrides, fs, clientcmd.RecommendedConfigOverrideFlags("kube-"))

	return cmd
}
