package main

import (
	"context"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/calyptia/cli/commands"
)

func main() {
	_ = godotenv.Load()

	root := commands.NewRootCmd(context.Background())
	cobra.CheckErr(root.Execute())
}
