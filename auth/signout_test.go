package auth

import (
	"context"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/ui/uitest"
)

func TestCmdAuthSignout(t *testing.T) {
	cmd := CmdAuthSignin
	cfg := cli.ConfigFromCmd(cmd)
	cfg.Test.SkipRunE = true

	t.Run("--help", func(t *testing.T) {
		cmdtest.TestOutput(t, cmd, "auth", "signout", "--help")
	})
}

func TestSignout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.Keyring.MockMode = true
	ctx = cli.ContextWithConfig(ctx, cfg)

	t.Run("signed-out", func(t *testing.T) {
		uitest.TestTUIError(ctx, t, new(SignOut).UI(), "secret not found in keyring")
	})

	t.Run("signed-in", func(t *testing.T) {
		t.Skip("pending singleton keyring")
		// kr := keyring.Keyring{}
		// if err := kr.Set(keyring.APIToken, "secret"); err != nil {
		// t.Fatal(err)
		// }
	})
}
