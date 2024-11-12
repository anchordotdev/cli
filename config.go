package cli

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"net"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/joeshaw/envdecode"
	"github.com/mcuadros/go-defaults"
	"github.com/mohae/deepcopy"
	"github.com/r3labs/diff/v3"
	"github.com/spf13/pflag"

	"github.com/anchordotdev/cli/toml"
)

type ConfigFetchFunc func(*Config) any

type Config struct {
	NonInteractive bool `env:"NON_INTERACTIVE" toml:",omitempty,readonly"`

	API struct {
		URL   string `default:"https://api.anchor.dev/v0" env:"API_URL" toml:"url,omitempty"`
		Token string `env:"API_TOKEN" toml:"api-token,omitempty,readonly"`
	} `toml:"api,omitempty"`

	File struct {
		Path string `default:"anchor.toml" env:"ANCHOR_CONFIG" toml:",omitempty,readonly"`
		Skip bool   `env:"ANCHOR_SKIP_CONFIG" toml:",omitempty,readonly"`
	} `toml:",omitempty,readonly"`

	Dashboard struct {
		URL string `default:"https://anchor.dev" env:"ANCHOR_HOST" toml:"url,omitempty"`
	} `toml:"dashboard,omitempty"`

	Lcl struct {
		LclHostURL string `default:"https://lcl.host" env:"LCL_HOST_URL" toml:",omitempty,readonly"`

		RealmAPID string `env:"REALM" toml:"realm-apid,omitempty"`

		Diagnostic struct {
			Addr      string `default:":4433" env:"DIAGNOSTIC_ADDR" toml:",omitempty"`
			Subdomain string `env:"DIAGNOSTIC_SUBDOMAIN" toml:",omitempty"`
		} `toml:",omitempty,readonly"`

		MkCert struct {
			Domains []string `flag:"domains" toml:",omitempty"`
			SubCa   string   `flag:"subca" toml:",omitempty"`
		} `toml:",omitempty,readonly"`
	} `toml:"lcl-host,omitempty"`

	Org struct {
		APID string `env:"ORG" toml:"apid,omitempty"`
		Name string `env:"ORG_NAME" toml:",omitempty,readonly"`
	} `toml:"org,omitempty"`

	Realm struct {
		APID string `env:"REALM"`
	} `toml:",omitempty,readonly"`

	Service struct {
		APID      string `env:"SERVICE" toml:"apid,omitempty"`
		Category  string `env:"SERVICE_CATEGORY" toml:"category,omitempty"`
		Framework string `env:"SERVICE_FRAMEWORK" toml:"framework,omitempty"`
		Name      string `env:"SERVICE_NAME" toml:",omitempty,readonly"`

		EnvOutput string `env:"ENV_OUTPUT" toml:",omitempty,readonly"`
		CertStyle string `env:"CERT_STYLE" toml:"cert-style,omitempty"`

		Verify struct {
			Timeout time.Duration `default:"2m" env:"VERIFY_TIMEOUT" toml:",omitempty,readonly"`
		} `toml:",omitempty,readonly"`
	} `toml:"service,omitempty"`

	Trust struct {
		NoSudo bool `flag:"no-sudo" env:"NO_SUDO" toml:",omitempty"`

		MockMode bool `env:"ANCHOR_CLI_TRUSTSTORE_MOCK_MODE" toml:",omitempty"`

		Stores []string `default:"[homebrew,nss,system]" env:"TRUST_STORES" toml:",omitempty"`

		Clean struct {
			States []string `default:"[expired]" env:"CERT_STATES" toml:",omitempty"`
		} `toml:",omitempty"`
	} `toml:",omitempty,readonly"`

	Keyring struct {
		MockMode bool `env:"ANCHOR_CLI_KEYRING_MOCK_MODE"`
	} `toml:",omitempty,readonly"`

	Test ConfigTest `fake:"-" toml:",omitempty,readonly"`

	Via struct {
		Defaults *Config `fake:"-" toml:",omitempty,readonly"`
		ENV      *Config `fake:"-" toml:",omitempty,readonly"`
		TOML     *Config `fake:"-" toml:",omitempty,readonly"`
	} `toml:",omitempty,readonly"`
}

type Dialer interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}

type DialFunc func(context.Context, string, string) (net.Conn, error)

func (fn DialFunc) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	return fn(ctx, network, address)
}

// values used for testing
type ConfigTest struct {
	Prefer map[string]ConfigTestPrefer // values for prism prefer header

	ACME struct {
		URL string
	}
	Browserless bool          // run as though browserless
	GOOS        string        // change OS identifier in tests
	ProcFS      fs.FS         // change the proc filesystem in tests
	LclHostPort int           // specify lcl host port in tests
	SkipRunE    bool          // skip RunE for testing purposes
	SystemFS    SystemFS      // change the system filesystem in tests
	Timestamp   time.Time     // timestamp to use/display in tests
	NetResolver *net.Resolver // DNS resolver for (some) tests
	NetDialer   Dialer        // TCP dialer for (some) tests
}

type ConfigTestPrefer struct {
	Code    int    `desc:"override response status"`
	Dynamic bool   `desc:"set dynamic mocking"`
	Example string `desc:"override example"`
}

func (c *Config) encodeTOML(w io.Writer) error {
	var cfg Config
	if err := cfg.setNonDefaults(c); err != nil {
		return err
	}
	return toml.NewEncoder[Config](w).Encode(cfg)
}

