package lcl

import (
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
	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdLclConfig, "lcl", "config", "--help")
	})

	t.Run("default --addr", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLclConfig)
		require.Equal(t, ":4433", cfg.Lcl.DiagnosticAddr)
	})

	t.Run("-a :4444", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLclConfig, "-a", ":4444")
		require.Equal(t, ":4444", cfg.Lcl.DiagnosticAddr)
	})

	t.Run("--addr :4455", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLclConfig, "--addr", ":4455")
		require.Equal(t, ":4455", cfg.Lcl.DiagnosticAddr)
	})

	t.Run("ADDR=:4466", func(t *testing.T) {
		t.Setenv("ADDR", ":4466")

		cfg := cmdtest.TestCfg(t, CmdLclConfig)
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
			defer close(errc)

			if err := cmd.UI().RunTUI(ctx, drv); err != nil {
				errc <- err
				return
			}
			errc <- tm.Quit()
		}()

		// wait for prompt

		uitest.WaitForGoldenContains(t, drv, errc,
			"? What lcl.host domain would you like to use for diagnostics?",
		)

		tm.Type("hello-world")
		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		if !srv.IsProxy() {
			t.Skip("diagnostic unsupported in mock mode")
		}

		uitest.WaitForGoldenContains(t, drv, errc,
			fmt.Sprintf("! Press Enter to open %s in your browser.", httpURL),
		)

		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		if _, err := http.Get(httpURL); err != nil {
			t.Fatal(err)
		}

		uitest.WaitForGoldenContains(t, drv, errc,
			"! Press Enter to install missing certificates. (requires sudo)",
		)

		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		uitest.WaitForGoldenContains(t, drv, errc,
			fmt.Sprintf("! Press Enter to open %s in your browser.", httpsURL),
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
		uitest.TestGolden(t, drv.Golden())
	})
}
