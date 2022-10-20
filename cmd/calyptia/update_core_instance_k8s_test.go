package main

import (
	"bytes"
	"context"
	"errors"
	"github.com/calyptia/api/types"
	"io"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func Test_newCmdUpdateCoreInstanceK8s(t *testing.T) {
	coreInstanceName := "testing"
	t.Run("error", func(t *testing.T) {
		cmd := newCmdUpdateCoreInstanceK8s(configWithMock(&ClientMock{
			UpdateAggregatorFunc: func(ctx context.Context, aggregatorID string, payload types.UpdateAggregator) error {
				return errors.New("internal server error")
			},
			AggregatorsFunc: func(ctx context.Context, projectID string, params types.AggregatorsParams) (types.Aggregators, error) {
				return types.Aggregators{
					Items: []types.Aggregator{
						{
							Name: coreInstanceName,
						},
					},
				}, nil
			},
		}), nil)

		cmd.SetArgs([]string{coreInstanceName})
		cmd.SetOut(io.Discard)

		err := cmd.Execute()
		wantErrMsg(t, `could not update core instance at calyptia cloud: internal server error`, err)
	})

	t.Run("ok", func(t *testing.T) {
		got := &bytes.Buffer{}
		cmd := newCmdUpdateCoreInstanceK8s(configWithMock(&ClientMock{
			AggregatorsFunc: func(ctx context.Context, projectID string, params types.AggregatorsParams) (types.Aggregators, error) {
				return types.Aggregators{
					Items: []types.Aggregator{
						{
							Name: coreInstanceName,
						},
					},
					EndCursor: nil,
				}, nil
			},
			UpdateAggregatorFunc: func(ctx context.Context, aggregatorID string, payload types.UpdateAggregator) error {
				return nil
			},
		}), fake.NewSimpleClientset())
		cmd.SetOut(got)
		cmd.SetArgs([]string{coreInstanceName})
		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, "calyptia-core instance successfully updated\n", got.String())
	})
}
