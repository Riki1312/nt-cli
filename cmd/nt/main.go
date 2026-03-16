package main

import (
	"github.com/Riki1312/nt-cli/internal/cli"
	"github.com/Riki1312/nt-cli/internal/output"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	cmd := cli.NewRootCmd(version, commit)
	if err := cmd.Execute(); err != nil {
		output.HandleError(err)
	}
}
