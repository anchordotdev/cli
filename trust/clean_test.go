package trust

import (
	"context"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/ui/uitest"
	"github.com/stretchr/testify/require"
)

func TestCmdTrustClean(t *testing.T) {
	cmd := CmdTrustClean
	cfg := cli.ConfigFromCmd(cmd)
	cfg.Test.SkipRunE = true

	t.Run("--help", func(t *testing.T) {
		cmdtest.TestOutput(t, cmd, "trust", "clean", "--help")
	})

	t.Run("--cert-states all", func(t *testing.T) {
		t.Skip()

		t.Cleanup(func() {
			cfg.Trust.Clean.States = []string{"expired"}
		})

		cmdtest.TestExecute(t, cmd, "trust", "clean", "--cert-states", "all")

		require.Equal(t, []string{"all"}, cfg.Trust.Clean.States)
	})

	t.Run("--organization test", func(t *testing.T) {
		t.Cleanup(func() {
			cfg.Trust.Org = ""
		})

		cmdtest.TestExecute(t, cmd, "trust", "clean", "--organization", "test")

		require.Equal(t, "test", cfg.Trust.Org)
	})

	t.Run("-o test", func(t *testing.T) {
		t.Cleanup(func() {
			cfg.Trust.Org = ""
		})

		cmdtest.TestExecute(t, cmd, "trust", "clean", "-o", "test")

		require.Equal(t, "test", cfg.Trust.Org)
	})

	t.Run("--realm test", func(t *testing.T) {
		t.Cleanup(func() {
			cfg.Trust.Realm = ""
		})

		cmdtest.TestExecute(t, cmd, "trust", "clean", "--realm", "test")

		require.Equal(t, "test", cfg.Trust.Realm)
	})

	t.Run("-r test", func(t *testing.T) {
		t.Cleanup(func() {
			cfg.Trust.Realm = ""
		})

		cmdtest.TestExecute(t, cmd, "trust", "clean", "-r", "test")

		require.Equal(t, "test", cfg.Trust.Realm)
	})

}

func TestClean(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.Trust.MockMode = true
	cfg.Trust.NoSudo = true
	cfg.Trust.Stores = []string{"mock"}
	var err error
	if cfg.API.Token, err = srv.GeneratePAT("anky@anchor.dev"); err != nil {
		t.Fatal(err)
	}
	ctx = cli.ContextWithConfig(ctx, cfg)

	t.Run("basics", func(t *testing.T) {
		if srv.IsProxy() {
			t.Skip("lcl clean unsupported in proxy mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cmd := Clean{}

		uitest.TestTUIOutput(ctx, t, cmd.UI())
	})
}
