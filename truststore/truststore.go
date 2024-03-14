package truststore

import (
	"crypto/subtle"
	"crypto/x509"
	"fmt"
	"io/fs"
	"os"
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
