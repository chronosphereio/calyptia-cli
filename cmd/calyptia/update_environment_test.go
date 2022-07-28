package main

import (
	"bytes"
	"context"
	cloud "github.com/calyptia/api/types"
	"testing"
)

func TestUpdateEnvironment(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		got := &bytes.Buffer{}
		cmd := newCmdUpdateEnvironment(configWithMock(&ClientMock{
			UpdateEnvironmentFunc: func(ctx context.Context, environmentID string, payload cloud.UpdateEnvironment) error {
				return nil
			},
			EnvironmentsFunc: func(ctx context.Context, projectID string, params cloud.EnvironmentsParams) (cloud.Environments, error) {
				return cloud.Environments{
					Items: []cloud.Environment{{ID: "999be8ae-36b6-439d-81dc-e6fd137b0ffe", Name: "test-environment"}},
				}, nil
			}}))
		cmd.SetOut(got)
		cmd.SetArgs([]string{"test-environment", "--name", "new-name"})
		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, "Updated environment ID: 999be8ae-36b6-439d-81dc-e6fd137b0ffe Name: new-name\n", got.String())
	})
	t.Run("same name", func(t *testing.T) {
		got := &bytes.Buffer{}
		cmd := newCmdUpdateEnvironment(configWithMock(&ClientMock{
			UpdateEnvironmentFunc: func(ctx context.Context, environmentID string, payload cloud.UpdateEnvironment) error {
				return nil
			},
			EnvironmentsFunc: func(ctx context.Context, projectID string, params cloud.EnvironmentsParams) (cloud.Environments, error) {
				return cloud.Environments{
					Items: []cloud.Environment{{ID: "999be8ae-36b6-439d-81dc-e6fd137b0ffe", Name: "test-environment"}},
				}, nil
			}}))
		cmd.SetOut(got)
		cmd.SetArgs([]string{"test-environment", "--name", "test-environment"})
		err := cmd.Execute()
		wantEq(t, "environment name unchanged", err.Error())
	})
}