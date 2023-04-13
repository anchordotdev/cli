package keyring

import (
	"os/user"
	"sync"

	"github.com/zalando/go-keyring"

	"github.com/anchordotdev/cli"
)

var ErrNotFound = keyring.ErrNotFound

type label string

const (
	APIToken label = "API Token"
)

type Keyring struct {
	Config *cli.Config

	inito sync.Once
}

func (k *Keyring) init() {
	k.inito.Do(func() {
		if k.Config.Keyring.MockMode {
			keyring.MockInit()
		}
	})
}

func (k *Keyring) Delete(id label) error {
	k.init()

	u, err := user.Current()
	if err != nil {
		return err
	}

	return keyring.Delete(k.service(id), u.Username)
}

func (k *Keyring) Get(id label) (string, error) {
	k.init()

	u, err := user.Current()
	if err != nil {
		return "", err
	}

	return keyring.Get(k.service(id), u.Username)
}

func (k *Keyring) Set(id label, secret string) error {
	k.init()

	u, err := user.Current()
	if err != nil {
		return err
	}

	return keyring.Set(k.service(id), u.Username, secret)
}

func (k *Keyring) service(id label) string {
	return k.Config.API.URL + " " + string(id)
}
