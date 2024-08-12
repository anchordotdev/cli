package cli_test

import (
	"reflect"
	"testing"

	"github.com/anchordotdev/cli"
	_ "github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/cmdtest"
	_ "github.com/anchordotdev/cli/lcl"
	_ "github.com/anchordotdev/cli/service"
	_ "github.com/anchordotdev/cli/testflags"
	_ "github.com/anchordotdev/cli/trust"
	_ "github.com/anchordotdev/cli/version"
)

func TestCmdRoot(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		cmdtest.TestHelp(t, cli.CmdRoot)
	})

	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, cli.CmdRoot, "--help")
	})

	// config

	tests := []struct {
		name string

		argv []string
		env  map[string]string

		want any
		get  func(*cli.Config) any
	}{
		// dashboard url
		{
			name: "default-dashboard-url",

			want: "https://anchor.dev",
			get:  func(cli *cli.Config) any { return cli.Dashboard.URL },
		},
		{
			name: "--dashboard-url",

			argv: []string{"--dashboard-url", "https://anchor.example.com"},

			want: "https://anchor.example.com",
			get:  func(cli *cli.Config) any { return cli.Dashboard.URL },
		},
		{
			name: "ANCHOR_HOST",

			env: map[string]string{"ANCHOR_HOST": "https://anchor.example.com"},

			want: "https://anchor.example.com",
			get:  func(cli *cli.Config) any { return cli.Dashboard.URL },
		},

		// api url
		{
			name: "default-api-url",

			want: "https://api.anchor.dev/v0",
			get:  func(cli *cli.Config) any { return cli.API.URL },
		},
		{
			name: "--api-url",

			argv: []string{"--api-url", "https://api.anchor.example.com"},

			want: "https://api.anchor.example.com",
			get:  func(cli *cli.Config) any { return cli.API.URL },
		},
		{
			name: "API_URL",

			env: map[string]string{"API_URL": "https://api.anchor.example.com"},

			want: "https://api.anchor.example.com",
			get:  func(cli *cli.Config) any { return cli.API.URL },
		},

		// api token
		{
			name: "default-api-token",

			want: "",
			get:  func(cli *cli.Config) any { return cli.API.Token },
		},
		{
			name: "--api-token",

			argv: []string{"--api-token", "f00f"},

			want: "f00f",
			get:  func(cli *cli.Config) any { return cli.API.Token },
		},
		{
			name: "API_TOKEN",

			env: map[string]string{"API_TOKEN": "f00f00f"},

			want: "f00f00f",
			get:  func(cli *cli.Config) any { return cli.API.Token },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for k, v := range test.env {
				t.Setenv(k, v)
			}

			cfg := cmdtest.TestCfg(t, cli.CmdRoot, test.argv...)

			if want, got := test.want, test.get(cfg); !reflect.DeepEqual(want, got) {
				t.Errorf("want dashboard host %s, got %s", want, got)
			}
		})
	}
}
