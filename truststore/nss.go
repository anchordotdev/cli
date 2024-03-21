package truststore

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type NSS struct {
	HomeDir string

	DataFS fs.StatFS
	SysFS  CmdFS

	inito               sync.Once
	dbPaths             []string
	certutilPath        string
	certutilInstallHelp string
	firefoxProfiles     []string
}

const (
	certUtilBadDatabaseOutput    = "SEC_ERROR_BAD_DATABASE"
	certUtilPRFileNotFoundOutput = "PR_FILE_NOT_FOUND_ERROR"
	certUtilSecReadOnlyOutput    = "SEC_ERROR_READ_ONLY"
)

var firefoxPaths = []string{
	"/usr/bin/firefox",
	"/usr/bin/firefox-nightly",
	"/usr/bin/firefox-developer-edition",
	"/snap/firefox",
	"/Applications/Firefox.app",
	"/Applications/FirefoxDeveloperEdition.app",
	"/Applications/Firefox Developer Edition.app",
	"/Applications/Firefox Nightly.app",
	"C:\\Program Files\\Mozilla Firefox",
}

func (s *NSS) init() {
	s.inito.Do(func() {
		dbPaths := []string{
			filepath.Join(s.HomeDir, ".pki/nssdb"),
			filepath.Join(s.HomeDir, "snap/chromium/current/.pki/nssdb"), // Snapcraft
			"/etc/pki/nssdb", // CentOS 7
		}

		allPaths := append(append([]string{}, dbPaths...), firefoxPaths...)
		for _, path := range allPaths {
			if pathExists(s.DataFS, path) {
				s.dbPaths = append(s.dbPaths, path)
			}
		}

		switch runtime.GOOS {
		case "darwin":
			switch {
			case binaryExists(s.SysFS, "certutil"):
				s.certutilPath, _ = s.SysFS.LookPath("certutil")
			case binaryExists(s.SysFS, "/usr/local/opt/nss/bin/certutil"):
				// Check the default Homebrew path, to save executing Ruby. #135
				s.certutilPath = "/usr/local/opt/nss/bin/certutil"
			default:
				if out, err := s.SysFS.Exec(s.SysFS.Command("brew", "--prefix", "nss")); err != nil {
					certutilPath := filepath.Join(strings.TrimSpace(string(out)), "bin", "certutil")
					if pathExists(s.DataFS, certutilPath) {
						s.certutilPath = certutilPath
					}
				}
			}

		case "linux":
			if binaryExists(s.SysFS, "certutil") {
				s.certutilPath, _ = s.SysFS.LookPath("certutil")
			}
		}

		s.certutilInstallHelp = certutilInstallHelp(s.SysFS)
		s.firefoxProfiles = firefoxProfiles(s.HomeDir)
	})
}

func (s *NSS) Browsers() string { return nssBrowsers }

func (s *NSS) Check() (bool, error) {
	s.init()

	if s.dbPaths == nil {
		return false, Error{
			Op: OpCheck,

			Warning: NSSError{
				Err: ErrNoNSSDB,

				CertutilInstallHelp: s.certutilInstallHelp,
				NSSBrowsers:         nssBrowsers,
			},
		}
	}
	if s.certutilPath == "" {
		return true, Error{
			Op: OpCheck,

			Warning: NSSError{
				Err: ErrNoCertutil,

				CertutilInstallHelp: s.certutilInstallHelp,
				NSSBrowsers:         nssBrowsers,
			},
		}
	}
	count, _ := s.forEachNSSProfile(func(profile string) error {
		return nil
	})
	if count == 0 {
		return false, Error{
			Op: OpInstall,

			Warning: NSSError{
				Err: ErrNoNSSDB,

				CertutilInstallHelp: s.certutilInstallHelp,
				NSSBrowsers:         nssBrowsers,
			},
		}
	}
	return true, nil
}

