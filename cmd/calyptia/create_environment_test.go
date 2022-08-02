package main

import (
	"bytes"
	"context"
	"github.com/calyptia/api/types"
	"testing"
)

func TestNewEnvironment(t *testing.T) {

	t.Run("ok", func(t *testing.T) {
		got := &bytes.Buffer{}
		cmd := newCmdCreateEnvironment(configWithMock(&ClientMock{
			CreateEnvironmentFunc: func(ctx context.Context, projectID string, payload types.CreateEnvironment) (types.CreatedEnvironment, error) {
				return types.CreatedEnvironment{
					ID: "999be8ae-36b6-439d-81dc-e6fd137b0ffe",
				}, nil
			},
		}))
		cmd.SetOut(got)
		cmd.SetArgs([]string{"test-environment"})
		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, "Created environment ID: 999be8ae-36b6-439d-81dc-e6fd137b0ffe Name: test-environment\n", got.String())
	})
}
