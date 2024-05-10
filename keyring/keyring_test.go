package keyring

import (
	"testing"

	"github.com/anchordotdev/cli"
	_ "github.com/anchordotdev/cli/testflags"
	"github.com/zalando/go-keyring"
)

func TestKeyring(t *testing.T) {
	cfg := new(cli.Config)
	cfg.Keyring.MockMode = true
	cfg.API.URL = "http://test-keyring-url.example.com/"

	kr := &Keyring{Config: cfg}

	val, err := kr.Get(label("test-missing-label"))
	if want, got := keyring.ErrNotFound, err; want != got {
		t.Fatalf("want read from empty keyring error %q, got %q", want, got)
	}
	if want, got := "", val; want != got {
		t.Errorf("want read from empty keyring value %q, got %q", want, got)
	}

	secret := "open sesame"
	if err := kr.Set(label("test-secret-label"), secret); err != nil {
		t.Fatal(err)
	}

	if val, err = kr.Get(label("test-secret-label")); err != nil {
		t.Fatal(err)
	}
	if want, got := secret, val; want != got {
		t.Errorf("want read after keyring write value %q, got %q", want, got)
	}

	if want, got := keyring.ErrNotFound, kr.Delete(label("test-missing-label")); want != got {
		t.Fatalf("want delete for unset keyring error %q, got %q", want, got)
	}

	if err := kr.Delete(label("test-secret-label")); err != nil {
		t.Fatal(err)
	}
}
