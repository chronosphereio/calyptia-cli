package main

import (
	"bytes"
	"context"
	"github.com/calyptia/api/types"
	"github.com/calyptia/cli/aws"
	"io"
	"testing"
)

func Test_newCmdCreateCoreInstanceOnAWS(t *testing.T) {
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
			configWithMock(&ClientMock{}),
			&aws.ClientMock{
				CreateInstanceFunc: func(ctx context.Context, in *aws.CreateInstanceParams) (aws.CreatedInstance, error) {
					return instanceParams, nil
				},
			}, &CoreInstancePollerMock{
				ReadyFunc: func(ctx context.Context, name string) error {
					return nil
				},
			})

		cmd.SetOut(&got)
		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, "Creating calyptia core instance on AWS\n"+
			"calyptia core instance running on AWS as: instance-id: "+instanceParams.EC2InstanceID+", instance-type: "+instanceParams.EC2InstanceType+", privateIPv4: "+instanceParams.PrivateIPv4+"\n"+
			"Calyptia core instance is ready to use.", got.String())
	})

	t.Run("AWS error", func(t *testing.T) {
		cmd := newCmdCreateCoreInstanceOnAWS(
			configWithMock(&ClientMock{}),
			&aws.ClientMock{
				CreateInstanceFunc: func(ctx context.Context, in *aws.CreateInstanceParams) (aws.CreatedInstance, error) {
					return aws.CreatedInstance{}, aws.ErrSubnetNotFound
				},
			}, &CoreInstancePollerMock{
				ReadyFunc: func(ctx context.Context, name string) error {
					return nil
				},
			})

		cmd.SetOut(io.Discard)
		err := cmd.Execute()
		wantErrMsg(t, `could not create AWS instance: subnet not found`, err)
	})

	t.Run("calyptia cloud error", func(t *testing.T) {
		cmd := newCmdCreateCoreInstanceOnAWS(
			configWithMock(&ClientMock{}),
			&aws.ClientMock{
				CreateInstanceFunc: func(ctx context.Context, in *aws.CreateInstanceParams) (aws.CreatedInstance, error) {
					return aws.CreatedInstance{}, nil
				},
			}, &CoreInstancePollerMock{
				ReadyFunc: func(ctx context.Context, name string) error {
					return errCoreInstanceNotRunning
				},
			})

		cmd.SetOut(io.Discard)
		err := cmd.Execute()
		wantErrMsg(t, `calyptia core instance could not reach ready status: core instance not in running status`, err)
	})
}
