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
		cmd := newCmdCreateCoreInstanceOnK8s(configWithMock(&ClientMock{
			CreateAggregatorFunc: func(ctx context.Context, payload types.CreateAggregator) (types.CreatedAggregator, error) {
				return types.CreatedAggregator{}, errors.New("internal server error")
			},
		}), nil)

		cmd.SetOut(io.Discard)

		err := cmd.Execute()
		wantErrMsg(t, `could not create core instance at calyptia cloud: internal server error`, err)
	})

	t.Run("ok", func(t *testing.T) {
		got := &bytes.Buffer{}
		cmd := newCmdCreateCoreInstanceOnK8s(configWithMock(&ClientMock{
			CreateAggregatorFunc: func(ctx context.Context, payload types.CreateAggregator) (types.CreatedAggregator, error) {
				return types.CreatedAggregator{
					ID:   "want-aggregator-id",
					Name: "want-aggregator-name",
				}, nil
			},
		}), fake.NewSimpleClientset())
		cmd.SetOut(got)

		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, "secret=\"want-aggregator-name-private-rsa.key\"\n"+
			"cluster_role=\"want-aggregator-name-cluster-role\"\n"+
			"service_account=\"want-aggregator-name-service-account\"\n"+
			"cluster_role_binding=\"want-aggregator-name-cluster-role-binding\"\n"+
			"deployment=\"want-aggregator-name-deployment\"\n", got.String())
	})
}
