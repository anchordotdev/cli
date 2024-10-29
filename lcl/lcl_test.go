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
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/api/apitest"
	"github.com/anchordotdev/cli/clipboard"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/trust"
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
		require.Equal(t, ":4433", cfg.Lcl.Diagnostic.Addr)
	})

	t.Run("-a :4444", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLcl, "-a", ":4444")
		require.Equal(t, ":4444", cfg.Lcl.Diagnostic.Addr)
	})

	t.Run("--addr :4455", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLcl, "--addr", ":4455")
		require.Equal(t, ":4455", cfg.Lcl.Diagnostic.Addr)
	})

	t.Run("DIAGNOSTIC_ADDR=:4466", func(t *testing.T) {
		t.Setenv("DIAGNOSTIC_ADDR", ":4466")

		cfg := cmdtest.TestCfg(t, CmdLcl)
		require.Equal(t, ":4466", cfg.Lcl.Diagnostic.Addr)
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

	t.Run("--category ruby", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLcl, "--category", "ruby")
		require.Equal(t, "ruby", cfg.Service.Category)
	})

	t.Run("--cert-style acme", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLcl, "--method", "acme")
		require.Equal(t, "acme", cfg.Service.CertStyle)
	})

	// alias

	t.Run("--language python", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLcl, "--language", "python")
		require.Equal(t, "python", cfg.Service.Category)
	})

	t.Run("--method acme", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLcl, "--method", "acme")
		require.Equal(t, "acme", cfg.Service.CertStyle)
	})
}

func TestLcl(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ensure lcl has no leftover data
	err := srv.RecreateUser("lcl")
	if err != nil {
		t.Fatal(err)
	}

	cfg := cmdtest.Config(ctx)
	if cfg.API.Token, err = srv.GeneratePAT("lcl@anchor.dev"); err != nil {
		t.Fatal(err)
	}
	cfg.API.URL = srv.URL
	cfg.Test.ACME.URL = "http://anchor.lcl.host:" + srv.RailsPort
	cfg.Lcl.Diagnostic.Subdomain = "hi-ankydotdev"
	cfg.Trust.MockMode = true
	cfg.Trust.NoSudo = true
	cfg.Trust.Stores = []string{"mock"}
	cfg.Test.Prefer = map[string]cli.ConfigTestPrefer{
		"/v0/orgs": {
			Example: "anky_personal",
		},
		"/v0/orgs/ankydotdev/realms": {
			Example: "anky_personal",
		},
		"/v0/orgs/ankydotdev/realms/localhost/x509/credentials": {
			Example: "anky_personal",
		},
	}
	ctx = cli.ContextWithConfig(ctx, cfg)

	_, diagPort, err := net.SplitHostPort(cfg.Lcl.Diagnostic.Addr)
	if err != nil {
		t.Fatal(err)
	}
	httpURL := "http://hello-world.lcl.host:" + diagPort
	httpsURL := "https://hello-world.lcl.host:" + diagPort

	setupGuideURL := cfg.SetupGuideURL("lcl", "test-app")

	t.Run("basics", func(t *testing.T) {
		if srv.IsMock() {
			t.Skip("lcl basic test unsupported in mock mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Command{
			clipboard: new(clipboard.Mock),
		}

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

		uitest.WaitForGoldenContains(t, drv, errc,
			"? What lcl.host domain would you like to use for diagnostics?",
		)

		tm.Type("hello-world")
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			fmt.Sprintf("! Press Enter to open %s in your browser.", httpURL),
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		if _, err := http.Get(httpURL); err != nil {
			t.Fatal(err)
		}

		uitest.WaitForGoldenContains(t, drv, errc,
			"! Press Enter to install missing certificates. (requires sudo)",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			fmt.Sprintf("! Press Enter to open %s in your browser.", httpsURL),
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

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

		// setup

		uitest.WaitForGoldenContains(t, drv, errc,
			"? What application server type?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? What is the application name?",
		)

		tm.Type("test-app")
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? What lcl.host domain would you like to use for local application development?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? How would you like to manage your lcl.host certificates?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			fmt.Sprintf("! Press Enter to open %s in your browser.", setupGuideURL),
		)

		anc, err := api.NewClient(ctx, cfg)
		if err != nil {
			t.Fatal(err)
		}

		srv, err := anc.GetService(ctx, "lcl", "test-app")
		if err != nil {
			t.Fatal(err)
		}

		lclUrl := fmt.Sprintf("https://test-app.lcl.host:%d", *srv.LocalhostPort)
		drv.Replace(lclUrl, "https://test-app.lcl.host:<service-port>")

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		uitest.TestGolden(t, drv.Golden())
	})

	// relies on diagnostic app existing and truststore updates from previous test
	t.Run("skip-diagnostic", func(t *testing.T) {
		if srv.IsMock() {
			t.Skip("lcl skip diagnostic test unsupported in mock mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Command{
			clipboard: new(clipboard.Mock),
		}

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

		uitest.WaitForGoldenContains(t, drv, errc,
			"? Which lcl/localhost service's lcl.host local development environment do you want to setup?",
		)

		uitest.TestGolden(t, drv.Golden())
		// just focus on initial portion to check for skip
	})

	t.Run(fmt.Sprintf("non-personal-cas-missing-%s", uitest.TestTagOS()), func(t *testing.T) {
		if !srv.IsMock() {
			t.Skip("lcl non-personal CAs missing test unsupported in mock mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cfg.Test.Prefer = map[string]cli.ConfigTestPrefer{
			"/v0/orgs": {
				Example: "anky_team",
			},
			"/v0/orgs/ankyco/realms": {
				Example: "ankyco_development",
			},
			"/v0/orgs/ankyco/realms/development/x509/credentials": {
				Example: "ankyco",
			},
			"/v0/orgs/ankyco/services/service-name/attachments": {
				Example: "ankyco_service",
			},
		}
		ctx = cli.ContextWithConfig(ctx, cfg)

		anc, err := api.NewClient(ctx, cfg)
		if err != nil {
			t.Fatal(err)
		}

		personalCAs, err := trust.FetchExpectedCAs(ctx, anc, "ankydotdev", "localhost")
		if err != nil {
			t.Fatal(err)
		}

		// pre-install ankydotdev/localhost CAs

		truststore.MockCAs = append(truststore.MockCAs, personalCAs...)

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Command{
			clipboard: new(clipboard.Mock),
		}

		errc := make(chan error, 1)
		go func() {
			if err := cmd.UI().RunTUI(ctx, drv); err != nil {
				errc <- err
			}
			if err := tm.Quit(); err != nil {
				errc <- err
			}
		}()

		uitest.WaitForGoldenContains(t, drv, errc,
			"! Press Enter to install missing certificates. (requires sudo)",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? Which organization do you want to manage your local development environment for?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? Which ankyco/development service's lcl.host local development environment do you want to setup?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? How would you like to manage your environment variables?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))

		uitest.TestGolden(t, drv.Golden())
	})
}
