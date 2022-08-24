package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/calyptia/cli/aws/ifaces"
)

func TestDefaultClient_EnsureKeyPair(t *testing.T) {
	defaultKeyPairName := "test"

	tt := []struct {
		name               string
		client             *ifaces.ClientMock
		createKeyPairCount int
		wantErr            bool
		wantPairName       string
	}{
		{
			name: "return existing keypair",
			client: &ifaces.ClientMock{
				DescribeKeyPairsFunc: func(ctx context.Context, params *ec2.DescribeKeyPairsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error) {
					return &ec2.DescribeKeyPairsOutput{KeyPairs: []types.KeyPairInfo{
						{
							KeyName: &defaultKeyPairName,
						},
					}}, nil
				},
			},
			wantErr:      false,
			wantPairName: defaultKeyPairName,
		},
		{
			name: "error in amazon, returns empty key",
			client: &ifaces.ClientMock{
				DescribeKeyPairsFunc: func(ctx context.Context, params *ec2.DescribeKeyPairsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error) {
					return nil, ErrKeyPairNotFound
				},
			},
			wantErr:      true,
			wantPairName: "",
		},
		{
			name: "non existing keypair creates a new keypair",
			client: &ifaces.ClientMock{
				DescribeKeyPairsFunc: func(ctx context.Context, params *ec2.DescribeKeyPairsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error) {
					return nil, nil
				},
				CreateKeyPairFunc: func(ctx context.Context, params *ec2.CreateKeyPairInput, optFns ...func(*ec2.Options)) (*ec2.CreateKeyPairOutput, error) {
					return &ec2.CreateKeyPairOutput{KeyName: &defaultKeyPairName}, nil
				},
			},
			wantErr:            false,
			wantPairName:       defaultKeyPairName,
			createKeyPairCount: 1,
		},
	}

	ctx := context.Background()

	for _, tc := range tt {
		cc := DefaultClient{ec2Client: tc.client}
		t.Run(tc.name, func(t *testing.T) {
			pair, err := cc.EnsureKeyPair(ctx, defaultKeyPairName, "default")
			if err != nil {
				if !tc.wantErr {
					t.Errorf("err: %v != nil", err)
					return
				}
			}

			if tc.wantPairName != "" {
				if want, got := tc.wantPairName, pair; want != got {
					t.Errorf("want key pair name: %s, got: %s", want, got)
					return
				}
			}

			if tc.createKeyPairCount > 0 {
				calls := tc.client.CreateKeyPairCalls()
				if want, got := tc.createKeyPairCount, len(calls); want != got {
					t.Errorf("want %d create key pair calls; got %d", want, got)
					return
				}
			}
		})
	}
}
