package gcp

import (
	"context"
	"fmt"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/deploymentmanager/v2"
	option "google.golang.org/api/option"
	"gopkg.in/yaml.v2"
)

//go:generate moq -out client_mock.go . Client
type Client interface {
	Delete(ctx context.Context, coreInstanceName string) error
	SetConfig(newConfig Config)
	Deploy(context.Context) error
	FollowOperations(context.Context) (*deploymentmanager.Operation, error)
	Rollback(context.Context) error
	GetInstance(ctx context.Context, zone, instance string) (*compute.Instance, error)
}

type DefaultClient struct {
	projectName    string
	config         Config
	deploymentName string
	manager        *deploymentmanager.Service
	environment    string
	compute        *compute.Service
}

func (c *DefaultClient) SetConfig(newConfig Config) {
	c.config = newConfig
}

func (c *DefaultClient) Deploy(ctx context.Context) error {
	configBytes, err := yaml.Marshal(c.config)
	if err != nil {
		return err
	}
	targetConfiguration := &deploymentmanager.TargetConfiguration{
		Config: &deploymentmanager.ConfigFile{Content: string(configBytes)},
	}

	deployment := &deploymentmanager.Deployment{
		Name:   fmt.Sprintf("%s-%s-deployment", c.config.Resources[0].Name, c.environment),
		Target: targetConfiguration,
	}
	if err != nil {
		return err
	}
	insertDeployment, err := c.manager.Deployments.Insert(c.projectName, deployment).Context(ctx).Do()
	if err != nil {
		return err
	}
	c.deploymentName = insertDeployment.Name
	return nil
}

func New(ctx context.Context, projectName string, environment string, credentials string) (*DefaultClient, error) {
	var authOpts []option.ClientOption

	if credentials != "" {
		authOpts = append(authOpts, option.WithCredentialsFile(credentials))
	}

	m, err := deploymentmanager.NewService(ctx, authOpts...)
	if err != nil {
		return nil, err
	}
	c, err := compute.NewService(ctx, authOpts...)
	if err != nil {
		return nil, err
	}
	if projectName == "" {
		return nil, fmt.Errorf("project name is mandatory")
	}
	return &DefaultClient{projectName: projectName, manager: m, compute: c, environment: environment}, nil
}

func (c *DefaultClient) FollowOperations(ctx context.Context) (*deploymentmanager.Operation, error) {
	operation, err := c.manager.Operations.Get(c.projectName, c.deploymentName).Context(ctx).Do()
	if operation != nil && operation.Error != nil {
		return operation, fmt.Errorf("occurred an error with the %s operation: %v", operation.Name, operation.Error.Errors)
	}
	return operation, err
}

func (c *DefaultClient) Rollback(ctx context.Context) error {
	if err := c.Delete(ctx, c.config.Resources[0].Name); err != nil {
		return err
	}
	return nil
}
func (c *DefaultClient) Delete(ctx context.Context, coreInstanceName string) error {
	deploymentName := fmt.Sprintf("%s-%s-deployment", coreInstanceName, c.environment)
	_, err := c.manager.Deployments.Delete(c.projectName, deploymentName).Context(ctx).Do()
	return err
}

func (c *DefaultClient) GetInstance(ctx context.Context, zone, instance string) (*compute.Instance, error) {
	return c.compute.Instances.Get(c.projectName, zone, instance).Context(ctx).Do()
}
