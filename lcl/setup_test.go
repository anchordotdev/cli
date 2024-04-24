package lcl

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/ui/uitest"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/stretchr/testify/require"
)

func TestCmdLclSetup(t *testing.T) {
	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdLclSetup, "lcl", "setup", "--help")
	})

	t.Run("--language ruby", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLclSetup, "--language", "ruby")
		require.Equal(t, "ruby", cfg.Lcl.Setup.Language)
	})
}

func TestSetup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.AnchorURL = "http://anchor.lcl.host:" + srv.RailsPort + "/"
	cfg.Lcl.Service = "hi-ankydotdev"
	cfg.Lcl.Subdomain = "hi-ankydotdev"
	cfg.Trust.MockMode = true
	cfg.Trust.NoSudo = true
	cfg.Trust.Stores = []string{"mock"}
	var err error
	if cfg.API.Token, err = srv.GeneratePAT("lcl_setup@anchor.dev"); err != nil {
		t.Fatal(err)
	}
	ctx = cli.ContextWithConfig(ctx, cfg)

	setupGuideURL := cfg.AnchorURL + "lcl_setup/services/test-app/guide"

	t.Run("basics", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Setup{}

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)

			tm.Quit()
		}()

		// wait for prompt

		teatest.WaitFor(
			t, drv.Out,
			func(bts []byte) bool {
				if len(errc) > 0 {
					t.Fatal(<-errc)
				}

				expect := "? What application server type?"
				return bytes.Contains(bts, []byte(expect))
			},
			teatest.WithCheckInterval(time.Millisecond*100),
			teatest.WithDuration(time.Second*3),
		)

		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		teatest.WaitFor(
			t, drv.Out,
			func(bts []byte) bool {
				if len(errc) > 0 {
					t.Fatal(<-errc)
				}

				expect := "? What is the application name?"
				return bytes.Contains(bts, []byte(expect))
			},
			teatest.WithCheckInterval(time.Millisecond*100),
			teatest.WithDuration(time.Second*3),
		)

		tm.Type("test-app")
		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		teatest.WaitFor(
			t, drv.Out,
			func(bts []byte) bool {
				if len(errc) > 0 {
					t.Fatal(<-errc)
				}

				expect := "? What lcl.host domain would you like to use for local application development?"
				return bytes.Contains(bts, []byte(expect))
			},
			teatest.WithCheckInterval(time.Millisecond*100),
			teatest.WithDuration(time.Second*3),
		)

		if !srv.IsProxy() {
			t.Skip("provisioning unsupported in mock mode")
		}

		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		t.Skip("Pending workaround for consistent setup guide port value")

		teatest.WaitFor(
			t, drv.Out,
			func(bts []byte) bool {
				if len(errc) > 0 {
					t.Fatal(<-errc)
				}

				expect := fmt.Sprintf("! Press Enter to open %s.", setupGuideURL)
				return bytes.Contains(bts, []byte(expect))
			},
			teatest.WithCheckInterval(time.Millisecond*100),
			teatest.WithDuration(time.Second*3),
		)

		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		teatest.RequireEqualOutput(t, drv.FinalOut())
	})
}
