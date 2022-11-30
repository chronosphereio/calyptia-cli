package config

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/config"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
)

const KeyCliHealth = "cli_health"

func NewCmdCheckInstall(c *cfg.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "check-install",
		Short: "Check the current configuration and report any issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			CheckInstall(c)
			return nil
		},
	}
}

func CheckInstall(c *cfg.Config) {
	ok := true
	if !checkToken(c) {
		ok = false
	}
	if !checkUrl(c) {
		ok = false
	}
	if !checkSSL(c) {
		ok = false
	}
	var k8sConfig *rest.Config
	var pass bool
	if k8sConfig, pass = checkK8sCredentials(); !pass {
		ok = false
	}
	if !checkK8sHealth(k8sConfig) {
		ok = false
	}
	if !ok {
		fmt.Printf("Calyptia CLI is not ready to use\n")
		return
	}
	err := c.LocalData.Save(KeyCliHealth, "ok")
	if err != nil {
		fmt.Printf("Calyptia CLI is ready to use but we couldnt save it health status\n")
	}
	fmt.Printf("Calyptia CLI is ready to use\n")

}

func checkToken(c *cfg.Config) bool {
	if c.ProjectToken == "" {
		fmt.Printf("You need to set a project token to use Calyptia CLI\n" +
			"\tTip: you can use the flag --token or the command calyptia config set_token to set your project token\n")
		return false
	}
	fmt.Printf("Token is set\n")
	return true
}

func checkUrl(c *cfg.Config) bool {
	ctx := context.Background()
	_, err := c.Cloud.Environments(ctx, c.ProjectID, types.EnvironmentsParams{})
	if err != nil {
		fmt.Printf("%s is not working properly\n", c.BaseURL)
		fmt.Printf("\tTip: Check if your url, token, internet connection, firewalls and blocked ports\n")
		return false
	}
	fmt.Printf("%s is reacheable\n", c.BaseURL)
	return true
}

func checkSSL(c *cfg.Config) bool {
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		},
	}
	resp, err := client.Get(c.BaseURL)
	if err != nil {
		fmt.Printf("Could not establish a secure connection with %s\n", c.BaseURL)
		return false
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Unexpected status code:", resp.StatusCode)
		return false
	}
	fmt.Printf("Could establish secure connection with %s\n", c.BaseURL)
	return true
}

func checkK8sCredentials() (*rest.Config, bool) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	kubeClientConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		fmt.Printf("Kubernetes credentials are not set\n")
		return nil, false
	}
	fmt.Printf("Kubernetes credentials are valid\n")
	return kubeClientConfig, true
}
func checkK8sHealth(kubeClientConfig *rest.Config) bool {
	client, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		fmt.Printf("Kubernetes is not reachable\n")
		return false
	}
	_, err = client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Kubernetes is not reachable\n")
		return false
	}
	fmt.Printf("Kubernetes is reachable\n")
	return true
}
