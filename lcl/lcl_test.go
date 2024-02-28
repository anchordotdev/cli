package lcl

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api/apitest"
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

	if cfg.API.Token, err = srv.GeneratePAT("anky@anchor.dev"); err != nil {
		t.Fatal(err)
	}

	t.Run("basics", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Command{
			Config: cfg,
		}

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)

			tm.Quit()
		}()

		// wait for prompt

		teatest.WaitFor(
			t, tm.Output(),
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
			t, tm.Output(),
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
			t, tm.Output(),
			func(bts []byte) bool {
				if len(errc) > 0 {
					t.Fatal(<-errc)
				}

				expect := "! Press Enter to test " + httpURL + ". (without HTTPS)"
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
			t, tm.Output(),
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
			t, tm.Output(),
			func(bts []byte) bool {
				if len(errc) > 0 {
					t.Fatal(<-errc)
				}

				expect := "! Press Enter to test " + httpsURL + ". (with HTTPS)"
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
			t, tm.Output(),
			func(bts []byte) bool {
				if len(errc) > 0 {
					t.Fatal(<-errc)
				}

				expect := "- Scanned current directory."
				return bytes.Contains(bts, []byte(expect))
			},
			teatest.WithCheckInterval(time.Millisecond*100),
			teatest.WithDuration(time.Second*3),
		)

		t.Skip("pending ability to stub out the detection.Perform results")

		teatest.WaitFor(
			t, tm.Output(),
			func(bts []byte) bool {
				if len(errc) > 0 {
					t.Fatal(<-errc)
				}

				expect := "What is your application server type?"

				return bytes.Contains(bts, []byte(expect))
			},
			teatest.WithCheckInterval(time.Millisecond*100),
			teatest.WithDuration(time.Second*3),
		)

		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		teatest.WaitFor(
			t, tm.Output(),
			func(bts []byte) bool {
				if len(errc) > 0 {
					t.Fatal(<-errc)
				}

				expect := "? What is your application name?"
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
			t, tm.Output(),
			func(bts []byte) bool {
				if len(errc) > 0 {
					t.Fatal(<-errc)
				}

				expect := "? What lcl.host subdomain would you like to use?"
				return bytes.Contains(bts, []byte(expect))
			},
			teatest.WithCheckInterval(time.Millisecond*100),
			teatest.WithDuration(time.Second*3),
		)

		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		teatest.WaitFor(
			t, tm.Output(),
			func(bts []byte) bool {
				if len(errc) > 0 {
					if err != nil {
						t.Fatal(<-errc)
					}
				}

				expect := "- Created test-app resources for lcl.host diagnostic server on Anchor.dev."
				return bytes.Contains(bts, []byte(expect))
			},
			teatest.WithCheckInterval(time.Millisecond*1000),
			teatest.WithDuration(time.Second*30),
		)

		// TODO: add tests once thing settle down
	})
}
