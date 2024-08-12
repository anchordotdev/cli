package cli

import (
	"github.com/spf13/cobra"
)

var CmdRoot = NewCmd[ShowHelp](nil, "anchor", func(cmd *cobra.Command) {
	cfg := ConfigFromCmd(cmd)

	cmd.PersistentFlags().StringVar(&cfg.API.Token, "api-token", Defaults.API.Token, "Anchor API personal access token (PAT).")
	cmd.PersistentFlags().StringVar(&cfg.API.URL, "api-url", Defaults.API.URL, "Anchor API endpoint URL.")
	cmd.PersistentFlags().StringVar(&cfg.File.Path, "config", Defaults.File.Path, "Service configuration file.")
	cmd.PersistentFlags().StringVar(&cfg.Dashboard.URL, "dashboard-url", Defaults.Dashboard.URL, "Anchor dashboard URL.")
	cmd.PersistentFlags().BoolVar(&cfg.File.Skip, "skip-config", Defaults.File.Skip, "Skip loading configuration file.")

	if err := cmd.PersistentFlags().MarkHidden("api-url"); err != nil {
		panic(err)
	}
	if err := cmd.PersistentFlags().MarkHidden("dashboard-url"); err != nil {
		panic(err)
	}
})

// ShowHelp calls cmd.HelpFunc() inside RunE instead of RunTUI

type ShowHelp struct{}

func (c ShowHelp) UI() UI {
	return UI{}
}
