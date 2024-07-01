package operator

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/calyptia/cli/coreversions"
	"github.com/calyptia/cli/k8s"
	"github.com/calyptia/core-images-index/go-index"
)

const defaultWaitTimeout = time.Second * 30

func NewCmdUpdate() *cobra.Command {
	var (
		coreOperatorVersion        string
		waitReady                  bool
		waitTimeout                time.Duration
		verbose                    bool
		externalTrafficPolicyLocal bool
	)

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	cmd := &cobra.Command{
		Use:     "operator",
		Aliases: []string{"opr"},
		Short:   "Update core operator",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if coreOperatorVersion == "" {
				return nil
			}
			if !strings.HasPrefix(coreOperatorVersion, "v") {
				coreOperatorVersion = fmt.Sprintf("v%s", coreOperatorVersion)
			}
			if _, err := version.NewSemver(coreOperatorVersion); err != nil {
				return err
			}

			operatorIndex, err := index.NewOperator()
			if err != nil {
				return err
			}

			_, err = operatorIndex.Match(cmd.Context(), coreOperatorVersion)
			if err != nil {
				return fmt.Errorf("core-operator image tag %s is not available", coreOperatorVersion)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

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
			if err != nil && !kerrors.IsNotFound(err) {
				return err
			}

			if coreOperatorVersion == "" {
				coreOperatorVersion = coreversions.DefaultCoreOperatorDockerImageTag
			}

			manifest, err := installManifest(namespace, coreversions.DefaultCoreOperatorDockerImage, coreOperatorVersion, kerrors.IsNotFound(err), externalTrafficPolicyLocal)
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
				cmd.Printf("Waiting for core operator manager to be updated...\n")
				err = k.WaitReady(ctx, namespace, deployment, false, waitTimeout)
				if err != nil {
					return err
				}
				cmd.Printf("Core operator manager is ready. Update took %s\n", time.Since(start))
			}

			cmd.Printf("Core operator manager successfully updated to version %s\n", coreOperatorVersion)
			return nil
		},
	}

	fs := cmd.Flags()

	fs.BoolVar(&waitReady, "wait", false, "Wait for the core instance to be ready before returning")
	fs.DurationVar(&waitTimeout, "timeout", defaultWaitTimeout, "Wait timeout")
	fs.BoolVar(&verbose, "verbose", false, "Print verbose command output")
	fs.StringVar(&coreOperatorVersion, "version", "", "Core instance version")
	fs.BoolVar(&externalTrafficPolicyLocal, "external-traffic-policy-local", false, "Set ExternalTrafficPolicy to local for all services used by core operator pipelines.")
	_ = cmd.Flags().MarkHidden("image")
	clientcmd.BindOverrideFlags(configOverrides, fs, clientcmd.RecommendedConfigOverrideFlags("kube-"))

	return cmd
}
