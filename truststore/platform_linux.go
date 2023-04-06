package truststore

import (
	"bytes"
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
)

var nssBrowsers = "Firefox and/or Chrome/Chromium"

type Platform struct {
	HomeDir string

	DataFS fs.StatFS
	SysFS  CmdFS

	inito                sync.Once
	nssBrowsers          string
	certutilInstallHelp  string
	trustFilenamePattern string
	trustCommand         []string
}

func firefoxProfiles(homeDir string) []string {
	return []string{
		filepath.Join(homeDir, "/.mozilla/firefox/*"),
		filepath.Join(homeDir, "/snap/firefox/common/.mozilla/firefox/*"),
	}
}

func certutilInstallHelp(sysFS CmdFS) string {
	switch {
	case binaryExists(sysFS, "apt"):
		return "apt install libnss3-tools"
	case binaryExists(sysFS, "yum"):
		return "yum install nss-tools"
	case binaryExists(sysFS, "zypper"):
		return "zypper install mozilla-nss-tools"
	}
	return ""
}

func (s *Platform) init() {
	s.inito.Do(func() {
		s.certutilInstallHelp = certutilInstallHelp(s.SysFS)

		switch {
		case pathExists(s.DataFS, "/etc/pki/ca-trust/source/anchors/"):
			s.trustFilenamePattern = "/etc/pki/ca-trust/source/anchors/%s.pem"
			s.trustCommand = []string{"update-ca-trust", "extract"}
		case pathExists(s.DataFS, "/usr/local/share/ca-certificates/"):
			s.trustFilenamePattern = "/usr/local/share/ca-certificates/%s.crt"
			s.trustCommand = []string{"update-ca-certificates"}
		case pathExists(s.DataFS, "/etc/ca-certificates/trust-source/anchors/"):
			s.trustFilenamePattern = "/etc/ca-certificates/trust-source/anchors/%s.crt"
			s.trustCommand = []string{"trust", "extract-compat"}
		case pathExists(s.DataFS, "/usr/share/pki/trust/anchors"):
			s.trustFilenamePattern = "/usr/share/pki/trust/anchors/%s.pem"
			s.trustCommand = []string{"update-ca-certificates"}
		}
	})
}

func (s *Platform) check() (bool, error) {
	s.init()

	if s.trustCommand == nil {
		return false, ErrUnsupportedDistro
	}

	return true, nil
}

func (s *Platform) installCA(ca *CA) (bool, error) {
	cert, err := ioutil.ReadFile(ca.FilePath)
	if err != nil {
		return false, fatalErr(err, "failed to read root certificate")
	}

	cmd := s.SysFS.Command("tee", s.trustFilenamePath(ca))
	cmd.Stdin = bytes.NewReader(cert)
	if out, err := s.SysFS.SudoExec(cmd); err != nil {
		return false, fatalCmdErr(err, "tee", out)
	}

	if out, err := s.SysFS.SudoExec(s.SysFS.Command(s.trustCommand[0], s.trustCommand[1:]...)); err != nil {
		return false, fatalCmdErr(err, strings.Join(s.trustCommand, " "), out)
	}

	return true, nil
}

func (s *Platform) uninstallCA(ca *CA) (bool, error) {
	cmd := s.SysFS.Command("rm", "-f", s.trustFilenamePath(ca))
	if out, err := s.SysFS.SudoExec(cmd); err != nil {
		return false, fatalCmdErr(err, "rm", out)
	}

	// We used to install under non-unique filenames.
	legacyFilename := fmt.Sprintf(s.trustFilenamePattern, "mkcert-rootCA")
	if pathExists(s.DataFS, legacyFilename) {
		cmd := s.SysFS.Command("rm", "-f", legacyFilename)
		if out, err := s.SysFS.SudoExec(cmd); err != nil {
			return false, fatalCmdErr(err, "rm (legacy filename)", out)
		}
	}

	cmd = s.SysFS.Command(s.trustCommand[0], s.trustCommand[1:]...)
	if out, err := s.SysFS.SudoExec(cmd); err != nil {
		return false, fatalCmdErr(err, strings.Join(s.trustCommand, " "), out)
	}

	return true, nil
}

func (s *Platform) trustFilenamePath(ca *CA) string {
	return fmt.Sprintf(s.trustFilenamePattern, strings.Replace(ca.UniqueName, " ", "_", -1))
}
