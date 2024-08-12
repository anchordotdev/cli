package trust

import (
	"context"
	"fmt"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api/apitest"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui/uitest"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/stretchr/testify/require"
)

var srv = &apitest.Server{
	Host:    "api.anchor.lcl.host",
	RootDir: "../..",
}

func TestMain(m *testing.M) {
	if err := srv.Start(context.Background()); err != nil {
		panic(err)
	}

	defer os.Exit(m.Run())

	srv.Close()
}

func TestCmdTrust(t *testing.T) {
	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdTrust, "trust", "--help")
	})

	t.Run("--no-sudo", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdTrust, "--no-sudo")
		require.Equal(t, true, cfg.Trust.NoSudo)
	})

	t.Run("--org testOrg", func(t *testing.T) {
		err := cmdtest.TestError(t, CmdTrust, "--org", "testOrg")
		require.ErrorContains(t, err, "if any flags in the group [org realm] are set they must all be set; missing [realm]")
	})

	t.Run("-o testOrg", func(t *testing.T) {
		err := cmdtest.TestError(t, CmdTrust, "-o", "testOrg")
		require.ErrorContains(t, err, "if any flags in the group [org realm] are set they must all be set; missing [realm]")
	})

	t.Run("-o testOrg -r testRealm", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdTrust, "-o", "testOrg", "-r", "testRealm")
		require.Equal(t, "testOrg", cfg.Org.APID)
		require.Equal(t, "testRealm", cfg.Realm.APID)
	})

	t.Run("--realm testRealm", func(t *testing.T) {
		err := cmdtest.TestError(t, CmdTrust, "--realm", "testRealm")
		require.ErrorContains(t, err, "if any flags in the group [org realm] are set they must all be set; missing [org]")
	})

	t.Run("-r testRealm", func(t *testing.T) {
		err := cmdtest.TestError(t, CmdTrust, "-r", "testRealm")
		require.ErrorContains(t, err, "if any flags in the group [org realm] are set they must all be set; missing [org]")
	})

	t.Run("default --trust-stores", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdTrust)
		require.Equal(t, []string{"homebrew", "nss", "system"}, cfg.Trust.Stores)
	})

	t.Run("--trust-stores nss,system", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdTrust, "--trust-stores", "nss,system")
		require.Equal(t, []string{"nss", "system"}, cfg.Trust.Stores)
	})
}

func TestTrust(t *testing.T) {
	if srv.IsProxy() {
		t.Skip("trust skipped in proxy mode to avoid golden conflicts")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.NonInteractive = true
	cfg.Trust.MockMode = true
	cfg.Trust.NoSudo = true
	cfg.Trust.Stores = []string{"mock"}
	var err error
	if cfg.API.Token, err = srv.GeneratePAT("anky@anchor.dev"); err != nil {
		t.Fatal(err)
	}
	ctx = cli.ContextWithConfig(ctx, cfg)

	t.Run(fmt.Sprintf("basics-%s", uitest.TestTagOS()), func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Command{}

		errc := make(chan error, 1)
		go func() {
			defer close(errc)

			if err := cmd.UI().RunTUI(ctx, drv); err != nil {
				errc <- err
				return
			}
			errc <- tm.Quit()
		}()

		uitest.WaitForGoldenContains(t, drv, errc,
			"! Press Enter to install missing certificates. (requires sudo)",
		)

		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		uitest.TestGolden(t, drv.Golden())
	})

	t.Run("noop", func(t *testing.T) {
		// truststore.MockCAs should still be set from basic

		cmd := Command{}

		drv, tm := uitest.TestTUI(ctx, t)

		errc := make(chan error, 1)
		go func() {
			defer close(errc)

			if err := cmd.UI().RunTUI(ctx, drv); err != nil {
				errc <- err
				return
			}
			errc <- tm.Quit()
		}()

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		uitest.TestGolden(t, drv.Golden())
	})

	t.Run("wsl-vm", func(t *testing.T) {
		truststore.ResetMockCAs()
		t.Cleanup(truststore.ResetMockCAs)

		cfg := *cfg
		cfg.Test.GOOS = "linux"
		cfg.Test.ProcFS = fstest.MapFS{
			"version": &fstest.MapFile{
				Data: unameWSL2,
			},
		}

		ctx, cancel := context.WithCancel(cli.ContextWithConfig(ctx, &cfg))
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Command{}

		errc := make(chan error, 1)
		go func() {
			defer close(errc)

			if err := cmd.UI().RunTUI(ctx, drv); err != nil {
				errc <- err
				return
			}
			errc <- tm.Quit()
		}()

		uitest.WaitForGoldenContains(t, drv, errc,
			"! Press Enter to install missing certificates. (requires sudo)",
		)

		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		uitest.TestGolden(t, drv.Golden())
	})
}
