package main

import (
	"github.com/Riki1312/nt-cli/internal/cli"
	"github.com/Riki1312/nt-cli/internal/output"
)

func main() {
	cmd := cli.NewRootCmd()
	if err := cmd.Execute(); err != nil {
		output.HandleError(err)
	}
}
