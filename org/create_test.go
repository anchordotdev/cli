package org

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/ui/uitest"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestCmdOrg(t *testing.T) {
	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdOrgCreate, "org", "create", "--help")
	})

	t.Run("--org-name org", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdOrgCreate, "--org-name", "org")
		require.Equal(t, "org", cfg.Org.Name)
	})
}

func TestCreateOrg(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	var err error
	if cfg.API.Token, err = srv.GeneratePAT("anky@anchor.dev"); err != nil {
		t.Fatal(err)
	}
	ctx = cli.ContextWithConfig(ctx, cfg)

	t.Run("basics", func(t *testing.T) {
		if srv.IsProxy() {
			t.Skip("org create unsupported in proxy mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Create{}

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

		uitest.WaitForGoldenContains(t, drv, errc,
			"? What is the new organization's name?",
		)
		tm.Send(tea.KeyMsg{
			Runes: []rune("Org Name"),
			Type:  tea.KeyRunes,
		})
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		uitest.TestGolden(t, drv.Golden())
	})
}
