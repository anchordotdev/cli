package truststore

import (
	"crypto/x509"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"sync"
)

type CA struct {
	*x509.Certificate

	FilePath   string
	UniqueName string
}

type Store struct {
	CAROOT string
	HOME   string

	DataFS fs.StatFS
	SysFS  CmdFS

	hasNSS       bool
	hasCertutil  bool
	certutilPath string
	initNSSOnce  sync.Once

	systemTrustFilenameTemplate string
	systemTrustCommand          []string
	certutilInstallHelp         string
	nssBrowsers                 string

	hasJava    bool
	hasKeytool bool

	javaHome    string
	cacertsPath string
	keytoolPath string
}

func (s *Store) binaryExists(name string) bool {
	_, err := s.SysFS.LookPath(name)
	return err == nil
}

func (s *Store) pathExists(path string) bool {
	_, err := s.DataFS.Stat(strings.Trim(path, string(os.PathSeparator)))
	return err == nil
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
