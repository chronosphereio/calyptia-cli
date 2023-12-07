package operator

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/calyptia/core-images-index/go-index"

	semver "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
)

const (
	defaultWaitTimeout = time.Second * 30
)

var (
	verbose bool
)

func NewCmdUpdate() *cobra.Command {
	loadingRules = clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	cmd := &cobra.Command{
		Use:     "operator",
		Aliases: []string{"opr"},
		Short:   "Update core operator",
		Annotations: map[string]string{ //needed to identify the command
			"name": "update",
		},

		PreRunE: func(cmd *cobra.Command, args []string) error {
			if coreInstanceVersion == "" {
				return nil
			}
			if !strings.HasPrefix(coreInstanceVersion, "v") {
				coreInstanceVersion = fmt.Sprintf("v%s", coreInstanceVersion)
			}
			if _, err := semver.NewSemver(coreInstanceVersion); err != nil {
				return err
			}

			operatorIndex, err := index.NewOperator()
			if err != nil {
				return err
			}

			_, err = operatorIndex.Match(cmd.Context(), coreInstanceVersion)
			if err != nil {
				return fmt.Errorf("core-operator image tag %s is not available", coreInstanceVersion)
			}

			return nil
		},
		RunE: InstallOperator,
	}

	fs := cmd.Flags()

	fs.BoolVar(&waitReady, "wait", false, "Wait for the core instance to be ready before returning")
	fs.DurationVar(&waitTimeout, "timeout", defaultWaitTimeout, "Wait timeout")
	fs.BoolVar(&verbose, "verbose", false, "Print verbose command output")
	fs.StringVar(&coreInstanceVersion, "version", "", "Core instance version")
	_ = cmd.Flags().MarkHidden("image")
	clientcmd.BindOverrideFlags(configOverrides, fs, clientcmd.RecommendedConfigOverrideFlags("kube-"))

	return cmd
}
