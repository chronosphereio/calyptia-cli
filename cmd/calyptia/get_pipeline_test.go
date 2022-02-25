package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/calyptia/api/types"
)

func Test_newCmdGetPipelines(t *testing.T) {
	t.Run("no_arg", func(t *testing.T) {
		cmd := newCmdGetPipelines(configWithMock(nil))
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		got := cmd.Execute()
		wantErrMsg(t, `required flag(s) "aggregator" not set`, got)
	})

	t.Run("error", func(t *testing.T) {
		want := errors.New("internal error")
		cmd := newCmdGetPipelines(configWithMock(&ClientMock{
			PipelinesFunc: func(ctx context.Context, aggregatorID string, params types.PipelinesParams) ([]types.Pipeline, error) {
				return nil, want
			},
		}))
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"--aggregator=" + zeroUUID4})
		got := cmd.Execute()
		wantEq(t, want, got)
	})

	t.Run("ok", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		want := []types.Pipeline{{
			ID:            "pipeline_id_1",
			Name:          "name_1",
			ReplicasCount: 4,
			Status: types.PipelineStatus{
				Status: types.PipelineStatusStarting,
			},
			CreatedAt: now.Add(time.Minute * -4),
		}, {
			ID:            "pipeline_id_2",
			Name:          "name_2",
			ReplicasCount: 5,
			Status: types.PipelineStatus{
				Status: types.PipelineStatusStarted,
			},
			CreatedAt: now.Add(time.Minute * -3),
		}}
		got := &bytes.Buffer{}
		cmd := newCmdGetPipelines(configWithMock(&ClientMock{
			PipelinesFunc: func(ctx context.Context, aggregatorID string, params types.PipelinesParams) ([]types.Pipeline, error) {
				wantNoEq(t, nil, params.Last)
				wantEq(t, uint64(2), *params.Last)
				return want, nil
			},
		}))
		cmd.SetOutput(got)
		cmd.SetArgs([]string{"--aggregator=" + zeroUUID4, "--last=2"})

		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, ""+
			"NAME   REPLICAS STATUS   AGE\n"+
			"name_1 4        STARTING 4 minutes\n"+
			"name_2 5        STARTED  3 minutes\n", got.String())

		t.Run("show_ids", func(t *testing.T) {
			got.Reset()
			cmd.SetArgs([]string{"--aggregator=" + zeroUUID4, "--show-ids"})

			err := cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, ""+
				"ID            NAME   REPLICAS STATUS   AGE\n"+
				"pipeline_id_1 name_1 4        STARTING 4 minutes\n"+
				"pipeline_id_2 name_2 5        STARTED  3 minutes\n", got.String())
		})

		t.Run("json", func(t *testing.T) {
			got.Reset()

			want, err := json.Marshal(want)
			wantEq(t, nil, err)

			cmd.SetArgs([]string{"--aggregator=" + zeroUUID4, "--output-format=json"})

			err = cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, string(want)+"\n", got.String())
		})
	})
}
