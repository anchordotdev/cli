package lcl

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/stretchr/testify/require"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api/apitest"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui/uitest"
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

func TestCmdLcl(t *testing.T) {
	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdLcl, "lcl", "--help")
	})

	// config

	t.Run("default --addr", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLcl)
		require.Equal(t, ":4433", cfg.Lcl.DiagnosticAddr)
	})

	t.Run("-a :4444", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLcl, "-a", ":4444")
		require.Equal(t, ":4444", cfg.Lcl.DiagnosticAddr)
	})

	t.Run("--addr :4455", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLcl, "--addr", ":4455")
		require.Equal(t, ":4455", cfg.Lcl.DiagnosticAddr)
	})

	t.Run("ADDR=:4466", func(t *testing.T) {
		t.Setenv("ADDR", ":4466")

		cfg := cmdtest.TestCfg(t, CmdLcl)
		require.Equal(t, ":4466", cfg.Lcl.DiagnosticAddr)
	})

	// mkcert

	t.Run("--domains test.lcl.host,test.localhost", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLcl, "--domains", "test.lcl.host,test.localhost")
		require.Equal(t, []string{"test.lcl.host", "test.localhost"}, cfg.Lcl.MkCert.Domains)
	})

	t.Run("--subca 1234:ABCD:EF123", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLcl, "--subca", "1234:ABCD:EF123")
		require.Equal(t, "1234:ABCD:EF123", cfg.Lcl.MkCert.SubCa)
	})

	// setup

	t.Run("--language ruby", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLcl, "--language", "ruby")
		require.Equal(t, "ruby", cfg.Lcl.Setup.Language)
	})

	t.Run("--method anchor", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLcl, "--method", "anchor")
		require.Equal(t, "anchor", cfg.Lcl.Setup.Method)
	})
}

func TestLcl(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	diagAddr, err := apitest.UnusedPort()
	if err != nil {
		t.Fatal(err)
	}

	_, diagPort, err := net.SplitHostPort(diagAddr)
	if err != nil {
		t.Fatal(err)
	}

	httpURL := "http://hello-world.lcl.host:" + diagPort
	httpsURL := "https://hello-world.lcl.host:" + diagPort

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.AnchorURL = "http://anchor.lcl.host:" + srv.RailsPort + "/"
	cfg.Lcl.DiagnosticAddr = diagAddr
	cfg.Lcl.Service = "hi-ankydotdev"
	cfg.Lcl.Subdomain = "hi-ankydotdev"
	cfg.Trust.MockMode = true
	cfg.Trust.NoSudo = true
	cfg.Trust.Stores = []string{"mock"}
	if cfg.API.Token, err = srv.GeneratePAT("lcl@anchor.dev"); err != nil {
		t.Fatal(err)
	}
	ctx = cli.ContextWithConfig(ctx, cfg)

	setupGuideURL := cfg.AnchorURL + "lcl/services/test-app/guide"

	t.Run("basics", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Command{}

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)

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

		uitest.WaitForGoldenContains(t, drv, errc,
			"? What application server type?",
		)

		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? What is the application name?",
		)

		tm.Type("test-app")
		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? What lcl.host domain would you like to use for local application development?",
		)

		if !srv.IsProxy() {
			t.Skip("provisioning unsupported in mock mode")
		}

		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? What certificate management method?",
		)
		uitest.WaitForGoldenContains(t, drv, errc,
			"Automatic via ACME - Anchor style - Recommended",
		)
		uitest.WaitForGoldenContains(t, drv, errc,
			"Manually Managed - mkcert style",
		)

		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		t.Skip("Pending workaround for consistent setup guide port value")

		uitest.WaitForGoldenContains(t, drv, errc,
			fmt.Sprintf("! Press Enter to open %s.", setupGuideURL),
		)

		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		uitest.TestGolden(t, drv.Golden())
	})
}
