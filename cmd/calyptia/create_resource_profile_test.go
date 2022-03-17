package main

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/calyptia/api/types"
)

func Test_newCmdCreateResourceProfile(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	spec := setupFile(t, "test-spec-*.json", []byte(`{}`))

	got := &bytes.Buffer{}
	mock := &ClientMock{
		AggregatorsFunc: func(ctx context.Context, projectID string, params types.AggregatorsParams) (types.Aggregators, error) {
			return types.Aggregators{
				Items: []types.Aggregator{{
					ID:   "want_aggregator",
					Name: "want_aggregator",
				}},
			}, nil
		},
		CreateResourceProfileFunc: func(ctx context.Context, aggregatorID string, payload types.CreateResourceProfile) (types.CreatedResourceProfile, error) {
			return types.CreatedResourceProfile{
				ID:        "want_resource_profile_id",
				CreatedAt: now,
			}, nil
		},
	}
	cmd := newCmdCreateResourceProfile(configWithMock(mock))
	cmd.SetOut(got)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{
		"--aggregator", "want_aggregator",
		"--name", "want_name",
		"--spec", spec.Name(),
	})

	wantEq(t, nil, cmd.Execute())
	wantEq(t, ""+
		"ID                       AGE\n"+
		"want_resource_profile_id Just now\n", got.String())

	calls := mock.CreateResourceProfileCalls()
	wantEq(t, 1, len(calls))

	call := calls[0]
	wantEq(t, "want_aggregator", call.AggregatorID)
	wantEq(t, "want_name", call.Payload.Name)
}
