package lcl

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui/uitest"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/stretchr/testify/require"
)

func TestCmdLclTrust(t *testing.T) {
	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdLclTrust, "lcl", "trust", "--help")
	})

	t.Run("default --trust-stores", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLclTrust)
		require.Equal(t, []string{"homebrew", "nss", "system"}, cfg.Trust.Stores)
	})

	t.Run("--trust-stores nss,system", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLclTrust, "--trust-stores", "nss,system")
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
	cfg.Test.Prefer = map[string]cli.ConfigTestPrefer{
		"/v0/orgs/org-slug/realms": {
			Example: "development",
		},
	}
	cfg.Trust.Stores = []string{"mock"}
	var err error
	if cfg.API.Token, err = srv.GeneratePAT("anky@anchor.dev"); err != nil {
		t.Fatal(err)
	}
	ctx = cli.ContextWithConfig(ctx, cfg)

	truststore.ResetMockCAs()
	t.Cleanup(truststore.ResetMockCAs)

	t.Run(fmt.Sprintf("basics-%s", uitest.TestTagOS()), func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Trust{}

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
