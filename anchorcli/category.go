package anchorcli

const (
	Deb   PackageFormat = "deb"
	Gem   PackageFormat = "gem"
	GoMod PackageFormat = "gomod"
	NPM   PackageFormat = "npm"
	SDist PackageFormat = "sdist"
)

const (
	SectionApplication Section = "application"
	SectionWebServer   Section = "web_server"
	SectionDatabase    Section = "database"
	SectionSystem      Section = "system"
)

type Section string

func (s Section) String() string { return string(s) }

type PackageFormat string

func (p PackageFormat) String() string { return string(p) }

type Category struct {
	ID          int           `json:"id"`
	Key         string        `json:"key"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Glyph       string        `json:"glyph"`
	Section     Section       `json:"section"`
	PkgFormat   PackageFormat `json:"pkgFormat"`
}

func (c Category) String() string { return string(c.Name) }

// Languages
var CategoryCustom = &Category{
	ID:          0,
	Key:         "custom",
	Name:        "Custom",
	Description: "A custom service",
	Glyph:       "code",
}

var CategoryGo = &Category{
	ID:          1,
	Key:         "go",
	Name:        "Go",
	Description: "A Go Application",
	Glyph:       "language-go",
	Section:     SectionApplication,
	PkgFormat:   GoMod,
}

var CategoryJavascript = &Category{
	ID:          2,
	Key:         "javascript",
	Name:        "Javascript",
	Description: "A JavaScript Application",
	Section:     SectionApplication,
	Glyph:       "language-javascript",
	PkgFormat:   NPM,
}

var CategoryPython = &Category{
	ID:          3,
	Key:         "python",
	Name:        "Python",
	Description: "A Python Application",
	Section:     SectionApplication,
	Glyph:       "language-python",
	PkgFormat:   SDist,
}

var CategoryRuby = &Category{
	ID:          4,
	Key:         "ruby",
	Name:        "Ruby",
	Description: "A Ruby Application",
	Section:     SectionApplication,
	Glyph:       "language-ruby",
	PkgFormat:   Gem,
}

// Web Servers
var CategoryApache = &Category{
	ID:          5,
	Key:         "apache",
	Name:        "Apache",
	Description: "An Apache Web Server",
	Section:     SectionWebServer,
}

var CategoryCaddy = &Category{
	ID:          6,
	Key:         "caddy",
	Name:        "Caddy",
	Description: "A Caddy Web Server",
	Section:     SectionWebServer,
	Glyph:       "caddy-logo",
}

var CategoryNginx = &Category{
	ID:          7,
	Key:         "nginx",
	Name:        "Nginx",
	Description: "An Nginx Web Server",
	Section:     SectionWebServer,
}

// Databases
var CategoryMonogoDB = &Category{
	ID:          8,
	Key:         "mongodb",
	Name:        "MongoDB",
	Description: "A MongoDB Database",
	Section:     SectionDatabase,
	Glyph:       "server-2",
}

var CategoryMySQL = &Category{
	ID:          9,
	Key:         "mysql",
	Name:        "MySQL",
	Description: "A MySQL Database",
	Section:     SectionDatabase,
	Glyph:       "server-2",
}

var CategoryPostgreSQL = &Category{
	ID:          10,
	Key:         "postgresql",
	Name:        "PostgreSQL",
	Description: "A PostgreSQL Database",
	Section:     SectionDatabase,
	Glyph:       "server-2",
}

// Systems/Browsers
var CategoryLocalhost = &Category{
	ID:          11,
	Key:         "localhost",
	Name:        "System",
	Description: "A lcl.host System",
	Section:     SectionSystem,
	Glyph:       "terminal",
}

var CategoryDebian = &Category{
	ID:          12,
	Key:         "debian",
	Name:        "Debian/Ubuntu",
	Description: "A Debian/Ubuntu System",
	Section:     SectionSystem,
	Glyph:       "debian-logo",
	PkgFormat:   Deb,
}

var CategoryDiagnostic = &Category{
	ID:          13,
	Key:         "diagnostic",
	Name:        "lcl.host Diagnostic",
	Description: "lcl.host Diagnostic System",
	Glyph:       "code",
}
