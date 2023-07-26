package main

import (
	"context"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	cmd "github.com/calyptia/cli/cmd"
)

//go:generate go-bindata -modtime 1 -prefix "../../" -o operator-manifest/manifest_data.go -pkg=cmd -ignore=debug/ -ignore=local/ -ignore=prometheus/ -ignore=samples/ ../../config/... manifests/...
func main() {
	_ = godotenv.Load()

	cmd := cmd.NewRootCmd(context.Background())
	cobra.CheckErr(cmd.Execute())
}