func (c *Config) GOOS() string {
	if goos := c.Test.GOOS; goos != "" {
		return goos
	}
	return runtime.GOOS
}

func (c *Config) AcmeURL(orgAPID string, realmAPID string, chainAPID string) string {
	baseURL := c.Test.ACME.URL
	if baseURL == "" {
		baseURL = c.Dashboard.URL
	}

	return baseURL + "/" + url.QueryEscape(orgAPID) + "/" + url.QueryEscape(realmAPID) + "/x509/" + url.QueryEscape(chainAPID) + "/acme"
}

func (c *Config) LclHostPort() *int {
	lclHostPort := c.Test.LclHostPort
	if lclHostPort == 0 {
		return nil
	}
	return &lclHostPort
}

func (c *Config) SetupGuideURL(orgAPID string, serviceAPID string) string {
	return c.Dashboard.URL + "/" + url.QueryEscape(orgAPID) + "/services/" + url.QueryEscape(serviceAPID) + "/guide"
}

func (c *Config) Copy() *Config {
	cfg := deepcopy.Copy(*c).(Config)
	// replace net-new objects with originals
	cfg.Test.SystemFS = c.Test.SystemFS

	return &cfg
}

func (c *Config) Load(ctx context.Context) error {
	if err := c.loadDefaults(); err != nil {
		return err
	}

	if err := c.loadTOML(c.SystemFS()); err != nil {
		return err
	}

	if err := c.loadENV(); err != nil {
		return err
	}

	if cfg := ConfigFromContext(ctx); cfg != nil {
		return c.setNonDefaults(cfg)
	}
	return nil
}

func (c *Config) ProcFS() fs.FS {
	if procFS := c.Test.ProcFS; procFS != nil {
		return procFS
	}
	return os.DirFS("/proc")
}

func (c *Config) SystemFS() SystemFS {
	if fs := c.Test.SystemFS; fs != nil {
		return fs
	}
	return osFS{}
}

func (c *Config) Timestamp() time.Time {
	if timestamp := c.Test.Timestamp; !timestamp.IsZero() {
		return timestamp
	}
	return time.Now().UTC()
}

func (c *Config) WriteTOML() error {
	var buf bytes.Buffer
	if err := c.encodeTOML(&buf); err != nil {
		return err
	}
	return c.SystemFS().WriteFile(c.File.Path, buf.Bytes(), 0644)
}

func (c *Config) loadDefaults() error {
	defaults.SetDefaults(c)
	c.Via.Defaults = defaultConfig()
	return nil
}

func (c *Config) loadENV() error {
	var cfg Config
	if err := envdecode.Decode(&cfg); err != nil && err != envdecode.ErrNoTargetFieldsAreSet {
		return err
	}
	if err := envdecode.Decode(c); err != nil && err != envdecode.ErrNoTargetFieldsAreSet {
		return err
	}
	c.Via.ENV = &cfg
	return nil
}

func (c *Config) loadTOML(fsys fs.FS) error {
	if path, ok := os.LookupEnv("ANCHOR_CONFIG"); ok {
		c.File.Path = path
	}
	if skip, ok := os.LookupEnv("ANCHOR_SKIP_CONFIG"); ok {
		c.File.Skip, _ = strconv.ParseBool(skip) // ignore errors
	}

	fs := pflag.NewFlagSet("", pflag.ContinueOnError)
	fs.StringVar(&c.File.Path, "config", Defaults.File.Path, "")
	fs.BoolVar(&c.File.Skip, "skip-config", Defaults.File.Skip, "")
	fs.ParseErrorsWhitelist.UnknownFlags = true
	fs.SetOutput(io.Discard)

	argv := os.Args
	for _, arg := range os.Args {
		if len(arg) > 0 && arg[0] == '-' {
			break
		}
		argv = argv[1:]
	}

	_ = fs.Parse(argv) // ignore errors

	if c.File.Skip {
		return nil
	}

	if f, err := fsys.Open(c.File.Path); err == nil {
		cfg := *Defaults
		if err := toml.NewDecoder(f).Decode(&cfg); err != nil {
			return err
		}
		if err := c.setNonDefaults(&cfg); err != nil {
			return err
		}
		c.Via.TOML = &cfg
	} else if c.File.Path != Defaults.File.Path {
		return err
	}
	return nil
}

func (c *Config) setNonDefaults(other *Config) error {
	changeLog, err := diff.Diff(Defaults, other)
	if err != nil {
		return err
	}

	_ = diff.Patch(changeLog, &c)
	return nil
}

func (c *Config) ViaSource(fetcher func(*Config) any) string {
	value := fetcher(c)

	if fetcher(c.Via.ENV) == value {
		return "env"
	}

	if c.Via.TOML != nil && fetcher(c.Via.TOML) == value {
		return c.File.Path
	}

	if fetcher(c.Via.Defaults) == value {
		return "default"
	}

	return "flag"
}

type SystemFS interface {
	Open(string) (fs.File, error)
	ReadDir(string) ([]fs.DirEntry, error)
	Stat(string) (fs.FileInfo, error)
	WriteFile(string, []byte, os.FileMode) error
}

// https://github.com/golang/go/issues/47803
type osFS struct{}

func (osFS) Open(name string) (fs.File, error)          { return os.Open(name) }
func (osFS) ReadDir(name string) ([]fs.DirEntry, error) { return os.ReadDir(name) }
func (osFS) Stat(name string) (fs.FileInfo, error)      { return os.Stat(name) }
func (osFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}
