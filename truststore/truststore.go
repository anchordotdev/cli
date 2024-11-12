package truststore

import (
	"crypto/sha1"
	"crypto/subtle"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"
)

type Store interface {
	Check() (bool, error)
	Description() string

	CheckCA(*CA) (bool, error)
	InstallCA(*CA) (bool, error)
	ListCAs() ([]*CA, error)
	UninstallCA(*CA) (bool, error)
}

type CA struct {
	*x509.Certificate

	FilePath   string
	NickName   string // only used by nss
	UniqueName string
}

func (c *CA) Equal(ca *CA) bool {
	return c.UniqueName == ca.UniqueName && subtle.ConstantTimeCompare(c.Raw, ca.Raw) == 1
}

var reWindowsThumbprintSpacer = regexp.MustCompile(".{8}")

func (c *CA) WindowsThumbprint() string {
	certSha := sha1.Sum(c.Raw)
	certHex := strings.ToUpper(hex.EncodeToString(certSha[:]))
	thumbprint := strings.TrimRight(reWindowsThumbprintSpacer.ReplaceAllString(certHex, "$0 "), " ")
	return thumbprint
}

func fatalErr(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}

func fatalCmdErr(err error, cmd string, out []byte) error {
	return fmt.Errorf("failed to execute \"%s\": %w\n\n%s\n", cmd, err, out)
}

func binaryExists(fs CmdFS, name string) bool {
	_, err := fs.LookPath(name)
	return err == nil
}

func pathExists(sfs fs.StatFS, path string) bool {
	_, err := sfs.Stat(strings.Trim(path, string(os.PathSeparator)))
	return err == nil
}

func parseCertificate(der []byte) (*x509.Certificate, error) {
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		if strings.HasPrefix(err.Error(), "x509: certificate contains duplicate extension") {
			return nil, nil
		}
		if strings.HasPrefix(err.Error(), "x509: inner and outer signature algorithm identifiers don't match") {
			return nil, nil
		}
		if strings.HasPrefix(err.Error(), "x509: negative serial number") {
			return nil, nil
		}
		return nil, err
	}
	return cert, nil
}