func (s *NSS) CheckCA(ca *CA) (installed bool, err error) {
	s.init()

	if s.certutilPath == "" {
		return false, nil // TODO: return a NSSError here
	}

	count, err := s.forEachNSSProfile(func(profile string) error {
		out, err := s.SysFS.Exec(s.SysFS.Command(s.certutilPath, "-V", "-d", profile, "-u", "L", "-n", ca.UniqueName))
		if err != nil {
			return NSSError{
				Err: errors.New(string(out)),

				CertutilInstallHelp: s.certutilInstallHelp,
				NSSBrowsers:         nssBrowsers,
			}
		}
		return nil
	})

	if err != nil {
		var exerr *exec.ExitError
		if errors.As(err, &exerr) && exerr.ProcessState.ExitCode() == 255 {
			return false, nil
		}

		return false, Error{
			Op: OpCheck,

			Warning: NSSError{
				Err: err,

				CertutilInstallHelp: s.certutilInstallHelp,
				NSSBrowsers:         nssBrowsers,
			},
		}
	}
	return count != 0, err
}

func (s *NSS) Description() string { return "Network Security Services (NSS)" }

func (s *NSS) InstallCA(ca *CA) (installed bool, err error) {
	s.init()

	if s.certutilPath == "" {
		if s.certutilInstallHelp == "" {
			return false, Error{
				Op: OpInstall,

				Warning: NSSError{
					Err: ErrNoNSS,

					CertutilInstallHelp: s.certutilInstallHelp,
					NSSBrowsers:         nssBrowsers,
				},
			}
		}

		return false, Error{
			Op: OpInstall,

			Warning: NSSError{
				Err: ErrNoCertutil,

				CertutilInstallHelp: s.certutilInstallHelp,
				NSSBrowsers:         nssBrowsers,
			},
		}
	}

	count, err := s.forEachNSSProfile(func(profile string) error {
		args := []string{
			"-A", "-d", profile,
			"-t", "C,,",
			"-n", ca.UniqueName,
			"-i", ca.FilePath,
		}

		if out, err := s.execCertutil(s.certutilPath, args...); err != nil {
			return fatalCmdErr(err, "certutil -A -d "+profile, out)
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	if count == 0 {
		return false, Error{
			Op: OpInstall,

			Warning: NSSError{
				Err: ErrNoNSSDB,

				CertutilInstallHelp: s.certutilInstallHelp,
				NSSBrowsers:         nssBrowsers,
			},
		}
	}
	if ok, _ := s.CheckCA(ca); !ok {
		return false, Error{Warning: ErrUnknownNSS}
	}
	return true, nil
}

func (s *NSS) ListCAs() ([]*CA, error) {
	s.init()

	if s.certutilPath == "" {
		return nil, NSSError{
			Err: ErrNoNSS,

			CertutilInstallHelp: s.certutilInstallHelp,
			NSSBrowsers:         nssBrowsers,
		}
	}

	var cas []*CA
	_, err := s.forEachNSSProfile(func(profile string) error {
		out, err := s.SysFS.Exec(s.SysFS.Command(s.certutilPath, "-L", "-d", profile))
		if err != nil {
			message := string(out)

			if bytes.Contains(out, []byte(certUtilBadDatabaseOutput)) {
				message = "certutil bad database `" + profile + "`"
			}

			return NSSError{
				Err: errors.New(message),

				CertutilInstallHelp: s.certutilInstallHelp,
				NSSBrowsers:         nssBrowsers,
			}
		}

		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) < 2 {
			return NSSError{
				Err: errors.New("unexpected certutil output"),

				CertutilInstallHelp: s.certutilInstallHelp,
				NSSBrowsers:         nssBrowsers,
			}
		}

		padLen := strings.Index(lines[0], "Trust Attributes")
		if padLen <= 0 {
			return NSSError{
				Err: errors.New("unexpected certutil output format"),

				CertutilInstallHelp: s.certutilInstallHelp,
				NSSBrowsers:         nssBrowsers,
			}
		}

		if len(lines) == 2 {
			return nil // no certs in the output
		}

		var nicks []string
		for _, line := range lines[3:] {
			if len(line) < padLen {
				return NSSError{
					Err: errors.New("unexpected certutil line format"),

					CertutilInstallHelp: s.certutilInstallHelp,
					NSSBrowsers:         nssBrowsers,
				}
			}

			nicks = append(nicks, strings.TrimSpace(line[:padLen]))
		}

		for _, nick := range nicks {
			out, err := s.SysFS.Exec(s.SysFS.Command(s.certutilPath, "-L", "-d", profile, "-n", nick, "-a"))
			if err != nil {
				return NSSError{
					Err: errors.New(string(out)),

					CertutilInstallHelp: s.certutilInstallHelp,
					NSSBrowsers:         nssBrowsers,
				}
			}

			for p, buf := pem.Decode(out); p != nil; p, buf = pem.Decode(buf) {
				cert, err := x509.ParseCertificate(p.Bytes)
				if err != nil {
					return err
				}

				ca := &CA{
					Certificate: cert,
					NickName:    nick,
					UniqueName:  cert.SerialNumber.Text(16),
				}

				cas = append(cas, ca)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return cas, nil
}

func (s *NSS) UninstallCA(ca *CA) (bool, error) {
	s.init()

	if s.certutilPath == "" {
		if s.certutilInstallHelp == "" {
			return false, Error{
				Op: OpUninstall,

				Warning: NSSError{
					Err: ErrNoNSS,

					CertutilInstallHelp: s.certutilInstallHelp,
					NSSBrowsers:         nssBrowsers,
				},
			}
		}

		return false, Error{
			Op: OpUninstall,

			Warning: NSSError{
				Err: ErrNoCertutil,

				CertutilInstallHelp: s.certutilInstallHelp,
				NSSBrowsers:         nssBrowsers,
			},
		}
	}

	// if we don't have a nickname value available, fallback to UniqueName
	nickName := ca.NickName
	if nickName == "" {
		nickName = ca.UniqueName
	}

	_, err := s.forEachNSSProfile(func(profile string) error {
		args := []string{
			"-V", "-d", profile,
			"-u", "L",
			"-n", nickName,
		}

		if out, err := s.SysFS.Exec(s.SysFS.Command(s.certutilPath, args...)); err != nil {
			// continue if this profile does not have a ca with this nickname
			if bytes.Contains(out, []byte(certUtilPRFileNotFoundOutput)) {
				return nil
			}
		}

		// TODO: ensure the certificate has our extension before removal
		args = []string{
			"-D", "-d", profile,
			"-n", nickName,
		}

		if out, err := s.execCertutil(s.certutilPath, args...); err != nil {
			return fatalCmdErr(err, "certutil -D -d "+profile, out)
		}
		return nil
	})

	return err == nil, err
}

// execCertutil will execute a "certutil" command and if needed re-execute
// the command with commandWithSudo to work around file permissions.
func (s *NSS) execCertutil(path string, arg ...string) ([]byte, error) {
	out, err := s.SysFS.Exec(s.SysFS.Command(path, arg...))
	if err != nil && bytes.Contains(out, []byte(certUtilSecReadOnlyOutput)) && runtime.GOOS != "windows" {
		out, err = s.SysFS.SudoExec(s.SysFS.Command(path, arg...))
	}
	return out, err
}

func (s *NSS) forEachNSSProfile(f func(profile string) error) (found int, err error) {
	var profiles []string
	profiles = append(profiles, s.dbPaths...)
	for _, ff := range s.firefoxProfiles {
		pp, _ := filepath.Glob(ff)
		profiles = append(profiles, pp...)
	}

	for _, profile := range profiles {
		if stat, err := os.Stat(profile); err != nil || !stat.IsDir() {
			continue
		}
		if pathExists(s.DataFS, filepath.Join(profile, "cert9.db")) {
			if err := f("sql:" + profile); err != nil {
				return 0, err
			}
			found++
		} else if pathExists(s.DataFS, filepath.Join(profile, "cert8.db")) {
			if err := f("dbm:" + profile); err != nil {
				return 0, err
			}
			found++
		}
	}
	return
}
