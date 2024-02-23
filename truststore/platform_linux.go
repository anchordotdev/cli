package truststore

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

var nssBrowsers = "Firefox and/or Chrome/Chromium"

type Platform struct {
	HomeDir string

	DataFS DataFS
	SysFS  CmdFS

	inito                sync.Once
	nssBrowsers          string
	certutilInstallHelp  string
	trustFilenamePattern string
	trustCommand         []string
	caBundleFileName     string
}

func firefoxProfiles(homeDir string) []string {
	return []string{
		filepath.Join(homeDir, "/.mozilla/firefox/*"),
		filepath.Join(homeDir, "/.mozilla/firefox-esr/*"),
		filepath.Join(homeDir, "/.mozilla/firefox-trunk/*"),
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

func (s *Platform) Description() string { return "System (Linux)" }

func (s *Platform) init() {
	s.inito.Do(func() {
		s.certutilInstallHelp = certutilInstallHelp(s.SysFS)

		switch {
		case pathExists(s.DataFS, "/etc/pki/ca-trust/source/anchors/"):
			s.trustFilenamePattern = "/etc/pki/ca-trust/source/anchors/%s.pem"
			s.caBundleFileName = "/etc/pki/tls/certs/ca-bundle.crt"
			s.trustCommand = []string{"update-ca-trust", "extract"}
		case pathExists(s.DataFS, "/usr/local/share/ca-certificates/"):
			s.trustFilenamePattern = "/usr/local/share/ca-certificates/%s.crt"
			s.caBundleFileName = "/etc/ssl/certs/ca-certificates.crt"
			s.trustCommand = []string{"update-ca-certificates"}
		case pathExists(s.DataFS, "/etc/ca-certificates/trust-source/anchors/"):
			s.trustFilenamePattern = "/etc/ca-certificates/trust-source/anchors/%s.crt"
			s.caBundleFileName = "/etc/ssl/certs/ca-certificates.crt"
			s.trustCommand = []string{"trust", "extract-compat"}
		case pathExists(s.DataFS, "/usr/share/pki/trust/anchors"):
			s.trustFilenamePattern = "/usr/share/pki/trust/anchors/%s.pem"
			s.caBundleFileName = "/etc/ssl/ca-bundle.pem"
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

func (s *Platform) checkCA(ca *CA) (bool, error) {
	if ok, err := s.check(); !ok {
		return ok, err
	}

	buf, err := s.DataFS.ReadFile(s.trustFilenamePath(ca))
	if err != nil {
		if errors.Is(err, syscall.ENOENT) {
			return false, nil
		}
		return false, err
	}

	var blk *pem.Block
	for len(buf) > 0 {
		if blk, buf = pem.Decode(buf); blk == nil {
			return false, nil
		}

		if bytes.Equal(ca.Raw, blk.Bytes) {
			return true, nil
		}
	}
	return false, nil
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

func (s *Platform) listCAs() ([]*CA, error) {
	if ok, err := s.check(); !ok {
		return nil, err
	}

	data, err := s.DataFS.ReadFile(s.caBundleFileName)
	if err != nil {
		if errors.Is(err, syscall.ENOENT) {
			return nil, nil
		}
		return nil, err
	}

	var cas []*CA
	for p, buf := pem.Decode(data); p != nil; p, buf = pem.Decode(buf) {
		cert, err := x509.ParseCertificate(p.Bytes)
		if err != nil {
			return nil, err
		}

		ca := &CA{
			Certificate: cert,
			UniqueName:  cert.SerialNumber.Text(16),
		}

		cas = append(cas, ca)
	}
	return cas, nil
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
