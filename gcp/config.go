package gcp

import (
	"fmt"
	"os"
	"strings"
)

const (
	DefaultZone        = "us"
	DefaultMachineType = "e2-highmem-4"
	BaseCalyptiaImage  = "projects/calyptia-infra/global/images/family/gold-calyptia-core"
)

// metadata
type metadata struct {
	Items []item `yaml:"items"`
}

// Items
type item struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// tags
type tags struct {
	Items []string `yaml:"items"`
}

// disk
type disk struct {
	DeviceName       string           `yaml:"deviceName"`
	Type             string           `yaml:"type"`
	Boot             bool             `yaml:"boot"`
	AutoDelete       bool             `yaml:"autoDelete"`
	InitializeParams initializeParams `yaml:"initializeParams"`
}

// networkInterface
type networkInterface struct {
	Network       string         `yaml:"network"`
	AccessConfigs []accessConfig `yaml:"accessConfigs"`
}

// accessConfig
type accessConfig struct {
	Name  string `yaml:"name"`
	NatIP string `yaml:"natIP,omitempty"`
}

// Config
type Config struct {
	projectID string      `yaml:"-"`
	Resources []resources `yaml:"resources"`
	Outputs   []output    `yaml:"outputs"`
}

// resources
type resources struct {
	Type       string     `yaml:"type"`
	Name       string     `yaml:"name"`
	Properties properties `yaml:"properties"`
}

// properties
type properties struct {
	Zone              string             `yaml:"zone"`
	MachineType       string             `yaml:"machineType"`
	Metadata          metadata           `yaml:"metadata"`
	Tags              tags               `yaml:"tags"`
	Disks             []disk             `yaml:"disks"`
	NetworkInterfaces []networkInterface `yaml:"networkInterfaces"`
}

// initializeParams
type initializeParams struct {
	SourceImage string `yaml:"sourceImage"`
}

// output
type output struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func NewConfig(projectID, coreInstanceName, environment string) Config {
	c := Config{
		projectID: projectID,
		Resources: []resources{
			{
				Type: "compute.v1.instance",
				Name: "calyptia-core-instance",
				Properties: properties{
					Zone:        DefaultZone,
					MachineType: fmt.Sprintf("projects/%s/zones/%s/machineTypes/%s", projectID, DefaultZone, DefaultMachineType),
					Metadata:    metadata{Items: []item{}},
					Tags:        tags{Items: []string{"calyptia-core-instance", coreInstanceName, environment}},
					Disks: []disk{
						{
							DeviceName: "boot",
							Type:       "PERSISTENT",
							Boot:       true,
							AutoDelete: true,
							InitializeParams: initializeParams{
								SourceImage: BaseCalyptiaImage,
							},
						},
					},
					NetworkInterfaces: []networkInterface{
						{
							Network: fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/networks/default", projectID),
							AccessConfigs: []accessConfig{
								{
									Name: "External NAT",
								},
							},
						},
					},
				},
			},
		},
		Outputs: []output{
			{
				Name:  "IPAddress",
				Value: "$(ref.calyptia-core-instance.networkInterfaces[0].accessConfigs[0].natIP)",
			},
		},
	}
	return c
}

func (c *Config) SetZone(zone string) *Config {
	if zone == "" {
		return c
	}
	c.Resources[0].Properties.Zone = zone
	return c
}

func (c *Config) SetMachineType(machineType string) *Config {
	if machineType == "" {
		return c
	}
	c.Resources[0].Properties.MachineType = fmt.Sprintf("projects/%s/zones/%s/machineTypes/%s", c.projectID, c.Resources[0].Properties.Zone, machineType)
	return c
}

func (c *Config) SetName(name string) *Config {
	if name == "" {
		return c
	}

	c.Resources[0].Name = name
	c.Outputs[0].Value = fmt.Sprintf("$(ref.%s.networkInterfaces[0].accessConfigs[0].natIP)", name)
	return c
}

func (c *Config) SetTags(tags []string) *Config {
	if len(tags) == 0 {
		return c
	}
	c.Resources[0].Properties.Tags.Items = append(c.Resources[0].Properties.Tags.Items, tags...)
	c.Resources[0].Properties.Metadata.Items = append(c.Resources[0].Properties.Metadata.Items, item{
		Key:   "CALYPTIA_CORE_INSTANCE_TAGS",
		Value: strings.Join(tags, ","),
	})
	return c
}

func (c *Config) SetImage(version string) *Config {
	if version == "" || version == "latest" {
		return c
	}
	c.Resources[0].Properties.Disks[0].InitializeParams.SourceImage = fmt.Sprintf("projects/%s/global/images/%s", c.projectID, version)
	return c
}
func (c *Config) SetAggregator(name string) *Config {
	if name == "" {
		return c
	}
	c.Resources[0].Properties.Metadata.Items = append(c.Resources[0].Properties.Metadata.Items, item{
		Key:   "CALYPTIA_CLOUD_AGGREGATOR_NAME",
		Value: name,
	})
	return c
}
func (c *Config) SetEnvironment(environment string) *Config {
	if environment == "" {
		return c
	}
	c.Resources[0].Properties.Metadata.Items = append(c.Resources[0].Properties.Metadata.Items, item{
		Key:   "CALYPTIA_CORE_INSTANCE_ENVIRONMENT",
		Value: environment,
	})
	return c
}

func (c *Config) SetProjectToken(token string) *Config {
	if token == "" {
		return c
	}
	c.Resources[0].Properties.Metadata.Items = append(c.Resources[0].Properties.Metadata.Items, item{
		Key:   "CALYPTIA_CLOUD_PROJECT_TOKEN",
		Value: token,
	})
	return c
}

func (c *Config) SetSSHKey(user string, key string) *Config {

	if user == "" || key == "" {
		return c
	}

	loadKey, _ := c.loadKey(key)

	c.Resources[0].Properties.Metadata.Items = append(c.Resources[0].Properties.Metadata.Items, item{
		Key:   "ssh-keys",
		Value: fmt.Sprintf("%s:%s", user, loadKey),
	})
	return c
}

func (c *Config) SetIP(ip string) *Config {
	if ip == "" {
		return c
	}
	c.Resources[0].Properties.NetworkInterfaces[0].AccessConfigs[0].NatIP = ip

	return c
}

func (c *Config) SetNetwork(network string) *Config {
	if network == "" {
		return c
	}
	c.Resources[0].Properties.NetworkInterfaces[0].Network = fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/networks/%s", c.projectID, network)
	return c
}

func (c *Config) loadKey(path string) (string, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("could not load key: %w", err)
	}
	return string(file), nil
}

func (c *Config) SetGitHubToken(token string) *Config {
	if token == "" {
		return c
	}

	c.Resources[0].Properties.Metadata.Items = append(c.Resources[0].Properties.Metadata.Items, item{
		Key:   "GITHUB_TOKEN",
		Value: token,
	})
	return c
}
