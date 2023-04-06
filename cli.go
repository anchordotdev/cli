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
		URL   string `default:"https://api.anchor.dev/" desc:"Anchor API endpoint URL." flag:"api-url,u" env:"API_URL" json:"api_url" toml:"api-url"`
		Token string `desc:"Anchor API personal access token (PAT)." flag:"api-token,t" env:"API_TOKEN" json:"api_token" toml:"token"`
	}

	Trust struct {
		Target string `arg:"0..1" desc:"Organization, Realm, or CA"`

		NoSudo bool `desc:"Disable sudo prompts." flag:"no-sudo" env:"NO_SUDO" toml:"no-sudo"`
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
}

type TUI struct {
	Run func(context.Context, termenv.File) error
}
