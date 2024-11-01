package service

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/stretchr/testify/require"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/clipboard"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/ui/uitest"
)

func TestCmdServiceEnv(t *testing.T) {
	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdServiceEnv, "service", "env", "--help")
	})

	t.Run("--method clipboard", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdServiceEnv, "--method", "clipboard")
		require.Equal(t, "clipboard", cfg.Service.EnvOutput)
	})

	t.Run("--org testOrg", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdServiceEnv, "--org", "testOrg")
		require.Equal(t, "testOrg", cfg.Org.APID)
	})

	t.Run("--realm testRealm", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdServiceEnv, "--realm", "testRealm")
		require.Equal(t, "testRealm", cfg.Realm.APID)
	})

	t.Run("--service testService", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdServiceEnv, "--service", "testService")
		require.Equal(t, "testService", cfg.Service.APID)
	})
}

func TestServiceEnv(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.Dashboard.URL = "http://anchor.lcl.host"
	cfg.API.URL = srv.URL
	var err error
	if cfg.API.Token, err = srv.GeneratePAT("anky@anchor.dev"); err != nil {
		t.Fatal(err)
	}
	ctx = cli.ContextWithConfig(ctx, cfg)

	t.Run("basics export", func(t *testing.T) {
		if srv.IsProxy() {
			t.Skip("service env unsupported in proxy mode")
		}

		drv, tm := uitest.TestTUI(ctx, t)
		cmd := Env{
			Clipboard: new(clipboard.Mock),
		}
		errc := make(chan error, 1)

		go func() {
			defer close(errc)

			if err := cmd.UI().RunTUI(ctx, drv); err != nil {
				errc <- err
				return
			}
			if err := tm.Quit(); err != nil {
				errc <- err
				return
			}
		}()

		uitest.WaitForGoldenContains(t, drv, errc,
			"? How would you like to manage your environment variables?",
		)

		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		if err := <-errc; err != nil {
			t.Fatal(err)
		}

		env, err := cmd.Clipboard.ReadAll()
		if err != nil {
			t.Fatal(err)
		}

		want := "export ACME_CONTACT=\"anky@anchor.dev\"\nexport ACME_DIRECTORY_URL=\"http://anchor.lcl.host/org-slug/realm-slug/x509/ca/acme\"\nexport ACME_HMAC_KEY=\"abcdefghijklmnopqrstuvwxyz0123456789-_ABCDEFGHIJKLMNOPQRSTUVWXYZ\"\nexport ACME_KID=\"aae_abcdefghijklmnopqrstuvwxyz0123456789-_ABCDEF\"\nexport HTTPS_PORT=\"4433\"\nexport SERVER_NAMES=\"service.lcl.host\"\n"
		if got := env; want != got {
			t.Errorf("Want env clipboard:\n\n%q,\n\nGot:\n\n%q\n\n", want, got)
		}

		uitest.TestGolden(t, drv.Golden())
	})

	t.Run("basics dotenv", func(t *testing.T) {
		if srv.IsProxy() {
			t.Skip("service env unsupported in proxy mode")
		}

		drv, tm := uitest.TestTUI(ctx, t)
		cmd := Env{
			Clipboard: new(clipboard.Mock),
		}
		errc := make(chan error, 1)

		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

		uitest.WaitForGoldenContains(t, drv, errc,
			"? How would you like to manage your environment variables?",
		)

		tm.Send(tea.KeyMsg{
			Type: tea.KeyDown,
		})
		tm.Send(tea.KeyMsg{
			Type: tea.KeyEnter,
		})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		if err := <-errc; err != nil {
			t.Fatal(err)
		}

		env, err := cmd.Clipboard.ReadAll()
		if err != nil {
			t.Fatal(err)
		}

		want := "ACME_CONTACT=\"anky@anchor.dev\"\nACME_DIRECTORY_URL=\"http://anchor.lcl.host/org-slug/realm-slug/x509/ca/acme\"\nACME_HMAC_KEY=\"abcdefghijklmnopqrstuvwxyz0123456789-_ABCDEFGHIJKLMNOPQRSTUVWXYZ\"\nACME_KID=\"aae_abcdefghijklmnopqrstuvwxyz0123456789-_ABCDEF\"\nHTTPS_PORT=\"4433\"\nSERVER_NAMES=\"service.lcl.host\"\n"
		if got := env; want != got {
			t.Errorf("Want .env contents:\n\n%q,\n\nGot:\n\n%q\n\n", want, got)
		}

		uitest.TestGolden(t, drv.Golden())
	})

	t.Run("basics display", func(t *testing.T) {
		if srv.IsProxy() {
			t.Skip("service env unsupported in proxy mode")
		}

		drv, tm := uitest.TestTUI(ctx, t)
		cmd := Env{
			Clipboard: new(clipboard.Mock),
		}
		errc := make(chan error, 1)

		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

		uitest.WaitForGoldenContains(t, drv, errc,
			"? How would you like to manage your environment variables?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		if err := <-errc; err != nil {
			t.Fatal(err)
		}

		uitest.TestGolden(t, drv.Golden())
	})

	t.Run("basics unattached service display", func(t *testing.T) {
		if srv.IsProxy() {
			t.Skip("service env unsupported in proxy mode")
		}

		cfg.Test.Prefer = map[string]cli.ConfigTestPrefer{
			"/v0/orgs/org-slug/services/service-name/attachments": {
				Example: "empty",
			},
		}
		ctx = cli.ContextWithConfig(ctx, cfg)

		drv, tm := uitest.TestTUI(ctx, t)
		cmd := Env{
			Clipboard: new(clipboard.Mock),
		}
		errc := make(chan error, 1)

		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

		uitest.WaitForGoldenContains(t, drv, errc,
			"? How would you like to manage your environment variables?",
		)

		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		if err := <-errc; err != nil {
			t.Fatal(err)
		}

		uitest.TestGolden(t, drv.Golden())
	})
}
