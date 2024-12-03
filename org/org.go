package org

import (
	"github.com/anchordotdev/cli"
	"github.com/spf13/cobra"
)

var CmdOrg = cli.NewCmd[cli.ShowHelp](cli.CmdRoot, "org", func(cmd *cobra.Command) {
})
