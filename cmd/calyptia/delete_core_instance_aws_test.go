package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/calyptia/api/types"
	"github.com/calyptia/cli/aws"
)

func Test_newCmdDeleteCoreInstanceOnAWS(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		got := bytes.Buffer{}

		instanceParams := aws.CreatedInstance{
			CoreInstanceName: "core-test",
			MetadataAWS: types.MetadataAWS{
				PrivateIPv4:     "192.168.0.1",
				PublicIPv4:      "",
				EC2InstanceID:   "i-0xdeadbeef",
				EC2InstanceType: aws.DefaultInstanceTypeName,
			},
		}

		cmd := newCmdCreateCoreInstanceOnAWS(
			configWithMock(&ClientMock{
				EnvironmentsFunc: func(ctx context.Context, projectID string, params types.EnvironmentsParams) (types.Environments, error) {
					return types.Environments{Items: []types.Environment{{Name: "default"}}}, nil
				},
			}),
			&aws.ClientMock{
				CreateInstanceFunc: func(ctx context.Context, in *aws.CreateInstanceParams) (aws.CreatedInstance, error) {
					return instanceParams, nil
				},
			}, &CoreInstancePollerMock{
				ReadyFunc: func(ctx context.Context, env, name string) (string, error) {
					return "", nil
				},
			})

		cmd.SetOut(&got)
		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, "Creating calyptia core instance on AWS\n"+
			"calyptia core instance running on AWS instance-id: "+instanceParams.EC2InstanceID+", instance-type: "+instanceParams.EC2InstanceType+", privateIPv4: "+instanceParams.PrivateIPv4+"\n"+
			"Calyptia core instance is ready to use.\n", got.String())
	})

	t.Run("error without env", func(t *testing.T) {
		got := bytes.Buffer{}

		instanceParams := aws.CreatedInstance{
			CoreInstanceName: "core-test",
			MetadataAWS: types.MetadataAWS{
				PrivateIPv4:     "192.168.0.1",
				PublicIPv4:      "",
				EC2InstanceID:   "i-0xdeadbeef",
				EC2InstanceType: aws.DefaultInstanceTypeName,
			},
		}

		cmd := newCmdCreateCoreInstanceOnAWS(
			configWithMock(
				&ClientMock{
					EnvironmentsFunc: func(ctx context.Context, projectID string, params types.EnvironmentsParams) (types.Environments, error) {
						return types.Environments{}, fmt.Errorf("not found env")
					},
				},
			),
			&aws.ClientMock{
				CreateInstanceFunc: func(ctx context.Context, in *aws.CreateInstanceParams) (aws.CreatedInstance, error) {
					return instanceParams, nil
				},
			}, &CoreInstancePollerMock{
				ReadyFunc: func(ctx context.Context, env, name string) (string, error) {
					return "", nil
				},
			})

		cmd.SetOut(&got)
		err := cmd.Execute()
		wantNoEq(t, nil, err)
	})

	t.Run("AWS error", func(t *testing.T) {
		cmd := newCmdCreateCoreInstanceOnAWS(
			configWithMock(&ClientMock{
				EnvironmentsFunc: func(ctx context.Context, projectID string, params types.EnvironmentsParams) (types.Environments, error) {
					return types.Environments{Items: []types.Environment{{Name: "default"}}}, nil
				},
			}),
			&aws.ClientMock{
				CreateInstanceFunc: func(ctx context.Context, in *aws.CreateInstanceParams) (aws.CreatedInstance, error) {
					return aws.CreatedInstance{}, aws.ErrSubnetNotFound
				},
			}, &CoreInstancePollerMock{
				ReadyFunc: func(ctx context.Context, env, name string) (string, error) {
					return "", nil
				},
			})

		cmd.SetOut(io.Discard)
		err := cmd.Execute()
		wantErrMsg(t, `could not create AWS instance: subnet not found`, err)
	})

	t.Run("calyptia cloud error", func(t *testing.T) {
		cmd := newCmdCreateCoreInstanceOnAWS(
			configWithMock(&ClientMock{
				EnvironmentsFunc: func(ctx context.Context, projectID string, params types.EnvironmentsParams) (types.Environments, error) {
					return types.Environments{Items: []types.Environment{{Name: "default"}}}, nil
				},
			}),
			&aws.ClientMock{
				CreateInstanceFunc: func(ctx context.Context, in *aws.CreateInstanceParams) (aws.CreatedInstance, error) {
					return aws.CreatedInstance{}, nil
				},
			}, &CoreInstancePollerMock{
				ReadyFunc: func(ctx context.Context, env, name string) (string, error) {
					return "", errCoreInstanceNotRunning
				},
			})

		cmd.SetOut(io.Discard)
		err := cmd.Execute()
		wantErrMsg(t, `calyptia core instance not ready: core instance not in running status`, err)
	})
}
