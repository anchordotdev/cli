package version

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/ui"
	"github.com/anchordotdev/cli/version/models"
)

var CmdVersion = cli.NewCmd[Command](cli.CmdRoot, "version", func(cmd *cobra.Command) {})

type Command struct{}

func (c Command) UI() cli.UI {
	return cli.UI{
		RunTUI: c.runTUI,
	}
}

func (c Command) runTUI(ctx context.Context, drv *ui.Driver) error {
	drv.Activate(ctx, &models.Version{
		Arch:    info.arch,
		Commit:  info.commit,
		Date:    info.date,
		OS:      info.os,
		Version: info.version,
	})

	return nil
}
