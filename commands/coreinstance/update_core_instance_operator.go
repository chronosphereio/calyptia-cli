package coreinstance

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/coreversions"
	"github.com/calyptia/cli/k8s"
	"github.com/calyptia/core-images-index/go-index"
)

func NewCmdUpdateCoreInstanceOperator(cfg *config.Config, testClientSet kubernetes.Interface) *cobra.Command {
	var newVersion, newName string
	var (
		disableClusterLogging                      bool
		enableClusterLogging                       bool
		noTLSVerify                                bool
		skipServiceCreation                        bool
		cloudProxy, httpProxy, httpsProxy, noProxy string
		metrics                                    bool
		verbose                                    bool
		waitTimeout                                time.Duration
	)
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	cmd := &cobra.Command{
		Use:               "operator CORE_INSTANCE",
		Aliases:           []string{"opr", "k8s"},
		Short:             "update a core instance operator",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cfg.Completer.CompleteCoreInstances,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if newVersion == "" {
				return nil
			}

			if !strings.HasPrefix(newVersion, "v") {
				newVersion = fmt.Sprintf("v%s", newVersion)
			}
			operatorIndex, err := index.NewOperator()
			if err != nil {
				return err
			}

			_, err = operatorIndex.Match(cmd.Context(), newVersion)
			if err != nil {
				return fmt.Errorf("core_instance image tag %s is not available", newVersion)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			coreInstanceKey := args[0]
			coreInstanceID, err := cfg.Completer.LoadCoreInstanceID(ctx, coreInstanceKey)
			if err != nil {
				return err
			}

			if coreInstanceKey == newName {
				return fmt.Errorf("cannot update core instance with the same name")
			}

			var opts cloudtypes.UpdateCoreInstance

			if newName != "" {
				opts.Name = &newName
			}

			if enableClusterLogging && disableClusterLogging {
				return fmt.Errorf("either --enable-cluster-logging or --disable-cluster-logging can be set")
			}

			if enableClusterLogging {
				opts.ClusterLogging = &enableClusterLogging
			} else if disableClusterLogging {
				disableClusterLogging = !disableClusterLogging
				opts.ClusterLogging = &disableClusterLogging
			}

			if skipServiceCreation {
				opts.SkipServiceCreation = &skipServiceCreation
			}

			err = cfg.Cloud.UpdateCoreInstance(ctx, coreInstanceID, opts)
			if err != nil {
				return fmt.Errorf("could not update core instance at calyptia cloud: %w", err)
			}

			if configOverrides.Context.Namespace == "" {
				configOverrides.Context.Namespace = apiv1.NamespaceDefault
			}

			if newVersion == "" {
				newVersion = coreversions.DefaultCoreOperatorFromCloudDockerImageTag
			}
			var clientSet kubernetes.Interface
			if testClientSet != nil {
				clientSet = testClientSet
			} else {
				kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
				kubeClientConfig, err := kubeConfig.ClientConfig()
				if err != nil {
					return err
				}

				clientSet, err = kubernetes.NewForConfig(kubeClientConfig)
				if err != nil {
					return err
				}

			}

			k8sClient := &k8s.Client{
				Interface:    clientSet,
				Namespace:    configOverrides.Context.Namespace,
				ProjectToken: cfg.ProjectToken,
				CloudBaseURL: cfg.BaseURL,
			}

			if err := k8sClient.EnsureOwnNamespace(ctx); err != nil {
				return fmt.Errorf("could not ensure kubernetes namespace exists: %w", err)
			}

			label := fmt.Sprintf("%s=%s", k8s.LabelInstance, coreInstanceKey)
			cmd.Println("Waiting for core-instance to update...")

			cmd.Println("METRICS", metrics)
			syncParams := k8s.UpdateCoreOperatorSync{
				Metrics:             metrics,
				SkipServiceCreation: skipServiceCreation,
				NoTLSVerify:         noTLSVerify,
				CloudProxy:          cloudProxy,
				HttpProxy:           httpProxy,
				HttpsProxy:          httpsProxy,
				NoProxy:             noProxy,
				Image:               newVersion,
			}
			if err := k8sClient.UpdateSyncDeploymentByLabel(ctx, label, syncParams, verbose, waitTimeout); err != nil {
				if !verbose {
					return fmt.Errorf("could not update core-instance to version %s for extra details use --verbose flag", newVersion)
				}
				return fmt.Errorf("could not update core-instance: to version %s %w", newVersion, err)
			}

			cmd.Printf("calyptia-core instance version updated to version %s\n", newVersion)

			return nil
		},
	}
	fs := cmd.Flags()
	fs.StringVar(&newVersion, "version", "", "New version of the calyptia-core instance")
	fs.StringVar(&newName, "name", "", "New core instance name")
	fs.BoolVar(&enableClusterLogging, "enable-cluster-logging", false, "Enable cluster logging functionality")
	fs.BoolVar(&disableClusterLogging, "disable-cluster-logging", false, "Disable cluster logging functionality")
	fs.BoolVar(&noTLSVerify, "no-tls-verify", false, "Disable TLS verification when connecting to Calyptia Cloud API.")
	fs.StringVar(&noProxy, "no-proxy", "", "http proxy to use on this core instance")
	fs.StringVar(&cloudProxy, "cloud-proxy", "", "proxy for cloud api client to use on this core instance")
	fs.StringVar(&httpProxy, "http-proxy", "", "no proxy to use on this core instance")
	fs.StringVar(&httpsProxy, "https-proxy", "", "https proxy to use on this core instance")
	fs.BoolVar(&verbose, "verbose", false, "Print verbose command output")
	fs.DurationVar(&waitTimeout, "timeout", time.Second*30, "Wait timeout")
	fs.BoolVar(&skipServiceCreation, "skip-service-creation", false, "Skip the creation of kubernetes services for any pipeline under this core instance.")
	fs.BoolVar(&metrics, "metrics", false, "flag to enable/disable core instance metrics")

	_ = cmd.RegisterFlagCompletionFunc("environment", cfg.Completer.CompleteEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("version", cfg.Completer.CompleteCoreOperatorVersion)
	clientcmd.BindOverrideFlags(configOverrides, fs, clientcmd.RecommendedConfigOverrideFlags("kube-"))
	return cmd
}
