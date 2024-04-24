package lcl

import (
	"context"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/ui/uitest"
	"github.com/stretchr/testify/require"
)

func TestCmdLclMkCert(t *testing.T) {
	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdLclMkCert, "lcl", "mkcert", "--help")
	})

	t.Run("--domains test.lcl.host,test.localhost", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLclMkCert, "--domains", "test.lcl.host,test.localhost")
		require.Equal(t, []string{"test.lcl.host", "test.localhost"}, cfg.Lcl.MkCert.Domains)
	})

	t.Run("--subca 1234:ABCD:EF123", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdLclMkCert, "--subca", "1234:ABCD:EF123")
		require.Equal(t, "1234:ABCD:EF123", cfg.Lcl.MkCert.SubCa)
	})
}

func TestLclMkcert(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.AnchorURL = "http://anchor.lcl.host:" + srv.RailsPort
	cfg.Lcl.Service = "hi-lcl-mkcert"
	cfg.Trust.MockMode = true
	cfg.Trust.NoSudo = true
	cfg.Trust.Stores = []string{"mock"}
	var err error
	if cfg.API.Token, err = srv.GeneratePAT("lcl_mkcert@anchor.dev"); err != nil {
		t.Fatal(err)
	}
	ctx = cli.ContextWithConfig(ctx, cfg)

	t.Run("basics", func(t *testing.T) {
		t.Skip("pending better support for building needed models before running")

		if !srv.IsProxy() {
			t.Skip("mkcert unsupported in proxy mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cmd := MkCert{
			domains:         []string{"hi-lcl-mkcert.lcl.host", "hi-lcl-mkcert.localhost"},
			subCaSubjectUID: "ABCD:EF12:23456",
		}

		uitest.TestTUIOutput(ctx, t, cmd.UI())
	})
}
