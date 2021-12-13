package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func newCmdTopPipeline(config *config) *cobra.Command {
	var start, interval time.Duration

	cmd := &cobra.Command{
		Use:               "pipeline PIPELINE",
		Short:             "Display metrics from a pipeline",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completePipelines,
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineKey := args[0]
			return tea.NewProgram(initialPipelineModel(pipelineKey), tea.WithAltScreen()).Start()
		},
	}

	fs := cmd.Flags()
	fs.DurationVar(&start, "start", time.Minute*-3, "Start time range")
	fs.DurationVar(&interval, "interval", time.Minute, "Interval rate")

	return cmd
}

func NewPipelineModel(pipelineKey string) PipelineModel {
	return PipelineModel{
		pipelineKey: pipelineKey,
	}
}

type PipelineModel struct {
	pipelineKey string
}

func (m PipelineModel) Init() tea.Cmd {
	return nil
}

func (m PipelineModel) Update(msg tea.Msg) (PipelineModel, tea.Cmd) {
	return m, nil
}

func (m PipelineModel) View() string {
	return "TODO: top " + m.pipelineKey
}
