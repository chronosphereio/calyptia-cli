package main

// import (
// 	"bytes"
// 	"context"
// 	"testing"

// 	"google.golang.org/api/deploymentmanager/v2"

// 	"github.com/calyptia/cli/gcp"
// )

// func Test_newCmdDeleteCoreInstanceOnGCP(t *testing.T) {
// 	t.Run("delete with default settings", func(t *testing.T) {
// 		got := bytes.Buffer{}
// 		cmd := newCmdDeleteCoreInstanceOnGCP(configWithMock(&ClientMock{}), &gcp.ClientMock{
// 			DeleteFunc: func(ctx context.Context, coreInstanceName string) error {
// 				return nil
// 			},
// 			FollowOperationsFunc: func(contextMoqParam context.Context) (*deploymentmanager.Operation, error) {
// 				return &deploymentmanager.Operation{
// 					Status: OperationConcluded,
// 				}, nil
// 			},
// 		})
// 		cmd.SetOut(&got)
// 		cmd.SetArgs([]string{"core-instance", "--project-id", "project-id", "--environment", "default"})
// 		err := cmd.Execute()
// 		wantEq(t, nil, err)
// 		wantEq(t, "[*] Waiting for delete operation...done.\n[*] The instance core-instance has been deleted", got.String())

// 	})

// }
