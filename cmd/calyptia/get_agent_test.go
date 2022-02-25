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

func Test_newCmdGetAgents(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got := &bytes.Buffer{}
		cmd := newCmdGetAgents(testConfig(nil))
		cmd.SetOut(got)

		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, "NAME TYPE VERSION STATUS AGE\n", got.String())
	})

	t.Run("empty_show_ids", func(t *testing.T) {
		got := &bytes.Buffer{}
		cmd := newCmdGetAgents(testConfig(nil))
		cmd.SetArgs([]string{"--show-ids"})
		cmd.SetOut(got)

		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, "ID NAME TYPE VERSION STATUS AGE\n", got.String())
	})

	t.Run("error", func(t *testing.T) {
		want := errors.New("internal error")
		cmd := newCmdGetAgents(testConfig(&ClientMock{
			AgentsFunc: func(ctx context.Context, projectID string, params types.AgentsParams) ([]types.Agent, error) {
				return nil, want
			},
		}))
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true

		got := cmd.Execute()
		wantEq(t, want, got)
	})

	t.Run("ok", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		want := []types.Agent{{
			ID:                 "agent_id_1",
			Name:               "name_1",
			Type:               types.AgentTypeFluentBit,
			Version:            "v1.8.6",
			LastMetricsAddedAt: now.Add(time.Second * -1),
			CreatedAt:          now.Add(time.Minute * -5),
		}, {
			ID:                 "agent_id_2",
			Name:               "name_2",
			Type:               types.AgentTypeFluentd,
			Version:            "v1.0.0",
			LastMetricsAddedAt: now.Add(time.Second * -30),
			CreatedAt:          now.Add(time.Minute * -10),
		}}
		got := &bytes.Buffer{}
		cmd := newCmdGetAgents(testConfig(&ClientMock{
			AgentsFunc: func(ctx context.Context, projectID string, params types.AgentsParams) ([]types.Agent, error) {
				wantNoEq(t, nil, params.Last)
				wantEq(t, uint64(2), *params.Last)
				return want, nil
			},
		}))
		cmd.SetArgs([]string{"--show-ids", "--last=2"})
		cmd.SetOut(got)

		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, ""+
			"ID         NAME   TYPE      VERSION STATUS AGE\n"+
			"agent_id_1 name_1 fluentbit v1.8.6  active 5 minutes\n"+
			"agent_id_2 name_2 fluentd   v1.0.0  active 10 minutes\n", got.String())

		t.Run("json", func(t *testing.T) {
			want, err := json.Marshal(want)
			wantEq(t, nil, err)

			got.Reset()
			cmd.SetArgs([]string{"--output-format=json"})

			err = cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, string(want)+"\n", got.String())
		})
	})
}

func Test_newCmdGetAgent(t *testing.T) {
	t.Run("no_arg", func(t *testing.T) {
		cmd := newCmdGetAgent(testConfig(nil))
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true

		got := cmd.Execute()
		wantErrMsg(t, "accepts 1 arg(s), received 0", got)
	})

	t.Run("error", func(t *testing.T) {
		want := errors.New("internal error")
		cmd := newCmdGetAgent(testConfig(&ClientMock{
			AgentFunc: func(ctx context.Context, agentID string) (types.Agent, error) {
				return types.Agent{}, want
			},
		}))
		cmd.SetArgs([]string{zeroUUID4})
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true

		got := cmd.Execute()
		wantEq(t, want, got)
	})

	t.Run("ok", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		want := types.Agent{
			ID:                 "agent_id",
			Name:               "name",
			Type:               types.AgentTypeFluentBit,
			Version:            "v1.8.6",
			RawConfig:          "raw_config",
			LastMetricsAddedAt: now.Add(time.Second * -1),
			CreatedAt:          now.Add(time.Minute * -5),
		}
		got := &bytes.Buffer{}
		cmd := newCmdGetAgent(testConfig(&ClientMock{
			AgentFunc: func(ctx context.Context, agentID string) (types.Agent, error) {
				return want, nil
			},
		}))
		cmd.SetArgs([]string{zeroUUID4, "--output-format=table"})
		cmd.SetOut(got)

		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, ""+
			"NAME TYPE      VERSION STATUS AGE\n"+
			"name fluentbit v1.8.6  active 5 minutes\n", got.String())

		t.Run("show_ids", func(t *testing.T) {
			got.Reset()
			cmd.SetArgs([]string{zeroUUID4, "--show-ids"})

			err := cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, ""+
				"ID       NAME TYPE      VERSION STATUS AGE\n"+
				"agent_id name fluentbit v1.8.6  active 5 minutes\n", got.String())
		})

		t.Run("only_config", func(t *testing.T) {
			got.Reset()
			cmd.SetArgs([]string{zeroUUID4, "--only-config"})

			err := cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, "raw_config\n", got.String())
		})

		t.Run("json", func(t *testing.T) {
			want, err := json.Marshal(want)
			wantEq(t, nil, err)

			got.Reset()
			// FIXME: Must override --only-config option back to false.
			cmd.SetArgs([]string{zeroUUID4, "--only-config=false", "--output-format=json"})

			err = cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, string(want)+"\n", got.String())
		})
	})
}
