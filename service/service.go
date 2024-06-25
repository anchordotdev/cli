package service

import (
	"github.com/anchordotdev/cli"
	"github.com/spf13/cobra"
)

var CmdService = cli.NewCmd[cli.ShowHelp](cli.CmdRoot, "service", func(cmd *cobra.Command) {
	cmd.Hidden = true
})
