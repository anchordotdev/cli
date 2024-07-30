package cli

import (
	"io/fs"
	"os"
	"runtime"
	"time"
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
		Org       string `desc:"Organization for lcl.host local development environment management." flag:"org,o" env:"ORG" json:"org" toml:"org"`
		Realm     string `desc:"Realm for lcl.host local development environment management." flag:"realm,r" env:"REALM" json:"realm" toml:"realm"`
		Service   string `desc:"Service for lcl.host local development environment management." flag:"service" env:"SERVICE" json:"service" toml:"service"`
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
			Method   string `desc:"Integration method for certificates."`
		} `cmd:"setup"`
	} `cmd:"lcl"`

	Service struct {
		Probe struct {
			Name string `desc:"service name"`

			Org   string `desc:"organization" flag:"org,o" env:"ORG" json:"org" toml:"org"`
			Realm string `desc:"realm" flag:"realm,r" env:"REALM" json:"realm" toml:"realm"`
		} `cmd:"probe"`

		Env struct {
			Method  string `desc:"Integration method for environment variables."`
			Org     string `desc:"organization" flag:"org,o" env:"ORG" json:"org" toml:"org"`
			Realm   string `desc:"realm" flag:"realm,r" env:"REALM" json:"realm" toml:"realm"`
			Service string `desc:"service" flag:"service,s" env:"SERVICE" json:"service" toml:"service"`
		} `cmd:"env"`
	} `cmd:"service"`

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

	Test ConfigTest
}

// values used for testing
type ConfigTest struct {
	Prefer map[string]ConfigTestPrefer `desc:"values for prism prefer header"`

	Browserless bool      `desc:"run as though browserless"`
	GOOS        string    `desc:"change OS identifier in tests"`
	ProcFS      fs.FS     `desc:"change the proc filesystem in tests"`
	SkipRunE    bool      `desc:"skip RunE for testing purposes"`
	Timestamp   time.Time `desc:"timestamp to use/display in tests"`
}

type ConfigTestPrefer struct {
	Code    int    `desc:"override response status"`
	Dynamic bool   `desc:"set dynamic mocking"`
	Example string `desc:"override example"`
}

func (c Config) GOOS() string {
	if goos := c.Test.GOOS; goos != "" {
		return goos
	}
	return runtime.GOOS
}

func (c Config) ProcFS() fs.FS {
	if procFS := c.Test.ProcFS; procFS != nil {
		return procFS
	}
	return os.DirFS("/proc")
}

func (c Config) Timestamp() time.Time {
	if timestamp := c.Test.Timestamp; !timestamp.IsZero() {
		return timestamp
	}
	return time.Now().UTC()
}
