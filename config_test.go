package cli

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"testing/fstest"
	"unicode"

	"github.com/MakeNowJust/heredoc"
	"github.com/anchordotdev/cli/clitest"
	"github.com/anchordotdev/cli/toml"
	"github.com/brianvoe/gofakeit/v7"
)

func TestConfigLoadENV(t *testing.T) {
	tests := []struct {
		name string

		env map[string]string

		cfgFn func(*Config)
	}{
		{
			name: "empty-environment",

			env: map[string]string{},

			cfgFn: func(cfg *Config) {},
		},
		{
			name: "exhaustive",

			env: map[string]string{
				"ANCHOR_CLI_KEYRING_MOCK_MODE":    "true",
				"ANCHOR_CLI_TRUSTSTORE_MOCK_MODE": "true",
				"ANCHOR_CONFIG":                   "other-anchor.toml",
				"ANCHOR_HOST":                     "https://anchor.example.com",
				"ANCHOR_SKIP_CONFIG":              "true",
				"API_TOKEN":                       "s3cr3t!",
				"API_URL":                         "https://api.anchor.example.com/v0",
				"CERT_STATES":                     "valid",
				"CERT_STYLE":                      "acme",
				"DIAGNOSTIC_ADDR":                 "4321",
				"DIAGNOSTIC_SUBDOMAIN":            "ankydotdev",
				"ENV_OUTPUT":                      "dotenv",
				"LCL_HOST_URL":                    "https://lcl.host.example.com",
				"NON_INTERACTIVE":                 "true",
				"NO_SUDO":                         "true",
				"ORG":                             "test-org",
				"REALM":                           "test-realm",
				"SERVICE":                         "test-service",
				"SERVICE_CATEGORY":                "rubby",
				"SERVICE_FRAMEWORK":               "rubby-on-rails",
				"SERVICE_NAME":                    "test-rails-app",
				"TRUST_STORES":                    "mock",
			},

			cfgFn: func(cfg *Config) {
				cfg.File.Path = "other-anchor.toml"
				cfg.File.Skip = true
				cfg.API.URL = "https://api.anchor.example.com/v0"
				cfg.API.Token = "s3cr3t!"
				cfg.Dashboard.URL = "https://anchor.example.com"
				cfg.Lcl.LclHostURL = "https://lcl.host.example.com"
				cfg.Lcl.RealmAPID = "test-realm"
				cfg.Lcl.Diagnostic.Addr = "4321"
				cfg.Lcl.Diagnostic.Subdomain = "ankydotdev"
				cfg.NonInteractive = true
				cfg.Org.APID = "test-org"
				cfg.Realm.APID = "test-realm"
				cfg.Service.APID = "test-service"
				cfg.Service.Category = "rubby"
				cfg.Service.CertStyle = "acme"
				cfg.Service.EnvOutput = "dotenv"
				cfg.Service.Framework = "rubby-on-rails"
				cfg.Service.Name = "test-rails-app"
				cfg.Trust.NoSudo = true
				cfg.Trust.MockMode = true
				cfg.Trust.Stores = []string{"mock"}
				cfg.Trust.Clean.States = []string{"valid"}
				cfg.Keyring.MockMode = true
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for k, v := range test.env {
				t.Setenv(k, v)
			}

			var cfg Config
			if err := cfg.loadENV(); err != nil {
				t.Fatal(err)
			}

			var expected Config
			test.cfgFn(&expected)

			if want, got := expected, cfg; !reflect.DeepEqual(want, got) {
				t.Errorf("want env loaded config %#v, got %#v", want, got)
			}
		})
	}
}

