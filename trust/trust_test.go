package trust

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/api/apitest"
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
	cfg.Trust.MockMode = true
	cfg.Trust.NoSudo = true

	var err error
	if cfg.API.Token, err = srv.GeneratePAT("example@example.com"); err != nil {
		t.Fatal(err)
	}

	t.Run("default to personal org and localhost realm", func(t *testing.T) {
		cmd := &Command{
			Config: cfg,
		}

		buf, err := apitest.RunTUI(ctx, cmd.TUI())
		if err != nil {
			t.Fatal(err)
		}

		msgPattern := regexp.MustCompile(` - AnchorCA" \w+ cert \([a-z0-9]+\) installed in the mock store$`)

		for _, line := range strings.Split(buf.String(), "\n") {
			if len(line) == 0 {
				continue
			}

			if !msgPattern.MatchString(line) {
				t.Errorf("want output %q to match %q", line, msgPattern)
			}
		}
	})

	t.Run("specified org and realm", func(t *testing.T) {
		cfg.Trust.Org = mustFetchPersonalOrgSlug(cfg)
		cfg.Trust.Realm = "localhost"

		cmd := &Command{
			Config: cfg,
		}

		buf, err := apitest.RunTUI(ctx, cmd.TUI())
		if err != nil {
			t.Fatal(err)
		}

		msgPattern := regexp.MustCompile(` - AnchorCA" \w+ cert \([a-z0-9]+\) installed in the mock store$`)

		for _, line := range strings.Split(buf.String(), "\n") {
			if len(line) == 0 {
				continue
			}

			if !msgPattern.MatchString(line) {
				t.Errorf("want output %q to match %q", line, msgPattern)
			}
		}
	})
}

func mustFetchPersonalOrgSlug(cfg *cli.Config) string {
	anc, err := api.Client(cfg)
	if err != nil {
		panic(err)
	}

	res, err := anc.Get("")
	if err != nil {
		panic(err)
	}

	var userInfo *api.Root
	if err = json.NewDecoder(res.Body).Decode(&userInfo); err != nil {
		panic(err)
	}

	return *userInfo.PersonalOrg.Slug
}
