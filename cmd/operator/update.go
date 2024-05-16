package operator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/calyptia/cli/cmd/utils"
	"github.com/calyptia/cli/confirm"
	"github.com/calyptia/core-images-index/go-index"

	semver "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"

	"github.com/calyptia/cli/k8s"
)

const (
	defaultWaitTimeout = time.Second * 30
)

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

			operatorIndex, err := index.NewOperator()
			if err != nil {
				return err
			}

			laststr, _ := operatorIndex.Last(cmd.Context())
			last, _ := semver.NewSemver(laststr)
			current, _ := semver.NewSemver(utils.DefaultCoreOperatorDockerImageTag)

			if current.LessThan(last) {
				cmd.Printf("Warning: Current version %s is less than the latest version %s\n", current.String(), last.String())
				cmd.Printf("To ensure that the operator functions as intended please install latest version of the CLI, and run %q\n", "calyptia update operator")
				cmd.Printf("Do you want to proceed with update? (y/N)\n")
				confirmed, err := confirm.Read(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					return fmt.Errorf("Aborted")
				}

			}

			if coreOperatorVersion == "" {
				return nil
			}
			if !strings.HasPrefix(coreOperatorVersion, "v") {
				coreOperatorVersion = fmt.Sprintf("v%s", coreOperatorVersion)
			}
			if _, err := semver.NewSemver(coreOperatorVersion); err != nil {
				return err
			}

			_, err = operatorIndex.Match(cmd.Context(), coreOperatorVersion)
			if err != nil {
				return fmt.Errorf("core-operator image tag %s is not available", coreOperatorVersion)
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

			if coreOperatorVersion == "" {
				coreOperatorVersion = utils.DefaultCoreOperatorDockerImageTag
			}

			manifest, err := installManifest(namespace, utils.DefaultCoreOperatorDockerImage, coreOperatorVersion, k8serrors.IsNotFound(err), externalTrafficPolicyLocal)
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
				fmt.Printf("Waiting for core operator manager to be updated...\n")
				err = k.WaitReady(context.Background(), namespace, deployment, false, waitTimeout)
				if err != nil {
					return err
				}
				fmt.Printf("Core operator manager is ready. Update took %s\n", time.Since(start))
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
