package cli

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/anchordotdev/cli/ui"
)

var CmdRoot = NewCmd[ShowHelp](nil, "anchor", func(cmd *cobra.Command) {
	cmd.Args = cobra.NoArgs

	// allow pass through of update arg for teatest golden tests
	cmd.Flags().Bool("update", false, "update .golden files")
	if err := cmd.Flags().MarkHidden("update"); err != nil {
		panic(err)
	}
	cmd.Flags().Bool("prism-proxy", false, "run prism in proxy mode")
	if err := cmd.Flags().MarkHidden("prism-proxy"); err != nil {
		panic(err)
	}
})

type ShowHelp struct{}

func (c ShowHelp) UI() UI {
	return UI{
		RunTUI: c.RunTUI,
	}
}

func (c ShowHelp) RunTUI(ctx context.Context, drv *ui.Driver) error {
	return pflag.ErrHelp
}
