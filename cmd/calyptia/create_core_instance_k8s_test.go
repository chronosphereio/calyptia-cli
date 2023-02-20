package main

// import (
// 	"bytes"
// 	"context"
// 	"errors"
// 	"io"
// 	"testing"

// 	"k8s.io/client-go/kubernetes/fake"

// 	"github.com/calyptia/api/types"
// )

// func Test_newCmdCreateCoreInstanceOnK8s(t *testing.T) {
// 	t.Run("error", func(t *testing.T) {
// 		cmd := newCmdCreateCoreInstanceOnK8s(configWithMock(&ClientMock{
// 			CreateCoreInstanceFunc: func(ctx context.Context, payload types.CreateCoreInstance) (types.CreatedCoreInstance, error) {
// 				return types.CreatedCoreInstance{}, errors.New("internal server error")
// 			},
// 		}), nil)

// 		cmd.SetOut(io.Discard)

// 		err := cmd.Execute()
// 		wantErrMsg(t, `could not create core instance at calyptia cloud: internal server error`, err)
// 	})

// 	t.Run("ok", func(t *testing.T) {
// 		got := &bytes.Buffer{}
// 		cmd := newCmdCreateCoreInstanceOnK8s(configWithMock(&ClientMock{
// 			CreateCoreInstanceFunc: func(ctx context.Context, payload types.CreateCoreInstance) (types.CreatedCoreInstance, error) {
// 				return types.CreatedCoreInstance{
// 					ID:              "want-CoreInstance-id",
// 					Name:            "want-CoreInstance-name",
// 					EnvironmentName: "default",
// 				}, nil
// 			},
// 		}), fake.NewSimpleClientset())
// 		cmd.SetOut(got)

// 		err := cmd.Execute()
// 		wantEq(t, nil, err)
// 		wantEq(t, "secret=\"calyptia-want-CoreInstance-name-default-secret\"\n"+
// 			"cluster_role=\"calyptia-want-CoreInstance-name-default-cluster-role\"\n"+
// 			"service_account=\"calyptia-want-CoreInstance-name-default-service-account\"\n"+
// 			"cluster_role_binding=\"calyptia-want-CoreInstance-name-default-cluster-role-binding\"\n"+
// 			"deployment=\"calyptia-want-CoreInstance-name-default-deployment\"\n", got.String())
// 	})
// }
