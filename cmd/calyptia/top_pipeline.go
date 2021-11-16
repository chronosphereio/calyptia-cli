package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/calyptia/cloud"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/term"
)

func newCmdTopPipeline(config *config) *cobra.Command {
	var start, interval time.Duration

	cmd := &cobra.Command{
		Use:               "pipeline key",
		Short:             "Display metrics from a pipeline",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completePipelines,
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineKey := args[0]
			pipelineID := pipelineKey
			if !validUUID(pipelineID) {
				pp, err := config.fetchAllPipelines()
				if err != nil {
					return err
				}

				pip, ok := findPipelineByName(pp, pipelineKey)
				if !ok {
					return fmt.Errorf("could not find pipeline %q", pipelineKey)
				}

				pipelineID = pip.ID
			}

			var pipeline cloud.AggregatorPipeline
			var metrics cloud.AgentMetrics

			g, gctx := errgroup.WithContext(config.ctx)
			g.Go(func() error {
				var err error
				pipeline, err = config.cloud.AggregatorPipeline(gctx, pipelineID)
				if err != nil {
					return fmt.Errorf("could not fetch pipeline: %w", err)
				}

				return nil
			})
			g.Go(func() error {
				var err error
				metrics, err = config.cloud.PipelineMetrics(gctx, pipelineID, start, interval)
				if err != nil {
					return fmt.Errorf("could not fetch pipeline metrics: %w", err)
				}

				return nil
			})
			if err := g.Wait(); err != nil {
				return err
			}

			_ = pipeline

			{
				fmt.Println(titleStyle.Render("Metrics"))

				if len(metrics.Measurements) == 0 {
					fmt.Println("No pipeline metrics to display")
				} else {
					tw := table.NewWriter()
					tw.Style().Options = table.OptionsNoBordersAndSeparators
					if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
						tw.SetAllowedRowLength(w)
					}

					for _, measurementName := range agentMeasurementNames(metrics.Measurements) {
						measurement := metrics.Measurements[measurementName]

						for _, pluginName := range pluginNames(measurement.Plugins) {
							// skip internal plugins.
							if strings.HasPrefix(pluginName, "calyptia.") || strings.HasPrefix(pluginName, "fluentbit_metrics.") {
								continue
							}

							plugin := measurement.Plugins[pluginName]
							values := fmtLatestMetrics(plugin.Metrics, interval)
							var value string
							if len(values) == 0 {
								value = "No data"
							} else {
								value = strings.Join(values, ", ")
							}

							tw.AppendRow(table.Row{fmt.Sprintf("%s (%s)", pluginName, measurementName), value})
						}
					}
					fmt.Println(tw.Render())
				}
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.DurationVar(&start, "start", time.Minute*-2, "Start time range")
	fs.DurationVar(&interval, "interval", time.Minute, "Interval rate")

	return cmd
}
