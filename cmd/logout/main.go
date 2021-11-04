package main

import "github.com/zalando/go-keyring"

const (
	serviceName = "cloud.calyptia.com"
)

func main() {
	_ = keyring.Delete(serviceName, "access_token")
}
