package auth

import (
	"context"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/ui/uitest"
)

func TestCmdAuthSignout(t *testing.T) {
	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdAuthSignout, "auth", "signout", "--help")
	})
}

func TestSignout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.Keyring.MockMode = true
	ctx = cli.ContextWithConfig(ctx, cfg)

	t.Run("signed-out", func(t *testing.T) {
		cmd := SignOut{}

		uitest.TestTUIOutput(ctx, t, cmd.UI())
	})

	t.Run("signed-in", func(t *testing.T) {
		t.Skip("pending singleton keyring")
		// kr := keyring.Keyring{}
		// if err := kr.Set(keyring.APIToken, "secret"); err != nil {
		// t.Fatal(err)
		// }
	})
}
