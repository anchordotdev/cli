package cli

import (
	"context"
	"fmt"
	"go/build"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/anchordotdev/cli/models"
	"github.com/anchordotdev/cli/ui"
	"github.com/cli/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var Version = struct {
	Version, Commit, Date string

	Os, Arch string
}{
	Version: "dev",
	Commit:  "none",
	Date:    "unknown",
	Os:      runtime.GOOS,
	Arch:    runtime.GOARCH,
}

func IsDevVersion() bool {
	return Version.Version == "dev"
}

func UserAgent() string {
	return "Anchor CLI " + VersionString()
}

func ReleaseTagName() string {
	return fmt.Sprintf("v%s", Version.Version)
}

func VersionString() string {
	return fmt.Sprintf("%s (%s/%s) Commit: %s BuildDate: %s", Version.Version, Version.Os, Version.Arch, Version.Commit, Version.Date)
}

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
		Browserless bool `desc:"run as though browserless"`
		SkipRunE    bool `desc:"skip RunE for testing purposes"`
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

var (
	stackHexRegexp = regexp.MustCompile(`0x[0-9a-f]{2,}\??`)
	stackNilRegexp = regexp.MustCompile(`0x0`)

	stackPathReplacer *strings.Replacer
)

func init() {
	goPaths := strings.Split(os.Getenv("GOPATH"), string(os.PathListSeparator))
	if len(goPaths) == 0 {
		goPaths = append(goPaths, build.Default.GOPATH)
	}
	if goPaths[0] == "" {
		goPaths[0] = build.Default.GOPATH
	}

	joinedGoPaths := strings.Join(goPaths, ",<gopath>,") + ",<gopath>"
	replacements := strings.Split(joinedGoPaths, ",")
	replacements = append(replacements, runtime.GOROOT(), "<goroot>")
	
	if pwd, _ := os.Getwd(); pwd != "" {
		replacements = append(replacements, pwd, "<pwd>")
	}
	
	stackPathReplacer = strings.NewReplacer(replacements...)
}

func normalizeStack(stack string) string {
	// TODO: more nuanced replace for other known values like true/false, maybe empty string?
	stack = stackPathReplacer.Replace(stack)
	stack = stackHexRegexp.ReplaceAllString(stack, "<hex>")
	stack = stackNilRegexp.ReplaceAllString(stack, "<nil>")
	stack = strings.TrimRight(stack, "\n")

	lines := strings.Split(stack, "\n")
	for i, line := range lines {
		if strings.Contains(line, "<goroot>") {
			// for lines like: `<goroot>/src/runtime/debug/stack.go:24 +<hex>`
			lines[i] = fmt.Sprintf("%s:<line> +<hex>", strings.Split(line, ":")[0])
		}
		if strings.Contains(line, "in goroutine") {
			lines[i] = fmt.Sprintf("%s in gouroutine <int>", strings.Split(line, " in goroutine ")[0])
		}
	}

	return strings.Join(lines, "\n")
}

func ReportError(ctx context.Context, drv *ui.Driver, cmd *cobra.Command, args []string, msg any, stack string) {
	cfg := ConfigFromContext(ctx)

	var flags []string
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		flags = append(flags, flag.Name)
	})

	q := url.Values{}
	q.Add("title", fmt.Sprintf("Error: %s", msg))

	var body strings.Builder

	fmt.Fprintf(&body, "**Are there any additional details you would like to share?**\n")
	fmt.Fprintf(&body, "\n")
	fmt.Fprintf(&body, "---\n")
	fmt.Fprintf(&body, "\n")
	fmt.Fprintf(&body, "**Command:** `%s`\n", cmd.CalledAs())
	fmt.Fprintf(&body, "**Version:** `%s`\n", VersionString())
	fmt.Fprintf(&body, "**Arguments:** `[%s]`\n", strings.Join(args, ", "))
	fmt.Fprintf(&body, "**Flags:** `[%s]`\n", strings.Join(flags, ", "))
	if stack != "" {
		fmt.Fprintf(&body, "**Stack:**\n```\n%s\n```\n", normalizeStack(stack))
	}
	fmt.Fprintf(&body, "**Stdout:**\n```\n%s\n```\n", strings.TrimRight(string(drv.FinalOut()), "\n"))
	q.Add("body", body.String())

	reportErrorConfirmCh := make(chan struct{})
	drv.Activate(ctx, &models.ReportError{
		ConfirmCh: reportErrorConfirmCh,
		Cmd:       cmd,
		Args:      args,
		Msg:       msg,
	})

	if !cfg.NonInteractive {
		<-reportErrorConfirmCh
	}

	newIssueURL := fmt.Sprintf("https://github.com/anchordotdev/cli/issues/new?%s", q.Encode())
	// FIXME: ? remove config val, switch to mocking this to always err in tests
	if cfg.Test.Browserless {
		drv.Activate(ctx, &models.Browserless{Url: newIssueURL})
	} else {
		if err := browser.OpenURL(newIssueURL); err != nil {
			drv.Activate(ctx, &models.Browserless{Url: newIssueURL})
		}
	}
}
