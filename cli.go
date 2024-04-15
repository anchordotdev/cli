package cli

import (
	"context"

	"github.com/muesli/termenv"

	"github.com/anchordotdev/cli/ui"
	"github.com/spf13/cobra"
)

type Config struct {
	JSON           bool `desc:"Only print JSON output to STDOUT." flag:"json,j" env:"JSON_OUTPUT" toml:"json-output"`
	NonInteractive bool `desc:"Run without ever asking for user input." flag:"non-interactive,n" env:"NON_INTERACTIVE" toml:"non-interactive"`
	Verbose        bool `desc:"Verbose output." flag:"verbose,v" env:"VERBOSE" toml:"verbose"`

	AnchorURL string `default:"https://anchor.dev" desc:"TODO" flag:"host" env:"ANCHOR_HOST" toml:"anchor-host"`

	API struct {
		URL   string `default:"https://api.anchor.dev/v0" desc:"Anchor API endpoint URL." flag:"api-url,u" env:"API_URL" json:"api_url" toml:"api-url"`
		Token string `desc:"Anchor API personal access token (PAT)." flag:"api-token,t" env:"API_TOKEN" json:"api_token" toml:"token"`
	}

	Lcl struct {
		Service   string `desc:"Name for lcl.host diagnostic service." flag:"service" env:"SERVICE" json:"service" toml:"service"`
		Subdomain string `desc:"Subdomain for lcl.host diagnostic service." flag:"subdomain" env:"SUBDOMAIN" json:"subdomain" toml:"subdomain"`

		DiagnosticAddr string `default:":4433" desc:"Local server address" flag:"addr,a" env:"ADDR" json:"address" toml:"address"`
		LclHostURL     string `default:"https://lcl.host" env:"LCL_HOST_URL"`

		Audit struct {
		} `cmd:"audit"`

		Clean struct {
		} `cmd:"clean"`

		Config struct {
		} `cmd:"config"`

		MkCert struct {
			Domains []string `flag:"domains"`
			SubCa   string   `flag:"subca"`
		} `cmd:"mkcert"`

		Setup struct {
			Language string `desc:"Language to use for integrating Anchor." flag:"language" json:"language" toml:"language"`
		} `cmd:"setup"`
	} `cmd:"lcl"`

	Test struct {
		SkipRunE bool `desc:"skip RunE for testing purposes"`
	}

	Trust struct {
		Org   string `desc:"organization" flag:"org,o" env:"ORG" json:"org" toml:"org"`
		Realm string `desc:"realm" flag:"realm,r" env:"REALM" json:"realm" toml:"realm"`

		NoSudo bool `desc:"Disable sudo prompts." flag:"no-sudo" env:"NO_SUDO" toml:"no-sudo"`

		MockMode bool `env:"ANCHOR_CLI_TRUSTSTORE_MOCK_MODE"`

		Stores []string `default:"[system,nss,homebrew]" desc:"trust stores" flag:"trust-stores" env:"TRUST_STORES" toml:"trust-stores"`

		Audit struct{} `cmd:"audit"`

		Clean struct {
			States []string `default:"[expired]" desc:"cert state(s)" flag:"cert-states" env:"CERT_STATES" toml:"cert-states"`
		} `cmd:"clean"`
	} `cmd:"trust"`

	User struct {
		Auth struct {
			SignIn struct{} `cmd:"signin"`

			SignOut struct{} `cmd:"signout"`

			WhoAmI struct{} `cmd:"whoami"`
		} `cmd:"auth"`
	} `group:"user,user management" toml:"user"`

	Keyring struct {
		MockMode bool `env:"ANCHOR_CLI_KEYRING_MOCK_MODE"`
	}

	Version struct{} `cmd:"version"`
}

type UI struct {
	RunTTY func(context.Context, termenv.File) error
	RunTUI func(context.Context, *ui.Driver) error
}

type ContextKey string

func ConfigFromContext(ctx context.Context) *Config {
	return ctx.Value(ContextKey("Config")).(*Config)
}

func ConfigFromCmd(cmd *cobra.Command) *Config {
	return ConfigFromContext(cmd.Context())
}

func ContextWithConfig(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, ContextKey("Config"), cfg)
}
