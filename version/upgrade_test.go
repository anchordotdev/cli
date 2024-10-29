package version

import (
	"context"
	"fmt"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/clipboard"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/ui/uitest"
)

func TestUpgrade(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := cmdtest.Config(ctx)
	ctx = cli.ContextWithConfig(ctx, cfg)

	t.Run(fmt.Sprintf("upgrade-available-%s", uitest.TestTagOS()), func(t *testing.T) {
		tagName := "vupgrade"
		cli.LatestRelease = &cli.Release{
			TagName: &tagName,
		}
		cmd := Upgrade{
			Clipboard: new(clipboard.Mock),
		}

		uitest.TestTUIOutput(ctx, t, cmd.UI())

		command, err := cmd.Clipboard.ReadAll()
		if err != nil {
			t.Fatal(err)
		}

		var want string
		if cfg.GOOS() == "windows" {
			want = "winget update Anchor.cli"
		} else {
			want = "brew update && brew upgrade anchor"
		}
		if got := command; want != got {
			t.Errorf("Want command:\n\n%q,\n\nGot:\n\n%q\n\n", want, got)
		}
	})

	t.Run("upgrade-unavailable", func(t *testing.T) {
		tagName := "vdev"
		cli.LatestRelease = &cli.Release{
			TagName: &tagName,
		}
		cmd := Upgrade{}

		uitest.TestTUIOutput(ctx, t, cmd.UI())
	})

}
