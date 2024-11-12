package lcl

import (
	"context"
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/stretchr/testify/require"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/clipboard"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui/uitest"
)

func TestCmdLclSetup(t *testing.T) {
	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdLclSetup, "lcl", "setup", "--help")
	})

	t.Run("--category ruby", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLclSetup, "--category", "ruby")
		require.Equal(t, "ruby", cfg.Service.Category)
	})

	t.Run("--cert-style acme", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLclSetup, "--method", "acme")
		require.Equal(t, "acme", cfg.Service.CertStyle)
	})

	// alias

	t.Run("--language python", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLcl, "--language", "python")
		require.Equal(t, "python", cfg.Service.Category)
	})

	t.Run("--method acme", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLclSetup, "--method", "acme")
		require.Equal(t, "acme", cfg.Service.CertStyle)
	})

}

func TestSetup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ensure lcl_setup has no leftover data
	err := srv.RecreateUser("lcl_setup")
	if err != nil {
		t.Fatal(err)
	}

	cfg := cmdtest.Config(ctx)
	cfg.API.URL = srv.URL
	cfg.Test.ACME.URL = "http://anchor.lcl.host:" + srv.RailsPort
	cfg.Trust.MockMode = true
	cfg.Trust.NoSudo = true
	cfg.Trust.Stores = []string{"mock"}
	if cfg.API.Token, err = srv.GeneratePAT("lcl_setup@anchor.dev"); err != nil {
		t.Fatal(err)
	}
	ctx = cli.ContextWithConfig(ctx, cfg)

	truststore.ResetMockCAs()
	t.Cleanup(truststore.ResetMockCAs)

	setupGuideURL := cfg.SetupGuideURL("lcl_setup", "test-app")

	t.Run("create-service-automated-basics", func(t *testing.T) {
		if srv.IsMock() {
			t.Skip("lcl setup create service unsupported in mock mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Setup{
			clipboard: new(clipboard.Mock),
		}

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

		uitest.WaitForGoldenContains(t, drv, errc,
			"? Which organization's lcl.host local development environment do you want to setup?",
		)
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // select first option, "lcl_setup"

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

		{
			anc, err := api.NewClient(ctx, cfg)
			if err != nil {
				t.Fatal(err)
			}

			srv, err := anc.GetService(ctx, "lcl_setup", "test-app")
			if err != nil {
				t.Fatal(err)
			}

			lclUrl := fmt.Sprintf("https://test-app.lcl.host:%d", *srv.LocalhostPort)

			drv.Replace(lclUrl, "https://test-app.lcl.host:<service-port>")
		}

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		uitest.TestGolden(t, drv.Golden())

		if _, err := cfg.Test.SystemFS.Stat("anchor.toml"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("create-service-manual-basics", func(t *testing.T) {
		if srv.IsMock() {
			t.Skip("lcl setup create service unsupported in mock mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Setup{
			clipboard: new(clipboard.Mock),
		}

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

		uitest.WaitForGoldenContains(t, drv, errc,
			"? Which organization's lcl.host local development environment do you want to setup?",
		)
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // select first option, "lcl_setup"

		uitest.WaitForGoldenContains(t, drv, errc,
			"? Which lcl_setup/localhost service's lcl.host local development environment do you want to setup?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

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

		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		uitest.TestGolden(t, drv.Golden())

		if _, err := cfg.Test.SystemFS.Stat("anchor.toml"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run(fmt.Sprintf("existing-service-basics-%s", uitest.TestTagOS()), func(t *testing.T) {
		if srv.IsProxy() {
			t.Skip("lcl setup existing service unsupported in proxy mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Setup{
			clipboard: new(clipboard.Mock),
		}

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

		uitest.WaitForGoldenContains(t, drv, errc,
			"? Which organization's lcl.host local development environment do you want to setup?",
		)
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // select first option, "org-solo"

		uitest.WaitForGoldenContains(t, drv, errc,
			"? Which org-slug/realm-slug service's lcl.host local development environment do you want to setup?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? How would you like to manage your environment variables?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))

		// FIXME: check clipboard values for accuracy (can't easily access values)

		uitest.TestGolden(t, drv.Golden())
	})

	t.Run("create-service-with-parameterized-name", func(t *testing.T) {
		if srv.IsMock() {
			t.Skip("lcl setup create service unsupported in mock mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Setup{
			clipboard: new(clipboard.Mock),
		}

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

		uitest.WaitForGoldenContains(t, drv, errc,
			"? Which organization's lcl.host local development environment do you want to setup?",
		)
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // select first option, "lcl_setup"

		uitest.WaitForGoldenContains(t, drv, errc,
			"? Which lcl_setup/localhost service's lcl.host local development environment do you want to setup?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? What application server type?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? What is the application name?",
		)

		tm.Type("Test App")
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? What lcl.host domain would you like to use for local application development?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? How would you like to manage your lcl.host certificates?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		uitest.TestGolden(t, drv.Golden())

		if _, err := cfg.Test.SystemFS.Stat("anchor.toml"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("create-service-with-custom-domain", func(t *testing.T) {
		if srv.IsMock() {
			t.Skip("lcl setup create service unsupported in mock mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Setup{
			clipboard: new(clipboard.Mock),
		}

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

		uitest.WaitForGoldenContains(t, drv, errc,
			"? Which organization's lcl.host local development environment do you want to setup?",
		)
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // select first option, "lcl_setup"

		uitest.WaitForGoldenContains(t, drv, errc,
			"? Which lcl_setup/localhost service's lcl.host local development environment do you want to setup?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? What application server type?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? What is the application name?",
		)

		tm.Type("test-explicit-subdomain-app")
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? What lcl.host domain would you like to use for local application development?",
		)

		tm.Type("this-is-my-weird-subdomain")
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			"? How would you like to manage your lcl.host certificates?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		uitest.TestGolden(t, drv.Golden())

		if _, err := cfg.Test.SystemFS.Stat("anchor.toml"); err != nil {
			t.Fatal(err)
		}
	})
}
