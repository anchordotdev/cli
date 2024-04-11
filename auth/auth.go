package auth

import (
	"github.com/spf13/cobra"

	"github.com/anchordotdev/cli"
)

var CmdAuth = cli.NewCmd[cli.ShowHelp](cli.CmdRoot, "auth", func(cmd *cobra.Command) {
	cmd.Args = cobra.NoArgs
})
