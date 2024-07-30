package lcl

import (
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

	t.Run("--method anchor", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLclSetup, "--method", "anchor")
		require.Equal(t, "anchor", cfg.Lcl.Setup.Method)
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

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.AnchorURL = "http://anchor.lcl.host:" + srv.RailsPort
	cfg.Trust.MockMode = true
	cfg.Trust.NoSudo = true
	cfg.Trust.Stores = []string{"mock"}
	if cfg.API.Token, err = srv.GeneratePAT("lcl_setup@anchor.dev"); err != nil {
		t.Fatal(err)
	}
	ctx = cli.ContextWithConfig(ctx, cfg)

	setupGuideURL := cfg.AnchorURL + "lcl_setup/services/test-app/guide"

	t.Run("create-service-automated-basics", func(t *testing.T) {
		if srv.IsMock() {
			t.Skip("lcl setup create service unsupported in mock mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Setup{}

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

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

		// FIXME: partial golden test unless/until setup guide port fixed
		uitest.TestGolden(t, drv.Golden())
		t.Skip("Pending workaround for consistent setup guide port value")

		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		uitest.TestGolden(t, drv.Golden())
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

	t.Run("create-service-manual-basics", func(t *testing.T) {
		if srv.IsMock() {
			t.Skip("lcl setup create service unsupported in mock mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Setup{}

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

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
	})

	t.Run(fmt.Sprintf("existing-service-basics-%s", uitest.TestTagOS()), func(t *testing.T) {
		if srv.IsProxy() {
			t.Skip("lcl setup existing service unsupported in proxy mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Setup{}

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

		uitest.WaitForGoldenContains(t, drv, errc,
			"? Which org-slug/realm-slug service's lcl.host local development environment do you want to setup?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		uitest.WaitForGoldenContains(t, drv, errc,
			"! Press Enter to install missing certificates. (requires sudo)",
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

		cmd := Setup{}

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

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
	})

	t.Run("create-service-with-custom-domain", func(t *testing.T) {
		if srv.IsMock() {
			t.Skip("lcl setup create service unsupported in mock mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Setup{}

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

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
	})
}
