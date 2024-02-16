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
	"github.com/anchordotdev/cli/truststore"
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
	cfg.Trust.Stores = []string{"mock"}
	cfg.Trust.NoSudo = true

	var err error
	if cfg.API.Token, err = srv.GeneratePAT("example@example.com"); err != nil {
		t.Fatal(err)
	}

	header := "# Run `anchor trust`"
	subheaderPattern := regexp.MustCompile(`  # Installing "[^"]+ - AnchorCA" \w+ \([a-z0-9]+\) certificate$`)
	installPattern := regexp.MustCompile(`    - installed in the Mock store.$`)
	skipPattern := regexp.MustCompile(`    - skipped awaiting broader support.$`)

	t.Run("default to personal org and localhost realm", func(t *testing.T) {
		defer func() { truststore.MockCAs = nil }()

		cmd := &Command{
			Config: cfg,
		}

		buf, err := apitest.RunTTY(ctx, cmd.UI())
		if err != nil {
			t.Fatal(err)
		}

		scanner := bufio.NewScanner(buf)
		if !scanner.Scan() {
			t.Fatalf("want sudo warning line got %q %v (nil is EOF)", scanner.Err(), scanner.Err())
		}
		if line := scanner.Text(); line != header {
			t.Errorf("want output %q to match %q", line, header)
		}

		if !scanner.Scan() {
			t.Fatalf("want sudo warning line got %q %v (nil is EOF)", scanner.Err(), scanner.Err())
		}
		if line := scanner.Text(); line != "  ! "+sudoWarning {
			t.Errorf("want output %q to match %q", line, "  ! "+sudoWarning)
		}

		for scanner.Scan() {
			if line := scanner.Text(); !subheaderPattern.MatchString(line) {
				t.Errorf("want output %q to match %q", line, subheaderPattern)
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
		defer func() { truststore.MockCAs = nil }()

		cfg.Trust.Org = mustFetchPersonalOrgSlug(cfg)
		cfg.Trust.Realm = "localhost"

		cmd := &Command{
			Config: cfg,
		}

		buf, err := apitest.RunTTY(ctx, cmd.UI())
		if err != nil {
			t.Fatal(err)
		}

		scanner := bufio.NewScanner(buf)
		if !scanner.Scan() {
			t.Fatalf("want sudo warning line got %q %v (nil is EOF)", scanner.Err(), scanner.Err())
		}
		if line := scanner.Text(); line != header {
			t.Errorf("want output %q to match %q", line, header)
		}

		if !scanner.Scan() {
			t.Fatalf("want sudo warning line got %q %v (nil is EOF)", scanner.Err(), scanner.Err())
		}
		if line := scanner.Text(); line != "  ! "+sudoWarning {
			t.Errorf("want output %q to match %q", line, "  ! "+sudoWarning)
		}
		for scanner.Scan() {
			if line := scanner.Text(); !subheaderPattern.MatchString(line) {
				t.Errorf("want output %q to match %q", line, subheaderPattern)
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
	anc, err := api.NewClient(cfg)
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

	return userInfo.PersonalOrg.Slug
}
