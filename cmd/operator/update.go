package operator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/calyptia/cli/cmd/utils"
	"github.com/calyptia/core-images-index/go-index"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/calyptia/cli/k8s"
	semver "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
)

func NewCmdUpdate() *cobra.Command {
	var coreOperatorVersion string
	var waitReady bool
	var verbose bool

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	cmd := &cobra.Command{
		Use:     "operator",
		Aliases: []string{"opr"},
		Short:   "Update core operator",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if !strings.HasPrefix(coreOperatorVersion, "v") {
				coreOperatorVersion = fmt.Sprintf("v%s", coreOperatorVersion)
			}
			if _, err := semver.NewSemver(coreOperatorVersion); err != nil {
				return err
			}

			containerIndex, err := index.NewContainer()
			if err != nil {
				return err
			}

			indices, err := containerIndex.All(cmd.Context())
			if err != nil {
				return err
			}

			var found bool
			for _, index := range indices {
				found = index == coreOperatorVersion
				if found {
					break
				}
			}

			if !found {
				return fmt.Errorf("%s version is not available", coreOperatorVersion)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var namespace string
			if configOverrides.Context.Namespace == "" {
				configOverrides.Context.Namespace = apiv1.NamespaceDefault
			}

			kubeNamespaceFlag := cmd.Flag("kube-namespace")
			if kubeNamespaceFlag != nil {
				namespace = kubeNamespaceFlag.Value.String()
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
				Namespace: configOverrides.Context.Namespace,
			}
			_, err = k.GetNamespace(cmd.Context(), namespace)
			if err != nil && !k8serrors.IsNotFound(err) {
				return err
			}

			label := fmt.Sprintf("%s=%s,%s=%s,%s=%s", k8s.LabelComponent, "manager", k8s.LabelCreatedBy, "operator", k8s.LabelInstance, "controller-manager")
			cmd.Printf("Waiting for core-operator to update...\n")
			if err := k.UpdateOperatorDeploymentByLabel(cmd.Context(), label, fmt.Sprintf("%s:%s", utils.DefaultCoreOperatorDockerImage, coreOperatorVersion), verbose); err != nil {
				if !verbose {
					return fmt.Errorf("could not update core-operator to version %s for extra details use --verbose flag", coreOperatorVersion)
				}
				return fmt.Errorf("could not update core-operator to version %s, \n%s", coreOperatorVersion, err)

			}

			cmd.Printf("Core operator manager successfully updated to version %s\n", coreOperatorVersion)

			return nil
		},
	}

	fs := cmd.Flags()

	fs.BoolVar(&waitReady, "wait", false, "Wait for the core instance to be ready before returning")
	fs.BoolVar(&verbose, "verbose", false, "Print verbose command output")
	fs.StringVar(&coreOperatorVersion, "version", utils.DefaultCoreOperatorDockerImageTag, "Core instance version")
	_ = cmd.Flags().MarkHidden("image")
	clientcmd.BindOverrideFlags(configOverrides, fs, clientcmd.RecommendedConfigOverrideFlags("kube-"))

	return cmd
}
