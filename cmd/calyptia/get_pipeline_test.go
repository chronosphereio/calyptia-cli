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
			PipelinesFunc: func(ctx context.Context, aggregatorID string, params types.PipelinesParams) (types.Pipelines, error) {
				return types.Pipelines{}, want
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
		want := types.Pipelines{
			Items: []types.Pipeline{{
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
			}},
		}
		got := &bytes.Buffer{}
		cmd := newCmdGetPipelines(configWithMock(&ClientMock{
			PipelinesFunc: func(ctx context.Context, aggregatorID string, params types.PipelinesParams) (types.Pipelines, error) {
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

			want, err := json.Marshal(want.Items)
			wantEq(t, nil, err)

			cmd.SetArgs([]string{"--aggregator=" + zeroUUID4, "--output-format=json"})

			err = cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, string(want)+"\n", got.String())
		})
	})
}

func Test_newCmdGetPipeline(t *testing.T) {
	t.Run("no_arg", func(t *testing.T) {
		cmd := newCmdGetPipeline(configWithMock(nil))
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		got := cmd.Execute()
		wantErrMsg(t, `accepts 1 arg(s), received 0`, got)
	})

	t.Run("error", func(t *testing.T) {
		t.Run("pipeline", func(t *testing.T) {
			want := errors.New("internal error")
			cmd := newCmdGetPipeline(configWithMock(&ClientMock{
				PipelineFunc: func(ctx context.Context, pipelineID string) (types.Pipeline, error) {
					return types.Pipeline{}, want
				},
			}))
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs([]string{zeroUUID4})
			got := cmd.Execute()
			wantEq(t, want, got)
		})

		t.Run("ports", func(t *testing.T) {
			want := errors.New("internal error")
			cmd := newCmdGetPipeline(configWithMock(&ClientMock{
				PipelinePortsFunc: func(ctx context.Context, pipelineID string, params types.PipelinePortsParams) (types.PipelinePorts, error) {
					return types.PipelinePorts{}, want
				},
			}))
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs([]string{zeroUUID4, "--include-endpoints"})
			got := cmd.Execute()
			wantEq(t, want, got)
		})

		t.Run("config_history", func(t *testing.T) {
			want := errors.New("internal error")
			cmd := newCmdGetPipeline(configWithMock(&ClientMock{
				PipelineConfigHistoryFunc: func(ctx context.Context, pipelineID string, params types.PipelineConfigHistoryParams) (types.PipelineConfigHistory, error) {
					return types.PipelineConfigHistory{}, want
				},
			}))
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs([]string{zeroUUID4, "--include-config-history"})
			got := cmd.Execute()
			wantEq(t, want, got)
		})

		t.Run("secrets", func(t *testing.T) {
			want := errors.New("internal error")
			cmd := newCmdGetPipeline(configWithMock(&ClientMock{
				PipelineSecretsFunc: func(ctx context.Context, pipelineID string, params types.PipelineSecretsParams) (types.PipelineSecrets, error) {
					return types.PipelineSecrets{}, want
				},
			}))
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs([]string{zeroUUID4, "--include-secrets"})
			got := cmd.Execute()
			wantEq(t, want, got)
		})
	})

	t.Run("ok", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		want := types.Pipeline{
			ID:            "pipeline_id",
			Name:          "name",
			ReplicasCount: 4,
			Status: types.PipelineStatus{
				Status: types.PipelineStatusNew,
			},
			CreatedAt: now.Add(-time.Minute),
		}
		got := &bytes.Buffer{}
		cmd := newCmdGetPipeline(configWithMock(&ClientMock{
			PipelineFunc: func(ctx context.Context, pipelineID string) (types.Pipeline, error) {
				wantEq(t, zeroUUID4, pipelineID)
				return want, nil
			},
			PipelinePortsFunc: func(ctx context.Context, pipelineID string, params types.PipelinePortsParams) (types.PipelinePorts, error) {
				return types.PipelinePorts{
					Items: []types.PipelinePort{{
						ID:           "port_id_1",
						Protocol:     "tcp",
						FrontendPort: 80,
						BackendPort:  81,
						Endpoint:     "endpoint_1",
						CreatedAt:    now.Add(-time.Minute),
					}, {
						ID:           "port_id_2",
						Protocol:     "udp",
						FrontendPort: 90,
						BackendPort:  91,
						Endpoint:     "endpoint_2",
						CreatedAt:    now.Add(time.Minute * -2),
					}},
				}, nil
			},
			PipelineConfigHistoryFunc: func(ctx context.Context, pipelineID string, params types.PipelineConfigHistoryParams) (types.PipelineConfigHistory, error) {
				return types.PipelineConfigHistory{
					Items: []types.PipelineConfig{{
						ID:        "config_id_1",
						CreatedAt: now.Add(-time.Minute),
					}, {
						ID:        "config_id_2",
						CreatedAt: now.Add(time.Minute * -2),
					}},
				}, nil
			},
			PipelineSecretsFunc: func(ctx context.Context, pipelineID string, params types.PipelineSecretsParams) (types.PipelineSecrets, error) {
				return types.PipelineSecrets{
					Items: []types.PipelineSecret{{
						ID:        "secret_id_1",
						Key:       "key_1",
						CreatedAt: now.Add(-time.Minute),
					}, {
						ID:        "secret_id_2",
						Key:       "key_2",
						CreatedAt: now.Add(time.Minute * -2),
					}},
				}, nil
			},
		}))
		cmd.SetOut(got)
		cmd.SetArgs([]string{zeroUUID4})

		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, ""+
			"NAME REPLICAS STATUS AGE\n"+
			"name 4        NEW    1 minute\n", got.String())

		t.Run("show_ids", func(t *testing.T) {
			got.Reset()

			cmd.SetArgs([]string{zeroUUID4, "--show-ids"})

			err := cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, ""+
				"ID          NAME REPLICAS STATUS AGE\n"+
				"pipeline_id name 4        NEW    1 minute\n", got.String())
		})

		t.Run("json", func(t *testing.T) {
			got.Reset()

			want, err := json.Marshal(want)
			wantEq(t, nil, err)

			cmd.SetArgs([]string{zeroUUID4, "--output-format=json"})

			err = cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, string(want)+"\n", got.String())
		})

		t.Run("include_endpoints", func(t *testing.T) {
			got.Reset()

			// Note: Must override --output-format
			// and --show-ids options back to table.
			cmd.SetArgs([]string{zeroUUID4, "--output-format=table", "--show-ids=false", "--include-endpoints"})

			err := cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, ""+
				"NAME REPLICAS STATUS AGE\n"+
				"name 4        NEW    1 minute\n"+
				"\n"+
				"## Endpoints\n"+
				"PROTOCOL FRONTEND-PORT BACKEND-PORT ENDPOINT   AGE\n"+
				"tcp      80            81           endpoint_1 1 minute\n"+
				"udp      90            91           endpoint_2 2 minutes\n", got.String())
		})

		t.Run("include_config_history", func(t *testing.T) {
			got.Reset()

			// Note: Must override --output-format,
			// --show-ids and --include-endpoints options back to table.
			cmd.SetArgs([]string{zeroUUID4, "--output-format=table", "--show-ids=false", "--include-endpoints=false", "--include-config-history"})

			err := cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, ""+
				"NAME REPLICAS STATUS AGE\n"+
				"name 4        NEW    1 minute\n"+
				"\n"+
				"## Configuration History\n"+
				"ID          AGE\n"+
				"config_id_1 1 minute\n"+
				"config_id_2 2 minutes\n", got.String())
		})

		t.Run("include_secrets", func(t *testing.T) {
			got.Reset()

			// Note: Must override --output-format,
			// --show-ids, --include-endpoints and --include-config-history
			// options back to table.
			cmd.SetArgs([]string{zeroUUID4, "--output-format=table", "--show-ids=false", "--include-endpoints=false", "--include-config-history=false", "--include-secrets"})

			err := cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, ""+
				"NAME REPLICAS STATUS AGE\n"+
				"name 4        NEW    1 minute\n"+
				"\n"+
				"## Secrets\n"+
				"KEY   AGE\n"+
				"key_1 1 minute\n"+
				"key_2 2 minutes\n", got.String())
		})

		t.Run("include_all", func(t *testing.T) {
			got.Reset()

			// Note: Must override --output-format,
			// --show-ids, --include-endpoints and --include-config-history
			// options back to table.
			cmd.SetArgs([]string{zeroUUID4, "--output-format=table", "--show-ids=true", "--include-endpoints=true", "--include-config-history=true", "--include-secrets=true"})

			err := cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, ""+
				"ID          NAME REPLICAS STATUS AGE\n"+
				"pipeline_id name 4        NEW    1 minute\n"+
				"\n"+
				"## Endpoints\n"+
				"ID        PROTOCOL FRONTEND-PORT BACKEND-PORT ENDPOINT   AGE\n"+
				"port_id_1 tcp      80            81           endpoint_1 1 minute\n"+
				"port_id_2 udp      90            91           endpoint_2 2 minutes\n"+
				"\n"+
				"## Configuration History\n"+
				"ID          AGE\n"+
				"config_id_1 1 minute\n"+
				"config_id_2 2 minutes\n"+
				"\n"+
				"## Secrets\n"+
				"ID          KEY   AGE\n"+
				"secret_id_1 key_1 1 minute\n"+
				"secret_id_2 key_2 2 minutes\n", got.String())
		})
	})
}
