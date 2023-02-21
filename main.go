package main

import (
	"context"

	cmd "github.com/calyptia/cli/cmd"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

func main() {
	_ = godotenv.Load()

	cmd := cmd.NewRootCmd(context.Background())
	cobra.CheckErr(cmd.Execute())
}
