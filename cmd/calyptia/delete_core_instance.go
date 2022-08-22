package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/calyptia/api/types"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"k8s.io/client-go/kubernetes"
)

func newCmdDeleteCoreInstance(config *config, testClientSet kubernetes.Interface) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "core_instance",
		Aliases: []string{"instance", "aggregator"},
		Short:   "Delete a core instance from either Kubernetes, Amazon EC2 (TODO), or Google Compute Engine (TODO)",
	}
	cmd.AddCommand(newCmdDeleteCoreInstanceK8s(config, nil))
	cmd.AddCommand(newCmdDeleteCoreInstanceOnAWS(config, nil))
	cmd.AddCommand(newCmdDeleteCoreInstanceOnGCP(config))
	return cmd
}

func newCmdDeleteCoreInstances(config *config) *cobra.Command {
	isNonInteractiveMode := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))

	var confirm bool
	cmd := &cobra.Command{
		Use:     "core_instances",
		Aliases: []string{"instances", "aggregators"},
		Short:   "Delete many core instances from the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				fmt.Print("Are you sure you want to delete all core instances? (y/N) ")

				confirm, err := readConfirm(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirm {
					cmd.Println("Aborted")
					return nil
				}
			}

			aa, err := config.cloud.Aggregators(config.ctx, config.projectID, types.AggregatorsParams{
				Last: ptr(uint64(200)),
			})
			if err != nil {
				return err
			}

			if len(aa.Items) == 0 {
				cmd.Println("No core instances left to delete")
				return nil
			}

			cmd.Printf("About to delete %d core instances\n", len(aa.Items))

			g := sync.WaitGroup{}

			var count uint
			for _, a := range aa.Items {
				g.Add(1)
				go func(a types.Aggregator) {
					defer g.Done()

					err := config.cloud.DeleteAggregator(config.ctx, a.ID)
					if err != nil {
						cmd.PrintErrf("Failed to delete core instance %q: %v\n", a.ID, err)
						return
					}

					count++
				}(a)
			}

			g.Wait()

			cmd.Printf("Deleted %d core instances\n", count)

			return nil
		},
	}

	fs := cmd.Flags()
	fs.BoolVarP(&confirm, "yes", "y", isNonInteractiveMode, "Confirm deletion of core instances")

	return cmd
}
