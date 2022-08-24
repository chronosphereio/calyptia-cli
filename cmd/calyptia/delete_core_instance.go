package main

import (
	"fmt"
	"os"
	"strings"
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
	cmd.AddCommand(
		newCmdDeleteCoreInstanceK8s(config, nil),
		newCmdDeleteCoreInstanceOnAWS(config, nil),
		newCmdDeleteCoreInstanceOnGCP(config),
	)
	return cmd
}

func newCmdDeleteCoreInstances(config *config) *cobra.Command {
	var confirmed bool

	cmd := &cobra.Command{
		Use:   "core_instances",
		Short: "Delete many core instances from project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			aa, err := config.cloud.Aggregators(ctx, config.projectID, types.AggregatorsParams{
				Last: ptr(uint(0)),
			})
			if err != nil {
				return fmt.Errorf("could not prefetch core instances to delete: %w", err)
			}

			if len(aa.Items) == 0 {
				cmd.Println("No core instances to delete")
				return nil
			}

			if !confirmed {
				cmd.Printf("You are about to delete:\n\n%s\n\nAre you sure you want to delete all of them? (y/N) ", strings.Join(aggregatorsKeys(aa.Items), "\n"))
				confirmed, err := readConfirm(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					cmd.Println("Aborted")
					return nil
				}
			}

			g := sync.WaitGroup{}

			var count uint
			for _, a := range aa.Items {
				a := a
				g.Add(1)
				go func() {
					defer g.Done()

					err := config.cloud.DeleteAggregator(ctx, a.ID)
					if err != nil {
						cmd.PrintErrf("Error: could not delete core instance with id %q: %v\n", a.ID, err)
						return
					}

					count++
				}()
			}

			g.Wait()

			cmd.Printf("Successfully deleted %d core instances\n", count)

			return nil
		},
	}

	isNonInteractive := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", isNonInteractive, "Confirm deletion")

	return cmd
}
