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

// func Test_newCmdUpdateCoreInstanceK8s(t *testing.T) {
// 	coreInstanceName := "testing"
// 	t.Run("error", func(t *testing.T) {
// 		cmd := newCmdUpdateCoreInstanceK8s(configWithMock(&ClientMock{
// 			UpdateCoreInstanceFunc: func(ctx context.Context, CoreInstanceID string, payload types.UpdateCoreInstance) error {
// 				return errors.New("internal server error")
// 			},
// 			CoreInstancesFunc: func(ctx context.Context, projectID string, params types.CoreInstancesParams) (types.CoreInstances, error) {
// 				return types.CoreInstances{
// 					Items: []types.CoreInstance{
// 						{
// 							Name: coreInstanceName,
// 						},
// 					},
// 				}, nil
// 			},
// 		}), nil)

// 		cmd.SetArgs([]string{coreInstanceName})
// 		cmd.SetOut(io.Discard)

// 		err := cmd.Execute()
// 		wantErrMsg(t, `could not update core instance at calyptia cloud: internal server error`, err)
// 	})

// 	t.Run("ok", func(t *testing.T) {
// 		got := &bytes.Buffer{}
// 		cmd := newCmdUpdateCoreInstanceK8s(configWithMock(&ClientMock{
// 			CoreInstancesFunc: func(ctx context.Context, projectID string, params types.CoreInstancesParams) (types.CoreInstances, error) {
// 				return types.CoreInstances{
// 					Items: []types.CoreInstance{
// 						{
// 							Name: coreInstanceName,
// 						},
// 					},
// 					EndCursor: nil,
// 				}, nil
// 			},
// 			UpdateCoreInstanceFunc: func(ctx context.Context, CoreInstanceID string, payload types.UpdateCoreInstance) error {
// 				return nil
// 			},
// 		}), fake.NewSimpleClientset())
// 		cmd.SetOut(got)
// 		cmd.SetArgs([]string{coreInstanceName})
// 		err := cmd.Execute()
// 		wantEq(t, nil, err)
// 		wantEq(t, "calyptia-core instance successfully updated\n", got.String())
// 	})
// }
