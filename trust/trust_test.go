package trust

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"os"
	"regexp"
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

	headerPattern := regexp.MustCompile(`Installing "[^"]+ - AnchorCA" \w+ \([a-z0-9]+\) certificate:$`)
	installPattern := regexp.MustCompile(`  - installed in the mock store.$`)
	skipPattern := regexp.MustCompile(`  - skipped awaiting broader support.$`)

	t.Run("default to personal org and localhost realm", func(t *testing.T) {
		cmd := &Command{
			Config: cfg,
		}

		buf, err := apitest.RunTUI(ctx, cmd.TUI())
		if err != nil {
			t.Fatal(err)
		}

		scanner := bufio.NewScanner(buf)
		if !scanner.Scan() {
			t.Fatalf("want sudo warning line got %q %v (nil is EOF)", scanner.Err(), scanner.Err())
		}
		if line := scanner.Text(); line != sudoWarning {
			t.Errorf("want output %q to match %q", line, sudoWarning)
		}
		for scanner.Scan() {
			if line := scanner.Text(); !headerPattern.MatchString(line) {
				t.Errorf("want output %q to match %q", line, headerPattern)
			}

			if !scanner.Scan() {
				t.Fatalf("want detail line got %q %v (nil is EOF)", scanner.Err(), scanner.Err())
			}

			if line := scanner.Text(); !(installPattern.MatchString(line) || skipPattern.MatchString(line)) {
				t.Errorf("want output %q to match %q or %q", line, installPattern, skipPattern)
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

		scanner := bufio.NewScanner(buf)
		if !scanner.Scan() {
			t.Fatalf("want sudo warning line got %q %v (nil is EOF)", scanner.Err(), scanner.Err())
		}
		if line := scanner.Text(); line != sudoWarning {
			t.Errorf("want output %q to match %q", line, sudoWarning)
		}
		for scanner.Scan() {
			if line := scanner.Text(); !headerPattern.MatchString(line) {
				t.Errorf("want output %q to match %q", line, headerPattern)
			}

			if !scanner.Scan() {
				t.Fatalf("want detail line got %q %v (nil is EOF)", scanner.Err(), scanner.Err())
			}

			if line := scanner.Text(); !(installPattern.MatchString(line) || skipPattern.MatchString(line)) {
				t.Errorf("want output %q to match %q or %q", line, installPattern, skipPattern)
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
