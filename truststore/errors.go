package truststore

import "errors"

var (
	ErrNoSudo = errors.New(`"sudo" is not available`)

	ErrNoKeytool = errors.New("no java keytool")

	ErrNoCertutil = errors.New("no certutil tooling")
	ErrNoNSS      = errors.New("no NSS browser")
	ErrNoNSSDB    = errors.New("no NSS database")
	ErrUnknownNSS = errors.New("unknown NSS install") // untested

	ErrUnsupportedDistro = errors.New("unsupported Linux distrobution")
)

type Op string

const (
	OpCheck     Op = "check"
	OpInstall      = "install"
	OpList         = "list"
	OpSudo         = "sudo"
	OpUninstall    = "uninstall"
)

type Error struct {
	Op

	Fatal   error
	Warning error
}

func (e Error) Error() string {
	if e.Fatal != nil {
		return e.Fatal.Error()
	}
	return e.Warning.Error()
}

type NSSError struct {
	Err error

	CertutilInstallHelp string
	NSSBrowsers         string
}

func (e NSSError) Error() string { return e.Err.Error() }

type PlatformError struct {
	Err error

	NSSBrowsers string
	RootCA      string
}

func (e PlatformError) Error() string { return e.Err.Error() }
