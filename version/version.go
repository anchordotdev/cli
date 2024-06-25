package version

import (
	"fmt"
	"time"

	"github.com/Masterminds/semver"
	"github.com/atotto/clipboard"
	"github.com/google/go-github/v54/github"
	"github.com/spf13/cobra"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/ui"
)

func ReleaseCheck(cmd *cobra.Command, args []string) error {
	if cli.IsDevVersion() {
		return nil
	}

	ctx := cmd.Context()

	release, _, err := github.NewClient(nil).Repositories.GetLatestRelease(ctx, "anchordotdev", "cli")
	if err != nil {
		return nil
	}
	if publishedAt := release.PublishedAt.GetTime(); publishedAt != nil && time.Since(*publishedAt).Hours() < 24 {
		return nil
	}

	if release.TagName == nil || *release.TagName != cli.ReleaseTagName() {
		fmt.Println(ui.Header("New CLI Release"))
		fmt.Println(ui.StepHint("A new release of the anchor CLI is available."))

		command := "brew update && brew upgrade anchor"
		if isWindowsRuntime(cli.ConfigFromCmd(cmd)) {
			command = "winget update Anchor.cli"
		}

		if err := clipboard.WriteAll(command); err == nil {
			fmt.Println(ui.StepAlert(fmt.Sprintf("Copied %s to your clipboard.", ui.Announce(command))))
		}
		fmt.Println(ui.StepAlert(fmt.Sprintf("%s `%s` to update to the latest version.", ui.Action("Run"), ui.Emphasize(command))))
		fmt.Println(ui.StepHint(fmt.Sprintf("Not using homebrew? Explore other options here: %s", ui.URL("https://github.com/anchordotdev/cli"))))
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
					ui.Danger(fmt.Sprintf("This version of the Anchor CLI is out-of-date, please update.")),
				},
			},
		}
	}
	return nil
}
