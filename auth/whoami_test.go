package auth

import (
	"context"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/ui/uitest"
)

func TestCmdAuthWhoAmI(t *testing.T) {
	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdAuthWhoami, "auth", "whoami", "--help")
	})

}

func TestWhoAmI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.Keyring.MockMode = true
	ctx = cli.ContextWithConfig(ctx, cfg)

	t.Run("signed-out", func(t *testing.T) {
		uitest.TestTUIError(ctx, t, new(WhoAmI).UI(), api.ErrSignedOut)
	})

	t.Run("signed-in", func(t *testing.T) {
		apiToken, err := srv.GeneratePAT("anky@anchor.dev")
		if err != nil {
			t.Fatal(err)
		}
		cfg.API.Token = apiToken

		cmd := WhoAmI{}

		uitest.TestTUIOutput(ctx, t, cmd.UI())
	})
}
