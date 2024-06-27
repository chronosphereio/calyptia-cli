package main

import (
	"context"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	cmd "github.com/calyptia/cli/commands"
)

func main() {
	_ = godotenv.Load()

	cmd := cmd.NewRootCmd(context.Background())
	cobra.CheckErr(cmd.Execute())
}
