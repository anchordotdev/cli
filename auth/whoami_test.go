package auth

import (
	"context"
	"runtime"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/api/apitest"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/stretchr/testify/require"
)

func TestCmdAuthWhoAmI(t *testing.T) {
	cmd := CmdAuthWhoami
	cfg := cli.ConfigFromCmd(cmd)
	cfg.Test.SkipRunE = true
	root := cmd.Root()

	t.Run("--help", func(t *testing.T) {
		cmdtest.TestOutput(t, root, "auth", "whoami", "--help")
	})

}

func TestWhoAmI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no pty support on windows")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.Keyring.MockMode = true
	ctx = cli.ContextWithConfig(ctx, cfg)

	t.Run("signed-out", func(t *testing.T) {
		_, err := apitest.RunTTY(ctx, new(WhoAmI).UI())
		require.Error(t, err, api.ErrSignedOut)
	})

	t.Run("signed-in", func(t *testing.T) {
		apiToken, err := srv.GeneratePAT("anky@anchor.dev")
		if err != nil {
			t.Fatal(err)
		}
		cfg.API.Token = apiToken

		buf, err := apitest.RunTTY(ctx, new(WhoAmI).UI())
		if err != nil {
			t.Fatal(err)
		}

		if want, got := "Hello anky@anchor.dev!\n", buf.String(); want != got {
			t.Errorf("want output %q, got %q", want, got)
		}
	})
}
