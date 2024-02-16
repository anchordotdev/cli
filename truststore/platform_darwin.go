package truststore

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"howett.net/plist"
)

// https://github.com/golang/go/issues/24652#issuecomment-399826583
var (
	nssBrowsers = "Firefox"

	trustSettings     []interface{}
	_, _              = plist.Unmarshal(trustSettingsData, &trustSettings)
	trustSettingsData = []byte(`
<array>
	<dict>
		<key>kSecTrustSettingsPolicy</key>
		<data>
		KoZIhvdjZAED
		</data>
		<key>kSecTrustSettingsPolicyName</key>
		<string>sslServer</string>
		<key>kSecTrustSettingsResult</key>
		<integer>1</integer>
	</dict>
	<dict>
		<key>kSecTrustSettingsPolicy</key>
		<data>
		KoZIhvdjZAEC
		</data>
		<key>kSecTrustSettingsPolicyName</key>
		<string>basicX509</string>
		<key>kSecTrustSettingsResult</key>
		<integer>1</integer>
	</dict>
</array>
`)
)

func certutilInstallHelp(_ CmdFS) string { return "brew install nss" }

func firefoxProfiles(homeDir string) []string {
	return []string{
		filepath.Join(homeDir, "/Library/Application Support/Firefox/Profiles/*"),
	}
}

type Platform struct {
	HomeDir string

	DataFS fs.StatFS
	SysFS  CmdFS

	inito               sync.Once
	nssBrowsers         string
	certutilInstallHelp string
	firefoxProfiles     []string
}

func (s *Platform) Description() string { return "System (MacOS Keychain)" }

func (s *Platform) check() (bool, error) {
	s.inito.Do(func() {
		s.certutilInstallHelp = certutilInstallHelp(s.SysFS)
		s.firefoxProfiles = firefoxProfiles(s.HomeDir)
	})

	return true, nil
}

func (s *Platform) checkCA(ca *CA) (bool, error) {
	args := []string{
		"find-certificate", "-a", "-c", ca.Subject.CommonName,
		"-p", "/Library/Keychains/System.keychain",
	}
	out, err := s.SysFS.Exec(s.SysFS.Command("security", args...))
	if err != nil {
		return false, fatalCmdErr(err, "security find-certificate", out)
	}

	var blk *pem.Block
	for buf := []byte(out); len(buf) > 0; {
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
	args := []string{
		"add-trusted-cert", "-d",
		"-k", "/Library/Keychains/System.keychain",
		ca.FilePath,
	}
	if out, err := s.SysFS.SudoExec(s.SysFS.Command("security", args...)); err != nil {
		return false, fatalCmdErr(err, "security add-trusted-cert", out)
	}

	// Make trustSettings explicit, as older Go does not know the defaults.
	// https://github.com/golang/go/issues/24652

	plistFile, err := os.CreateTemp("", "trust-settings")
	if err != nil {
		return false, fatalErr(err, "failed to create temp file")
	}
	defer os.Remove(plistFile.Name())

	args = []string{
		"trust-settings-export",
		"-d", plistFile.Name(),
	}
	if out, err := s.SysFS.SudoExec(s.SysFS.Command("security", args...)); err != nil {
		return false, fatalCmdErr(err, "security trust-settings-export", out)
	}

	plistData, err := os.ReadFile(plistFile.Name())
	if err != nil {
		return false, fatalErr(err, "failed to read trust settings")
	}

	var plistRoot map[string]interface{}
	if _, err = plist.Unmarshal(plistData, &plistRoot); err != nil {
		return false, fatalErr(err, "failed to parse trust settings")
	}
	if plistRoot["trustVersion"].(uint64) != 1 {
		return false, fmt.Errorf("unsupported trust settings version: %s", plistRoot["trustVersion"])
	}

	rootSubjectASN1, err := asn1.Marshal(ca.Certificate.Subject.ToRDNSequence())
	if err != nil {
		return false, fatalErr(err, "failed to marshal certificate subject")
	}

	trustList := plistRoot["trustList"].(map[string]interface{})
	for key := range trustList {
		entry := trustList[key].(map[string]interface{})
		if _, ok := entry["issuerName"]; !ok {
			continue
		}
		issuerName := entry["issuerName"].([]byte)
		if !bytes.Equal(rootSubjectASN1, issuerName) {
			continue
		}
		entry["trustSettings"] = trustSettings
		break
	}

	if plistData, err = plist.MarshalIndent(plistRoot, plist.XMLFormat, "\t"); err != nil {
		return false, fatalErr(err, "failed to serialize trust settings")
	}
	if err = os.WriteFile(plistFile.Name(), plistData, 0600); err != nil {
		return false, fatalErr(err, "failed to write trust settings")
	}

	args = []string{
		"trust-settings-import",
		"-d", plistFile.Name(),
	}
	if out, err := s.SysFS.SudoExec(s.SysFS.Command("security", args...)); err != nil {
		return false, fatalCmdErr(err, "security trust-settings-import", out)
	}

	return true, nil
}

func (s *Platform) listCAs() ([]*CA, error) {
	args := []string{
		"find-certificate", "-a", "-p",
	}

	out, err := s.SysFS.Exec(s.SysFS.Command("security", args...))
	if err != nil {
		return nil, fatalCmdErr(err, "security add-trusted-cert", out)
	}

	var cas []*CA
	for p, buf := pem.Decode(out); p != nil; p, buf = pem.Decode(buf) {
		cert, err := x509.ParseCertificate(p.Bytes)
		if err != nil {
			if isDupExtErr(err) {
				continue
			}
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

const untrustedCertOutput = "SecTrustSettingsRemoveTrustSettings: The specified item could not be found in the keychain.\n"

func (s *Platform) uninstallCA(ca *CA) (bool, error) {
	args := []string{
		"remove-trusted-cert",
		"-d", ca.FilePath,
	}
	if out, err := s.SysFS.SudoExec(s.SysFS.Command("security", args...)); err != nil {
		if !bytes.Equal(out, []byte(untrustedCertOutput)) {
			return false, fatalCmdErr(err, "security remove-trusted-cert", out)
		}
	}

	sum256 := sha256.Sum256(ca.Raw)

	args = []string{
		"delete-certificate",
		"-Z", hex.EncodeToString(sum256[:]),
	}

	if out, err := s.SysFS.SudoExec(s.SysFS.Command("security", args...)); err != nil {
		if !bytes.Equal(out, []byte(untrustedCertOutput)) {
			return false, fatalCmdErr(err, "security delete-certificate", out)
		}
	}
	return true, nil
}

func isDupExtErr(err error) bool {
	return err.Error() == "x509: certificate contains duplicate extensions"
}