func TestConfigLoadTOML(t *testing.T) {
	tests := []struct {
		name string

		toml string

		cfgFn func(*Config)
	}{
		{
			name: "defaults-empty",

			toml: "",

			cfgFn: func(cfg *Config) {},
		},
		{
			name: "service-acme-example",

			toml: heredoc.Doc(`
				[lcl-host]
				realm-apid = "localhost"

				[org]
				apid = "test-org"

				[service]
				apid = "test-service"
				category = "ruby"
				cert-style = "acme"
			`),

			cfgFn: func(cfg *Config) {
				cfg.Lcl.RealmAPID = "localhost"
				cfg.Org.APID = "test-org"
				cfg.Service.APID = "test-service"
				cfg.Service.Category = "ruby"
				cfg.Service.CertStyle = "acme"
			},
		},
	}

	defaultConfig := func() *Config {
		cfg := new(Config)
		*cfg = *Defaults
		return cfg
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := defaultConfig()

			fs := fstest.MapFS{
				"anchor.toml": &fstest.MapFile{
					Data: []byte(test.toml),
				},
			}

			if err := cfg.loadTOML(fs); err != nil {
				t.Fatal(err)
			}

			expected := defaultConfig()
			test.cfgFn(expected)

			if want, got := expected, cfg.TOML; !reflect.DeepEqual(want, got) {
				t.Errorf("want config from toml %#v, got %#v", want, got)
			}

			cfg.TOML = nil
			if want, got := expected, cfg; !reflect.DeepEqual(want, got) {
				t.Errorf("want combined config %#v, got %#v", want, got)
			}
		})
	}
}

func TestConfigEncodeTOML(t *testing.T) {
	tests := []struct {
		name string

		cfgFn func(*Config)

		toml string
	}{
		{
			name: "empty-when-only-readonlys-set",

			cfgFn: func(cfg *Config) {
				cfg.NonInteractive = true
				cfg.API.Token = "f00f"
				cfg.Lcl.LclHostURL = "https://lcl-host.example.com"
				cfg.Lcl.Diagnostic.Addr = ":4567"
				cfg.Lcl.Diagnostic.Subdomain = "not-anky"
				cfg.Lcl.MkCert.Domains = []string{"example.com"}
				cfg.Lcl.MkCert.SubCa = "f00f:f00f:f00f"
				cfg.Realm.APID = "not-localhost"
				cfg.Service.EnvOutput = "clipboard"
				cfg.Trust.NoSudo = true
				cfg.Trust.MockMode = true
				cfg.Trust.Stores = []string{"mock"}
				cfg.Trust.Clean.States = []string{"valid"}
				cfg.Keyring.MockMode = true
			},

			toml: "",
		},
		{
			name: "service-acme-example",

			cfgFn: func(cfg *Config) {
				cfg.Lcl.RealmAPID = "localhost"
				cfg.Org.APID = "test-org"
				cfg.Service.APID = "test-service"
				cfg.Service.Category = "ruby"
				cfg.Service.CertStyle = "acme"
			},

			toml: heredoc.Doc(`
				[lcl-host]
				realm-apid = 'localhost'

				[org]
				apid = 'test-org'

				[service]
				apid = 'test-service'
				category = 'ruby'
				cert-style = 'acme'
			`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var cfg Config
			test.cfgFn(&cfg)

			var buf bytes.Buffer
			if err := cfg.encodeTOML(&buf); err != nil {
				t.Fatal(err)
			}

			if want, got := test.toml, buf.String(); want != got {
				t.Errorf("want config toml %s, got %s", want, got)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string

		cfgFn func(*Config)

		toml string
	}{
		{
			name: "service-acme-example",

			cfgFn: func(cfg *Config) {
				cfg.Lcl.RealmAPID = "localhost"
				cfg.Org.APID = "test-org"
				cfg.Service.APID = "test-service"
				cfg.Service.Category = "ruby"
				cfg.Service.CertStyle = "acme"
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			want := *Defaults
			want.Test.SystemFS = clitest.TestFS{}

			test.cfgFn(&want)

			if err := want.WriteTOML(); err != nil {
				t.Fatal(err)
			}

			var got Config
			got.Test.SystemFS = want.Test.SystemFS

			if err := got.Load(context.Background()); err != nil {
				t.Fatal(err)
			}

			got.TOML = nil

			if !reflect.DeepEqual(want, got) {
				t.Errorf("want round-tripped config %+v, got %+v", want, got)
			}
		})
	}
}

func TestConfigFieldTags(t *testing.T) {
	var cfg Config
	if err := gofakeit.Struct(&cfg); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := cfg.encodeTOML(&buf); err != nil {
		t.Fatal(err)
	}

	var checkfn func(v map[string]any)

	checkfn = func(v map[string]any) {
		for k, v := range v {
			if unicode.IsUpper([]rune(k)[0]) {
				t.Errorf("config field %q is missing struct tags", k)
			}
			if m, ok := v.(map[string]any); ok {
				checkfn(m)
			}
		}
	}

	m := make(map[string]any)
	if err := toml.NewDecoder(&buf).Decode(&m); err != nil {
		t.Fatal(err)
	}

	checkfn(m)
}
