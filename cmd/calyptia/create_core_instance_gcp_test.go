package main

// import (
// 	"bytes"
// 	"context"
// 	"testing"

// 	"google.golang.org/api/compute/v1"

// 	"google.golang.org/api/deploymentmanager/v2"

// 	"github.com/calyptia/api/types"
// 	"github.com/calyptia/cli/gcp"
// )

// func Test_newCmdCreateCoreInstanceOnGCP(t *testing.T) {
// 	t.Run("create with default settings", func(t *testing.T) {
// 		got := bytes.Buffer{}
// 		cmd := newCmdCreateCoreInstanceOnGCP(configWithMock(&ClientMock{
// 			EnvironmentsFunc: func(ctx context.Context, projectID string, params types.EnvironmentsParams) (types.Environments, error) {
// 				return types.Environments{Items: []types.Environment{{Name: "default"}}}, nil
// 			},
// 		}), &gcp.ClientMock{
// 			SetConfigFunc: func(newConfig gcp.Config) {

// 			},
// 			DeployFunc: func(contextMoqParam context.Context) error {
// 				return nil
// 			},
// 			FollowOperationsFunc: func(contextMoqParam context.Context) (*deploymentmanager.Operation, error) {
// 				return &deploymentmanager.Operation{
// 					Status: OperationConcluded,
// 				}, nil
// 			},
// 			GetInstanceFunc: func(ctx context.Context, zone string, instance string) (*compute.Instance, error) {

// 				return &compute.Instance{
// 					NetworkInterfaces: []*compute.NetworkInterface{
// 						{
// 							NetworkIP: "10.0.0.1",
// 							AccessConfigs: []*compute.AccessConfig{
// 								{
// 									NatIP: "111.111.111.111",
// 								},
// 							},
// 						},
// 					},
// 				}, nil
// 			},
// 		})

// 		cmd.SetOut(&got)
// 		err := cmd.Execute()
// 		wantEq(t, nil, err)
// 		wantEq(t, "[*] Waiting for create operation...done.\n[*] Calyptia Core Instance created.\nExternal IP: 111.111.111.111\nInternal IP: 10.0.0.1", got.String())

// 	})
// }
