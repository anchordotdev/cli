package trust

import (
	"bytes"
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/x/exp/teatest"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api/apitest"
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

func TestTrust(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.NonInteractive = true
	cfg.Trust.MockMode = true
	cfg.Trust.NoSudo = true
	cfg.Trust.Stores = []string{"mock"}

	var err error
	if cfg.API.Token, err = srv.GeneratePAT("anky@anchor.dev"); err != nil {
		t.Fatal(err)
	}

	t.Run("iterative basics", func(t *testing.T) {
		if !srv.IsProxy() {
			t.Skip("trust unsupported in mock mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := Command{
			Config: cfg,
		}

		if err := cmd.UI().RunTUI(ctx, drv); err != nil {
			t.Fatal(err)
		}
		defer tm.Quit()

		teatest.WaitFor(
			t, tm.Output(),
			func(bts []byte) bool {
				return bytes.Contains(bts, []byte("* Comparing local and expected CA certificates…*"))
			},
			teatest.WithCheckInterval(time.Millisecond*100),
			teatest.WithDuration(time.Second*3),
		)

		teatest.WaitFor(
			t, tm.Output(),
			func(bts []byte) bool {
				return bytes.Contains(bts, []byte("- Compared local and expected CA certificates: need to install 2 missing certificates.")) &&
					bytes.Contains(bts, []byte("! Installing 2 missing certificates. (requires sudo)")) &&
					bytes.Contains(bts, []byte("- Updated Mock: installed ankydotdev/localhost - AnchorCA [RSA, ECDSA]"))
			},
			teatest.WithCheckInterval(time.Millisecond*100),
			teatest.WithDuration(time.Second*3),
		)
	})

	t.Run("basics", func(t *testing.T) {
		t.Skip("pending fixing golden file based tests issues with resizing on CI")

		if !srv.IsProxy() {
			t.Skip("trust unsupported in mock mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cmd := Command{
			Config: cfg,
		}

		uitest.TestTUIOutput(ctx, t, cmd.UI())
	})
}
