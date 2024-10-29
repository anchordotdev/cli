package version

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/spf13/cobra"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/ui"
)

func ReleaseCheck(cmd *cobra.Command, args []string) error {
	if cli.SkipReleaseCheck || cli.IsDevVersion() {
		return nil
	}

	ctx := cmd.Context()

	isFresh, err := cli.IsFreshLatestRelease(ctx)
	if err != nil {
		return err
	}
	if isFresh {
		return nil
	}

	isUpgradeable, err := cli.IsUpgradeable(ctx)
	if err != nil {
		return err
	}

	if isUpgradeable {
		fmt.Println(ui.Header("New CLI Release Available"))
		fmt.Println(ui.StepAlert("Run `anchor version upgrade` to upgrade."))
	}
	return nil
}

func isWindowsRuntime(cfg *cli.Config) bool {
	return cfg.GOOS() == "windows"
}

func MinimumVersionCheck(minimumVersion string) error {
	if cli.IsDevVersion() {
		return nil
	}

	minVersion, err := semver.NewVersion(minimumVersion)
	if err != nil {
		return nil // unexpected version string from the server
	}

	cliVersion, err := semver.NewVersion(cli.Version.Version)
	if err != nil {
		return nil
	}

	if cliVersion.LessThan(minVersion) {
		return ui.Error{
			Model: ui.Section{
				Name: "MinimumVersionCheck",
				Model: ui.MessageLines{
					ui.Header(
						fmt.Sprintf("%s This version of the Anchor CLI is out-of-date, please update.",
							ui.Danger("Error!"),
						),
					),
				},
			},
		}
	}
	return nil
}
