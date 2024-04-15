package lcl

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
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
	flag.Parse()

	if err := srv.Start(context.Background()); err != nil {
		panic(err)
	}

	defer os.Exit(m.Run())

	srv.Close()
}

func TestCmdLcl(t *testing.T) {
	cmd := CmdLcl
	cfg := cli.ConfigFromCmd(cmd)
	cfg.Test.SkipRunE = true

	t.Run("--help", func(t *testing.T) {
		cmdtest.TestOutput(t, cmd, "lcl", "--help")
	})

	// config

	t.Run("default --addr", func(t *testing.T) {
		cmdtest.TestExecute(t, cmd, "lcl", "config")

		require.Equal(t, ":4433", cfg.Lcl.DiagnosticAddr)
	})

	t.Run("-a :4444", func(t *testing.T) {
		t.Cleanup(func() {
			cfg.Lcl.DiagnosticAddr = ":4433"
		})

		cmdtest.TestExecute(t, cmd, "lcl", "-a", ":4444")

		require.Equal(t, ":4444", cfg.Lcl.DiagnosticAddr)
	})

	t.Run("--addr :4455", func(t *testing.T) {
		t.Cleanup(func() {
			cfg.Lcl.DiagnosticAddr = ":4433"
		})

		cmdtest.TestExecute(t, cmd, "lcl", "--addr", ":4455")

		require.Equal(t, ":4455", cfg.Lcl.DiagnosticAddr)
	})

	t.Run("ADDR=:4466", func(t *testing.T) {
		t.Cleanup(func() {
			cfg.Lcl.DiagnosticAddr = ":4433"
		})

		t.Setenv("ADDR", ":4466")

		cmdtest.TestExecute(t, cmd, "lcl")

		require.Equal(t, ":4466", cfg.Lcl.DiagnosticAddr)
	})

	// mkcert

	t.Run("--domains test.lcl.host,test.localhost", func(t *testing.T) {
		t.Cleanup(func() {
			cfg.Lcl.MkCert.Domains = []string{}
		})

		cmdtest.TestExecute(t, cmd, "lcl", "--domains", "test.lcl.host,test.localhost")

		require.Equal(t, []string{"test.lcl.host", "test.localhost"}, cfg.Lcl.MkCert.Domains)
	})

	t.Run("--subca 1234:ABCD:EF123", func(t *testing.T) {
		t.Cleanup(func() {
			cfg.Lcl.MkCert.SubCa = ""
		})

		cmdtest.TestExecute(t, cmd, "lcl", "--subca", "1234:ABCD:EF123")

		require.Equal(t, "1234:ABCD:EF123", cfg.Lcl.MkCert.SubCa)
	})

	// setup

	t.Run("--language ruby", func(t *testing.T) {
		t.Cleanup(func() {
			cfg.Lcl.Setup.Language = ""
		})

		cmdtest.TestExecute(t, cmd, "lcl", "--language", "ruby")

		require.Equal(t, "ruby", cfg.Lcl.Setup.Language)
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

			tm.Quit()
		}()

		// wait for prompt

		teatest.WaitFor(
			t, drv.Out,
			func(bts []byte) bool {
				if len(errc) > 0 {
					t.Fatal(<-errc)
				}

				expect := "? What lcl.host domain would you like to use for diagnostics?"
				return bytes.Contains(bts, []byte(expect))
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
			teatest.WithDuration(time.Second*5),
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
