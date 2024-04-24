package version

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/anchordotdev/cli/ui"
	"github.com/atotto/clipboard"
	"github.com/google/go-github/v54/github"
	"github.com/spf13/cobra"
)

var info = struct {
	version, commit, date string

	os, arch string
}{
	version: "dev",
	commit:  "none",
	date:    "unknown",
	os:      runtime.GOOS,
	arch:    runtime.GOARCH,
}

func Set(version, commit, date string) {
	info.version = version
	info.commit = commit
	info.date = date
}

func String() string {
	return fmt.Sprintf("%s (%s/%s) Commit: %s BuildDate: %s", info.version, info.os, info.arch, info.commit, info.date)
}

func UserAgent() string {
	return "Anchor CLI " + String()
}

func VersionCheck(cmd *cobra.Command, args []string) error {
	if info.version == "dev" {
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

	if release.TagName == nil || *release.TagName != "v"+info.version {
		fmt.Println(ui.StepHint("A new release of the anchor CLI is available."))
		if !isWindowsRuntime() {
			command := "brew update && brew upgrade anchor"
			if err := clipboard.WriteAll(command); err == nil {
				fmt.Println(ui.StepAlert(fmt.Sprintf("Copied %s to your clipboard.", ui.Announce(command))))
			}
			fmt.Println(ui.StepAlert(fmt.Sprintf("%s `%s` to update to the latest version.", ui.Action("Run"), ui.Emphasize(command))))
			fmt.Println(ui.StepHint(fmt.Sprintf("Not using homebrew? Explore other options here: %s", ui.URL("https://github.com/anchordotdev/cli"))))
			fmt.Println()
		} else {
			// TODO(amerine): Add chocolatey instructions.
		}
	}
	return nil
}

func isWindowsRuntime() bool {
	return os.Getenv("GOOS") == "windows" || runtime.GOOS == "windows"
}
