package lcl

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/stretchr/testify/require"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui/uitest"
)

func TestCmdLclConfig(t *testing.T) {
	cmd := CmdLclConfig
	cfg := cli.ConfigFromCmd(cmd)
	cfg.Test.SkipRunE = true

	t.Run("--help", func(t *testing.T) {
		cmdtest.TestOutput(t, cmd, "lcl", "config", "--help")
	})

	t.Run("default --addr", func(t *testing.T) {
		t.Cleanup(func() {
			cfg.Lcl.DiagnosticAddr = ":4433"
		})

		cmdtest.TestExecute(t, cmd, "lcl", "config")

		require.Equal(t, ":4433", cfg.Lcl.DiagnosticAddr)
	})

	t.Run("-a :4444", func(t *testing.T) {
		t.Cleanup(func() {
			cfg.Lcl.DiagnosticAddr = ":4433"
		})

		cmdtest.TestExecute(t, cmd, "lcl", "config", "-a", ":4444")

		require.Equal(t, ":4444", cfg.Lcl.DiagnosticAddr)
	})

	t.Run("--addr :4455", func(t *testing.T) {
		t.Cleanup(func() {
			cfg.Lcl.DiagnosticAddr = ":4433"
		})

		cmdtest.TestExecute(t, cmd, "lcl", "config", "--addr", ":4455")

		require.Equal(t, ":4455", cfg.Lcl.DiagnosticAddr)
	})

	t.Run("ADDR=:4466", func(t *testing.T) {
		t.Cleanup(func() {
			cfg.Lcl.DiagnosticAddr = ":4433"
		})

		t.Setenv("ADDR", ":4466")

		cmdtest.TestExecute(t, cmd, "lcl", "config")

		require.Equal(t, ":4466", cfg.Lcl.DiagnosticAddr)
	})
}

func TestLclConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	diagPort := "4433"

	diagAddr := "0.0.0.0:" + diagPort
	httpURL := "http://hello-world.lcl.host:" + diagPort
	httpsURL := "https://hello-world.lcl.host:" + diagPort

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.AnchorURL = "http://anchor.lcl.host:" + srv.RailsPort
	cfg.Lcl.DiagnosticAddr = diagAddr
	cfg.Lcl.Service = "hi-ankydotdev"
	cfg.Lcl.Subdomain = "hi-ankydotdev"
	cfg.Trust.MockMode = true
	cfg.Trust.NoSudo = true
	cfg.Trust.Stores = []string{"mock"}
	var err error
	if cfg.API.Token, err = srv.GeneratePAT("lcl_config@anchor.dev"); err != nil {
		t.Fatal(err)
	}
	ctx = cli.ContextWithConfig(ctx, cfg)

	t.Run("basics", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := LclConfig{}

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

				return bytes.Contains(bts, []byte("? What lcl.host domain would you like to use for diagnostics?"))
			},
			teatest.WithCheckInterval(time.Millisecond*100),
			teatest.WithDuration(time.Second*3),
		)

		tm.Type("hello-world")
		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		if !srv.IsProxy() {
			t.Skip("diagnostic unsupported in mock mode")
		}

		teatest.WaitFor(
			t, drv.Out,
			func(bts []byte) bool {
				if len(errc) > 0 {
					t.Fatal(<-errc)
				}

				expect := "Entered hello-world.lcl.host domain for lcl.host diagnostic certificate."
				return bytes.Contains(bts, []byte(expect))
			},
			teatest.WithCheckInterval(time.Millisecond*100),
			teatest.WithDuration(time.Second*3),
		)

		teatest.WaitFor(
			t, drv.Out,
			func(bts []byte) bool {
				if len(errc) > 0 {
					t.Fatal(<-errc)
				}

				expect := fmt.Sprintf("! Press Enter to open %s in your browser.", httpURL)
				return bytes.Contains(bts, []byte(expect))
			},
			teatest.WithCheckInterval(time.Millisecond*100),
			teatest.WithDuration(time.Second*3),
		)

		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		if _, err := http.Get(httpURL); err != nil {
			t.Fatal(err)
		}

		teatest.WaitFor(
			t, drv.Out,
			func(bts []byte) bool {
				if len(errc) > 0 {
					t.Fatal(<-errc)
				}

				expect := "! Press Enter to install 2 missing certificates. (requires sudo)"
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

				expect := fmt.Sprintf("! Press Enter to open %s in your browser.", httpsURL)
				return bytes.Contains(bts, []byte(expect))
			},
			teatest.WithCheckInterval(time.Millisecond*100),
			teatest.WithDuration(time.Second*3),
		)

		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		pool := x509.NewCertPool()
		for _, ca := range truststore.MockCAs {
			pool.AddCert(ca.Certificate)
		}

		httpsClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: pool,
				},
			},
		}

		if _, err := httpsClient.Get(httpsURL); err != nil {
			t.Fatal(err)
		}

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		teatest.RequireEqualOutput(t, drv.FinalOut())
	})
}
