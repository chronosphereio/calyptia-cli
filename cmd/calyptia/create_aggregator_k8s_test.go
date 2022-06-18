package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"k8s.io/client-go/kubernetes/fake"

	"github.com/calyptia/api/types"
)

func Test_newCmdCreateAggregatorOnK8s(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		cmd := newCmdCreateAggregatorOnK8s(configWithMock(&ClientMock{
			CreateAggregatorFunc: func(ctx context.Context, payload types.CreateAggregator) (types.CreatedAggregator, error) {
				return types.CreatedAggregator{}, errors.New("internal server error")
			},
		}), nil)
		cmd.SetOutput(io.Discard)

		err := cmd.Execute()
		wantErrMsg(t, `could not create core instance at calyptia cloud: internal server error`, err)
	})

	t.Run("ok", func(t *testing.T) {
		got := &bytes.Buffer{}
		cmd := newCmdCreateAggregatorOnK8s(configWithMock(&ClientMock{
			CreateAggregatorFunc: func(ctx context.Context, payload types.CreateAggregator) (types.CreatedAggregator, error) {
				return types.CreatedAggregator{
					ID:   "want-aggregator-id",
					Name: "want-aggregator-name",
				}, nil
			},
		}), fake.NewSimpleClientset())
		cmd.SetOutput(got)

		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, "cluster role: \"want-aggregator-name-cluster-role\"\n"+
			"service account: \"want-aggregator-name-service-account\"\n"+
			"cluster role binding: \"want-aggregator-name-cluster-role-binding\"\n"+
			"deployment: \"want-aggregator-name-deployment\"\n", got.String())
	})
}
