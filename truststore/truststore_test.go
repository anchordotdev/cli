package truststore

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"io/fs"
	"os"
	"path"
	"slices"
	"testing"
	"testing/fstest"
	"time"

	"github.com/anchordotdev/cli/internal/must"
)

var (
	_ = flag.Bool("prism-verbose", false, "ignored")
	_ = flag.Bool("prism-proxy", false, "ignored")
	_ = flag.Bool("update", false, "ignored")
)

func testStore(t *testing.T, store Store) {
	if ok, err := store.Check(); err != nil {
		t.Fatal(err)
	} else if !ok {
		t.Fatalf("%q: initial check failed", store.Description())
	}

	if initialCAs, err := store.ListCAs(); err != nil {
		t.Fatal(err)
	} else if slices.ContainsFunc(initialCAs, ca.Equal) {
		t.Fatalf("%q: initial ca list already contains %+v ca", store, ca)
	}

	ok, err := store.CheckCA(ca)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("%q: check ca with %+v unexpectedly passed", store.Description(), ca)
	}

	if ok, err = store.InstallCA(ca); err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("%q: install ca with %+v failed", store.Description(), ca)
	}

	if ok, err = store.CheckCA(ca); err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("%q: check ca with %+v failed", store.Description(), ca)
	}

	if allCAs, err := store.ListCAs(); err != nil {
		t.Fatal(err)
	} else if !slices.ContainsFunc(allCAs, ca.Equal) {
		t.Fatalf("%q: ca list does not contain %+v ca", store.Description(), ca)
	}

	if ok, err := store.UninstallCA(ca); err != nil {
		t.Fatal(err)
	} else if !ok {
		t.Fatalf("%q: uninstall ca with %+v failed", store.Description(), ca)
	}

	if allCAs, err := store.ListCAs(); err != nil {
		t.Fatal(err)
	} else if slices.ContainsFunc(allCAs, ca.Equal) {
		t.Fatalf("%q: ca list still contains %+v ca", store.Description(), ca)
	}
}

var ca = mustCA(must.CA(&x509.Certificate{
	Subject: pkix.Name{
		CommonName:   "Example CA",
		Organization: []string{"Example, Inc"},
	},
	KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,

	ExtraExtensions: []pkix.Extension{},
}))

func mustCA(cert *must.Certificate) *CA {
	uniqueName := cert.Leaf.SerialNumber.Text(16)

	return &CA{
		Certificate: cert.Leaf,
		FilePath:    "example-ca-" + uniqueName + ".pem",
		UniqueName:  uniqueName,
	}
}

type TestFS fstest.MapFS

func (fsys TestFS) Open(name string) (fs.File, error)     { return fstest.MapFS(fsys).Open(name) }
func (fsys TestFS) ReadFile(name string) ([]byte, error)  { return fstest.MapFS(fsys).ReadFile(name) }
func (fsys TestFS) Stat(name string) (fs.FileInfo, error) { return fstest.MapFS(fsys).Stat(name) }

func (fsys TestFS) AppendToFile(name string, p []byte) error {
	f, ok := fsys[name]
	if !ok {
		f = new(fstest.MapFile)
		fsys[name] = f
	}

	f.Data = append(f.Data, p...)
	f.Sys = mapFI{
		name: name,
		size: len(f.Data),
	}

	return nil
}

func (fsys TestFS) Remove(name string) error {
	delete(fsys, name)
	return nil
}

func (fsys TestFS) Rename(oldpath, newpath string) error {
	fsys[newpath] = fsys[oldpath]
	return fsys.Remove(oldpath)
}

// golang.org/x/tools/godoc/vfs/mapfs

type mapFI struct {
	name string
	size int
	dir  bool
}

func (fi mapFI) IsDir() bool        { return fi.dir }
func (fi mapFI) ModTime() time.Time { return time.Time{} }
func (fi mapFI) Mode() os.FileMode {
	if fi.IsDir() {
		return 0755 | os.ModeDir
	}
	return 0444
}
func (fi mapFI) Name() string     { return path.Base(fi.name) }
func (fi mapFI) Size() int64      { return int64(fi.size) }
func (fi mapFI) Sys() interface{} { return nil }
