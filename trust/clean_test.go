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
	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdTrustClean, "trust", "clean", "--help")
	})

	t.Run("--cert-states all", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdTrustClean, "--cert-states", "all")
		require.Equal(t, []string{"all"}, cfg.Trust.Clean.States)
	})

	t.Run("--org testOrg", func(t *testing.T) {
		err := cmdtest.TestError(t, CmdTrustClean, "--org", "testOrg")
		require.ErrorContains(t, err, "if any flags in the group [org realm] are set they must all be set; missing [realm]")
	})

	t.Run("-o testOrg", func(t *testing.T) {
		err := cmdtest.TestError(t, CmdTrustClean, "-o", "testOrg")
		require.ErrorContains(t, err, "if any flags in the group [org realm] are set they must all be set; missing [realm]")
	})

	t.Run("-o testOrg -r testRealm", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdTrustClean, "-o", "testOrg", "-r", "testRealm")
		require.Equal(t, "testOrg", cfg.Trust.Org)
		require.Equal(t, "testRealm", cfg.Trust.Realm)
	})

	t.Run("--realm testRealm", func(t *testing.T) {
		err := cmdtest.TestError(t, CmdTrustClean, "--realm", "testRealm")
		require.ErrorContains(t, err, "if any flags in the group [org realm] are set they must all be set; missing [org]")
	})

	t.Run("-r testRealm", func(t *testing.T) {
		err := cmdtest.TestError(t, CmdTrustClean, "-r", "testRealm")
		require.ErrorContains(t, err, "if any flags in the group [org realm] are set they must all be set; missing [org]")
	})

	t.Run("default --trust-stores", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdTrustClean)
		require.Equal(t, []string{"homebrew", "nss", "system"}, cfg.Trust.Stores)
	})

	t.Run("--trust-stores nss,system", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdTrustClean, "--trust-stores", "nss,system")
		require.Equal(t, []string{"nss", "system"}, cfg.Trust.Stores)
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
