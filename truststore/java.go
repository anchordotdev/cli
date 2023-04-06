package truststore

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"hash"
	"io/fs"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type Java struct {
	HomeDir string

	JavaHomeDir string
	StorePass   string

	DataFS fs.StatFS
	SysFS  CmdFS

	inito       sync.Once
	keytoolPath string
	cacertsPath string
}

func (s *Java) init() {
	s.inito.Do(func() {
		keytoolCommand := "keytool"
		if runtime.GOOS == "windows" {
			keytoolCommand = "keytool.exe"
		}

		if keytoolPath := filepath.Join(s.JavaHomeDir, "bin", keytoolCommand); binaryExists(s.SysFS, keytoolPath) {
			s.keytoolPath = keytoolPath
		}

		if cacertsPath := filepath.Join(s.JavaHomeDir, "lib", "security", "cacerts"); pathExists(s.SysFS, cacertsPath) {
			s.cacertsPath = cacertsPath
		}

		if cacertsPath := filepath.Join(s.JavaHomeDir, "jre", "lib", "security", "cacerts"); pathExists(s.SysFS, cacertsPath) {
			s.cacertsPath = cacertsPath
		}
	})
}

func (s *Java) CheckCA(ca *CA) (bool, error) {
	s.init()

	if s.keytoolPath == "" {
		return false, Error{
			Op:      OpCheck,
			Warning: ErrNoKeytool,
		}
	}

	// exists returns true if the given x509.Certificate's fingerprint
	// is in the keytool -list output
	exists := func(c *x509.Certificate, h hash.Hash, keytoolOutput []byte) bool {
		h.Write(c.Raw)
		fp := strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
		return bytes.Contains(keytoolOutput, []byte(fp))
	}

	args := []string{
		"-list",
		"-keystore", s.cacertsPath,
		"-storepass", s.StorePass,
	}

	keytoolOutput, err := s.SysFS.Exec(s.SysFS.Command(s.keytoolPath, args...))
	if err != nil {
		return false, fatalCmdErr(err, "keytool -list", keytoolOutput)
	}

	// keytool outputs SHA1 and SHA256 (Java 9+) certificates in uppercase hex
	// with each octet pair delimitated by ":". Drop them from the keytool output
	keytoolOutput = bytes.Replace(keytoolOutput, []byte(":"), nil, -1)

	// pre-Java 9 uses SHA1 fingerprints
	s1, s256 := sha1.New(), sha256.New()
	return exists(ca.Certificate, s1, keytoolOutput) || exists(ca.Certificate, s256, keytoolOutput), nil
}

func (s *Java) InstallCA(ca *CA) (bool, error) {
	s.init()

	if s.keytoolPath == "" {
		return false, Error{
			Op:      OpInstall,
			Warning: ErrNoKeytool,
		}
	}

	args := []string{
		"-importcert", "-noprompt",
		"-keystore", s.cacertsPath,
		"-storepass", s.StorePass,
		"-file", ca.FilePath,
		"-alias", ca.UniqueName,
	}

	if out, err := s.execKeytool(s.SysFS.Command(s.keytoolPath, args...)); err != nil {
		return false, fatalCmdErr(err, "keytool -importcert", out)
	}
	return true, nil
}

func (s *Java) UninstallCA(ca *CA) (bool, error) {
	s.init()

	if s.keytoolPath == "" {
		return false, Error{
			Op:      OpUninstall,
			Warning: ErrNoKeytool,
		}
	}

	args := []string{
		"-delete",
		"-alias", ca.UniqueName,
		"-keystore", s.cacertsPath,
		"-storepass", s.StorePass,
	}
	out, err := s.execKeytool(s.SysFS.Command(s.keytoolPath, args...))
	if bytes.Contains(out, []byte("does not exist")) {
		return false, nil // cert didn't exist
	}
	if err != nil {
		return false, fatalCmdErr(err, "keytool -delete", out)
	}
	return true, nil
}

// execKeytool will execute a "keytool" command and if needed re-execute
// the command with commandWithSudo to work around file permissions.
func (s *Java) execKeytool(cmd *exec.Cmd) ([]byte, error) {
	out, err := s.SysFS.Exec(cmd)
	if err != nil && bytes.Contains(out, []byte("java.io.FileNotFoundException")) && runtime.GOOS != "windows" {
		cmd = s.SysFS.Command(cmd.Args[0], cmd.Args[1:]...)
		cmd.Env = []string{
			"JAVA_HOME=" + s.JavaHomeDir,
		}
		return s.SysFS.SudoExec(cmd)
	}
	return out, err
}
