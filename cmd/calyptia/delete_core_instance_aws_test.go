package main

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	types2 "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/calyptia/api/types"
	"github.com/calyptia/cli/aws"
)

func Test_newCmdDeleteCoreInstanceOnAWS(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		instanceID := "0xdeadbeef"
		got := bytes.Buffer{}
		resourcesToDelete := []aws.Resource{
			{
				ID:   "0xdeadbeef",
				Type: types2.ResourceTypeInstance,
			},
		}

		instanceParams := aws.CreatedInstance{
			CoreInstanceName: "core-test",
			MetadataAWS: types.MetadataAWS{
				PrivateIPv4:     "192.168.0.1",
				PublicIPv4:      "",
				EC2InstanceID:   instanceID,
				EC2InstanceType: aws.DefaultInstanceTypeName,
			},
		}

		cmd := newCmdDeleteCoreInstanceOnAWS(
			configWithMock(&ClientMock{
				EnvironmentsFunc: func(ctx context.Context, projectID string, params types.EnvironmentsParams) (types.Environments, error) {
					return types.Environments{Items: []types.Environment{{Name: "default"}}}, nil
				},
				AggregatorsFunc: func(ctx context.Context, projectID string, params types.AggregatorsParams) (types.Aggregators, error) {
					return types.Aggregators{
						Items: []types.Aggregator{
							{
								Name: "core-instance",
							},
						},
					}, nil
				},
			}),
			&aws.ClientMock{
				GetResourcesByTagsFunc: func(ctx context.Context, tags aws.TagSpec) ([]aws.Resource, error) {
					return resourcesToDelete, nil
				},
				CreateInstanceFunc: func(ctx context.Context, in *aws.CreateInstanceParams) (aws.CreatedInstance, error) {
					return instanceParams, nil
				},
				DeleteResourcesFunc: func(ctx context.Context, resources []aws.Resource) error {
					return nil
				},
			},
		)

		cmd.SetOut(&got)
		cmd.SetArgs([]string{"core-instance", "--environment", "default"})

		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, "The following resources will be removed from your AWS account:\n"+
			"Resource type: instance - Unique ID: "+instanceParams.EC2InstanceID+"\n", got.String())
	})
	t.Run("aws error", func(t *testing.T) {
		instanceID := "0xdeadbeef"
		got := bytes.Buffer{}

		instanceParams := aws.CreatedInstance{
			CoreInstanceName: "core-test",
			MetadataAWS: types.MetadataAWS{
				PrivateIPv4:     "192.168.0.1",
				PublicIPv4:      "",
				EC2InstanceID:   instanceID,
				EC2InstanceType: aws.DefaultInstanceTypeName,
			},
		}

		cmd := newCmdDeleteCoreInstanceOnAWS(
			configWithMock(&ClientMock{
				EnvironmentsFunc: func(ctx context.Context, projectID string, params types.EnvironmentsParams) (types.Environments, error) {
					return types.Environments{Items: []types.Environment{{Name: "default"}}}, nil
				},
				AggregatorsFunc: func(ctx context.Context, projectID string, params types.AggregatorsParams) (types.Aggregators, error) {
					return types.Aggregators{
						Items: []types.Aggregator{
							{
								Name: "core-instance",
							},
						},
					}, nil
				},
			}),
			&aws.ClientMock{
				GetResourcesByTagsFunc: func(ctx context.Context, tags aws.TagSpec) ([]aws.Resource, error) {
					return []aws.Resource{}, fmt.Errorf("cannot get tags")
				},
				CreateInstanceFunc: func(ctx context.Context, in *aws.CreateInstanceParams) (aws.CreatedInstance, error) {
					return instanceParams, nil
				},
				DeleteResourcesFunc: func(ctx context.Context, resources []aws.Resource) error {
					return nil
				},
			},
		)

		cmd.SetOut(&got)
		cmd.SetArgs([]string{"core-instance", "--environment", "default"})

		err := cmd.Execute()
		wantNoEq(t, nil, err)
	})
	t.Run("calyptia cloud error", func(t *testing.T) {
		instanceID := "0xdeadbeef"
		got := bytes.Buffer{}

		instanceParams := aws.CreatedInstance{
			CoreInstanceName: "core-test",
			MetadataAWS: types.MetadataAWS{
				PrivateIPv4:     "192.168.0.1",
				PublicIPv4:      "",
				EC2InstanceID:   instanceID,
				EC2InstanceType: aws.DefaultInstanceTypeName,
			},
		}

		cmd := newCmdDeleteCoreInstanceOnAWS(
			configWithMock(
				&ClientMock{
					EnvironmentsFunc: func(ctx context.Context, projectID string, params types.EnvironmentsParams) (types.Environments, error) {
						return types.Environments{Items: []types.Environment{{Name: "default"}}}, nil
					},
					AggregatorsFunc: func(ctx context.Context, projectID string, params types.AggregatorsParams) (types.Aggregators, error) {
						return types.Aggregators{}, fmt.Errorf("could not get core-instance")
					},
				}),
			&aws.ClientMock{
				GetResourcesByTagsFunc: func(ctx context.Context, tags aws.TagSpec) ([]aws.Resource, error) {
					return []aws.Resource{}, fmt.Errorf("cannot get tags")
				},
				CreateInstanceFunc: func(ctx context.Context, in *aws.CreateInstanceParams) (aws.CreatedInstance, error) {
					return instanceParams, nil
				},
				DeleteResourcesFunc: func(ctx context.Context, resources []aws.Resource) error {
					return nil
				},
			},
		)

		cmd.SetOut(&got)
		cmd.SetArgs([]string{"core-instance", "--environment", "default"})

		err := cmd.Execute()
		wantNoEq(t, nil, err)
		wantErrMsg(t, `could not load core instance ID: could not get core-instance`, err)

	})

}
