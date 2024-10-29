package version

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/clipboard"
	"github.com/anchordotdev/cli/ui"
	"github.com/anchordotdev/cli/version/models"
)

var CmdVersionUpgrade = cli.NewCmd[Upgrade](CmdVersion, "upgrade", func(cmd *cobra.Command) {})

type Upgrade struct {
	Clipboard cli.Clipboard
}

func (c Upgrade) UI() cli.UI {
	return cli.UI{
		RunTUI: c.runTUI,
	}
}

func (c Upgrade) runTUI(ctx context.Context, drv *ui.Driver) error {
	cli.SkipReleaseCheck = true

	drv.Activate(ctx, models.VersionUpgradeHeader)

	if c.Clipboard == nil {
		c.Clipboard = clipboard.System
	}

	command := "brew update && brew upgrade anchor"
	if isWindowsRuntime(cli.ConfigFromContext(ctx)) {
		command = "winget update Anchor.cli"
	}

	isUpgradeable, err := cli.IsUpgradeable(ctx)
	if err != nil {
		return err
	}

	clipboardErr := c.Clipboard.WriteAll(command)

	if isUpgradeable {
		drv.Activate(ctx, &models.VersionUpgrade{
			InClipboard: (clipboardErr == nil),
			Command:     command,
		})

		return nil
	}

	drv.Activate(ctx, &models.VersionUpgradeUnavailable)
	return nil
}
