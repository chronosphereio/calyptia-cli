package main

import "github.com/zalando/go-keyring"

const (
	serviceName = "cloud.calyptia.com"
	authPrefix  = "auth."
)

func main() {
	cleanupLocalAuth()
}

func cleanupLocalAuth() {
	_ = keyring.Delete(serviceName, authPrefix+"access_token")
	_ = keyring.Delete(serviceName, authPrefix+"expires_in")
	_ = keyring.Delete(serviceName, authPrefix+"refresh_token")
}
