package truststore

import (
	"bytes"
	"io/fs"
	"os"
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
	return true, nil
}

func (s *NSS) CheckCA(ca *CA) (installed bool, err error) {
	s.init()

	if s.certutilPath == "" {
		return false, nil // TODO: return a NSSError here
	}

	count, err := s.forEachNSSProfile(func(profile string) error {
		_, err := s.SysFS.Exec(s.SysFS.Command(s.certutilPath, "-V", "-d", profile, "-u", "L", "-n", ca.UniqueName))
		return err
	})

	if err != nil {
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

	_, err := s.forEachNSSProfile(func(profile string) error {
		args := []string{
			"-V", "-d", profile,
			"-u", "L",
			"-n", ca.UniqueName,
		}

		if _, err := s.SysFS.Exec(s.SysFS.Command(s.certutilPath, args...)); err != nil {
			return nil
		}

		args = []string{
			"-D", "-d", profile,
			"-n", ca.UniqueName,
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
	if err != nil && bytes.Contains(out, []byte("SEC_ERROR_READ_ONLY")) && runtime.GOOS != "windows" {
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
