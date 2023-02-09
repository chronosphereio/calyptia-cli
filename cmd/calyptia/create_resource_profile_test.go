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
		CoreInstancesFunc: func(ctx context.Context, projectID string, params types.CoreInstancesParams) (types.CoreInstances, error) {
			return types.CoreInstances{
				Items: []types.CoreInstance{{
					ID:   "want_core_instance",
					Name: "want_core_instance",
				}},
			}, nil
		},
		CreateResourceProfileFunc: func(ctx context.Context, CoreInstanceID string, payload types.CreateResourceProfile) (types.CreatedResourceProfile, error) {
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
		"--core-instance", "want_core_instance",
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
	wantEq(t, "want_core_instance", call.S)
	wantEq(t, "want_name", call.CreateResourceProfile.Name)
}
