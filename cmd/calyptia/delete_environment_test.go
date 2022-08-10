package main

import (
	"bytes"
	"context"
	"testing"

	cloud "github.com/calyptia/api/types"
)

func TestDeleteEnvironment(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		got := &bytes.Buffer{}
		cmd := newCmdDeleteEnvironment(configWithMock(&ClientMock{
			DeleteEnvironmentFunc: func(ctx context.Context, environmentID string) error {
				return nil
			},
			EnvironmentsFunc: func(ctx context.Context, projectID string, params cloud.EnvironmentsParams) (cloud.Environments, error) {
				return cloud.Environments{
					Items: []cloud.Environment{{ID: "999be8ae-36b6-439d-81dc-e6fd137b0ffe", Name: "test-environment"}},
				}, nil
			}}))
		cmd.SetOut(got)
		cmd.SetArgs([]string{"test-environment"})
		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, "Deleted environment ID: 999be8ae-36b6-439d-81dc-e6fd137b0ffe Name: test-environment\n", got.String())
	})
}
