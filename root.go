package cli

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/anchordotdev/cli/ui"
)

var CmdRoot = NewCmd[ShowHelp](nil, "anchor", func(cmd *cobra.Command) {})

type ShowHelp struct{}

func (c ShowHelp) UI() UI {
	return UI{
		RunTUI: c.RunTUI,
	}
}

func (c ShowHelp) RunTUI(ctx context.Context, drv *ui.Driver) error {
	return pflag.ErrHelp
}
