package cli

import (
	"github.com/spf13/cobra"
)

var CmdRoot = NewCmd[ShowHelp](nil, "anchor", func(cmd *cobra.Command) {})

// ShowHelp calls cmd.HelpFunc() inside RunE instead of RunTUI

type ShowHelp struct{}

func (c ShowHelp) UI() UI {
	return UI{}
}
