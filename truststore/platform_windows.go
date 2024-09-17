package truststore

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	modcrypt32                           = syscall.NewLazyDLL("crypt32.dll")
	procCertAddEncodedCertificateToStore = modcrypt32.NewProc("CertAddEncodedCertificateToStore")
	procCertCloseStore                   = modcrypt32.NewProc("CertCloseStore")
	procCertDeleteCertificateFromStore   = modcrypt32.NewProc("CertDeleteCertificateFromStore")
	procCertDuplicateCertificateContext  = modcrypt32.NewProc("CertDuplicateCertificateContext")
	procCertEnumCertificatesInStore      = modcrypt32.NewProc("CertEnumCertificatesInStore")
	procCertOpenSystemStoreW             = modcrypt32.NewProc("CertOpenSystemStoreW")

	nssBrowsers = "Firefox"
)

func certutilInstallHelp(_ CmdFS) string {
	return "" // certutil unsupported on Windows
}

func firefoxProfiles(homeDir string) []string {
	return []string{
		filepath.Join(homeDir, "\\AppData\\Roaming\\Mozilla\\Firefox\\Profiles"),
	}
}

type Platform struct {
	HomeDir string

	DataFS fs.StatFS
	SysFS  CmdFS
}

func (s *Platform) Description() string { return "System (Windows)" }

func (s *Platform) check() (bool, error) {
	store, err := openWindowsRootStore()
	if err != nil {
		return false, nil
	}
	store.close()

	return true, nil
}

func (s *Platform) checkCA(ca *CA) (bool, error) {
	store, err := openWindowsRootStore()
	if err != nil {
		return false, fatalErr(err, "open root store")
	}
	defer store.close()

	cert, err := store.findCertWithSerial(ca.Certificate.SerialNumber)
	if err != nil {
		return false, fatalErr(err, "find ca")
	}
	return (cert != nil), nil
}

func (s *Platform) installCA(ca *CA) (bool, error) {
	// Load cert
	cert, err := os.ReadFile(ca.FilePath)
	if err != nil {
		return false, fatalErr(err, "failed to read root certificate")
	}
	// Decode PEM
	certBlock, _ := pem.Decode(cert)
	if certBlock == nil || certBlock.Type != "CERTIFICATE" {
		return false, fatalErr(fmt.Errorf("invalid PEM data"), "decode pem")
	}
	cert = certBlock.Bytes
	// Open root store
	store, err := openWindowsRootStore()
	if err != nil {
		return false, fatalErr(err, "open root store")
	}
	defer store.close()
	// Add cert
	if err := store.addCert(cert); err != nil {
		return false, fatalErr(store.addCert(cert), "add cert")
	}
	return true, nil
}

func (s *Platform) listCAs() ([]*CA, error) {
	store, err := openWindowsRootStore()
	if err != nil {
		return nil, fatalErr(err, "open root store")
	}
	defer store.close()

	certs, err := store.listCerts()
	if err != nil {
		return nil, fatalErr(err, "list cas")
	}

	var cas []*CA
	for _, cert := range certs {
		ca := &CA{
			Certificate: cert,
			UniqueName:  cert.SerialNumber.Text(16),
		}

		cas = append(cas, ca)
	}
	return cas, nil
}

func (s *Platform) uninstallCA(ca *CA) (bool, error) {
	// We'll just remove all certs with the same serial number
	// Open root store
	store, err := openWindowsRootStore()
	if err != nil {
		return false, fatalErr(err, "open root store")
	}
	defer store.close()
	// Do the deletion
	deletedAny, err := store.deleteCertsWithSerial(ca.Certificate.SerialNumber)
	if err == nil && !deletedAny {
		err = fmt.Errorf("no certs found")
	}
	if err != nil {
		return false, fatalErr(err, "delete cert")
	}
	return true, nil
}

type windowsRootStore uintptr

func openWindowsRootStore() (windowsRootStore, error) {
	rootStr, err := syscall.UTF16PtrFromString("ROOT")
	if err != nil {
		return 0, err
	}
	store, _, err := procCertOpenSystemStoreW.Call(0, uintptr(unsafe.Pointer(rootStr)))
	if store != 0 {
		return windowsRootStore(store), nil
	}
	return 0, fmt.Errorf("failed to open windows root store: %v", err)
}

func (w windowsRootStore) close() error {
	ret, _, err := procCertCloseStore.Call(uintptr(w), 0)
	if ret != 0 {
		return nil
	}
	return fmt.Errorf("failed to close windows root store: %v", err)
}

