package cli

import (
	"context"

	"github.com/muesli/termenv"
)

type Config struct {
	JSON           bool `desc:"Only print JSON output to STDOUT." flag:"json,j" env:"JSON_OUTPUT" toml:"json-output"`
	NonInteractive bool `desc:"Run without ever asking for user input." flag:"non-interactive,n" env:"NON_INTERACTIVE" toml:"non-interactive"`
	Verbose        bool `desc:"Verbose output." flag:"verbose,v" env:"VERBOSE" toml:"verbose"`

	API struct {
		URL   string `default:"https://api.anchor.dev/v0" desc:"Anchor API endpoint URL." flag:"api-url,u" env:"API_URL" json:"api_url" toml:"api-url"`
		Token string `desc:"Anchor API personal access token (PAT)." flag:"api-token,t" env:"API_TOKEN" json:"api_token" toml:"token"`
	}

	Trust struct {
		Org   string `desc:"organization" flag:"org,o" env:"ORG" json:"org" toml:"org"`
		Realm string `desc:"realm" flag:"realm,r" env:"REALM" json:"realm" toml:"realm"`

		NoSudo bool `desc:"Disable sudo prompts." flag:"no-sudo" env:"NO_SUDO" toml:"no-sudo"`

		MockMode bool `env:"ANCHOR_CLI_TRUSTSTORE_MOCK_MODE"`
	} `cmd:"trust"`

	User struct {
		Auth struct {
			SignIn struct {
				Email string `desc:"primary email address" flag:"email,e" env:"EMAIL" toml:"email"`
			} `cmd:"signin"`

			SignOut struct{} `cmd:"signout"`

			WhoAmI struct{} `cmd:"whoami"`
		} `cmd:"auth"`
	} `group:"user,user management" toml:"user"`

	Keyring struct {
		MockMode bool `env:"ANCHOR_CLI_KEYRING_MOCK_MODE"`
	}
}

type TUI struct {
	Run func(context.Context, termenv.File) error
}