func (w windowsRootStore) addCert(cert []byte) error {
	// TODO: ok to always overwrite?
	ret, _, err := procCertAddEncodedCertificateToStore.Call(
		uintptr(w), // HCERTSTORE hCertStore
		uintptr(syscall.X509_ASN_ENCODING|syscall.PKCS_7_ASN_ENCODING), // DWORD dwCertEncodingType
		uintptr(unsafe.Pointer(&cert[0])),                              // const BYTE *pbCertEncoded
		uintptr(len(cert)),                                             // DWORD cbCertEncoded
		3,                                                              // DWORD dwAddDisposition (CERT_STORE_ADD_REPLACE_EXISTING is 3)
		0,                                                              // PCCERT_CONTEXT *ppCertContext
	)
	if ret != 0 {
		return nil
	}
	return fmt.Errorf("failed adding cert: %v", err)
}

func (w windowsRootStore) deleteCertsWithSerial(serial *big.Int) (bool, error) {
	// Go over each, deleting the ones we find
	var cert *syscall.CertContext
	deletedAny := false
	for {
		// Next enum
		certPtr, _, err := procCertEnumCertificatesInStore.Call(uintptr(w), uintptr(unsafe.Pointer(cert)))
		if cert = (*syscall.CertContext)(unsafe.Pointer(certPtr)); cert == nil {
			if errno, ok := err.(syscall.Errno); ok && errno == 0x80092004 {
				break
			}
			return deletedAny, fmt.Errorf("failed enumerating certs: %v", err)
		}
		// Parse cert
		certBytes := (*[1 << 20]byte)(unsafe.Pointer(cert.EncodedCert))[:cert.Length]
		parsedCert, err := x509.ParseCertificate(certBytes)
		// We'll just ignore parse failures for now
		if err == nil && parsedCert.SerialNumber != nil && parsedCert.SerialNumber.Cmp(serial) == 0 {
			// Duplicate the context so it doesn't stop the enum when we delete it
			dupCertPtr, _, err := procCertDuplicateCertificateContext.Call(uintptr(unsafe.Pointer(cert)))
			if dupCertPtr == 0 {
				return deletedAny, fmt.Errorf("failed duplicating context: %v", err)
			}
			if ret, _, err := procCertDeleteCertificateFromStore.Call(dupCertPtr); ret == 0 {
				return deletedAny, fmt.Errorf("failed deleting certificate: %v", err)
			}
			deletedAny = true
		}
	}
	return deletedAny, nil
}

func (w windowsRootStore) findCertWithSerial(serial *big.Int) (*x509.Certificate, error) {
	var cert *syscall.CertContext
	for {
		// Next enum
		certPtr, _, err := procCertEnumCertificatesInStore.Call(uintptr(w), uintptr(unsafe.Pointer(cert)))
		if cert = (*syscall.CertContext)(unsafe.Pointer(certPtr)); cert == nil {
			if errno, ok := err.(syscall.Errno); ok && errno == 0x80092004 {
				break
			}
			return nil, fmt.Errorf("failed enumerating certs: %v", err)
		}

		// Parse cert
		certBytes := (*[1 << 20]byte)(unsafe.Pointer(cert.EncodedCert))[:cert.Length]
		parsedCert, err := x509.ParseCertificate(certBytes)
		if err == nil && parsedCert.SerialNumber != nil && parsedCert.SerialNumber.Cmp(serial) == 0 {
			return parsedCert, nil
		}
	}
	return nil, nil
}

func (w windowsRootStore) listCerts() ([]*x509.Certificate, error) {
	var certs []*x509.Certificate

	var cert *syscall.CertContext
	for {
		// Next enum
		certPtr, _, err := procCertEnumCertificatesInStore.Call(uintptr(w), uintptr(unsafe.Pointer(cert)))
		if cert = (*syscall.CertContext)(unsafe.Pointer(certPtr)); cert == nil {
			if errno, ok := err.(syscall.Errno); ok && errno == 0x80092004 {
				break
			}
			return nil, fmt.Errorf("failed enumerating certs: %v", err)
		}

		// Parse cert
		certBytes := (*[1 << 20]byte)(unsafe.Pointer(cert.EncodedCert))[:cert.Length]
		if cert, err := parseCertificate(certBytes); cert != nil && err == nil {
			// ignore individual cert parsing errors
			certs = append(certs, cert)
		}
	}
	return certs, nil
}
