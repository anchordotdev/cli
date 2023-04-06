package keyring

import (
	"os/user"

	"github.com/zalando/go-keyring"

	"github.com/anchordotdev/cli"
)

var ErrNotFound = keyring.ErrNotFound

type label string

const (
	APIToken label = "API Token"
)

func Delete(cfg *cli.Config, id label) error {
	u, err := user.Current()
	if err != nil {
		return err
	}

	return keyring.Delete(service(cfg, id), u.Username)
}

func Get(cfg *cli.Config, id label) (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}

	return keyring.Get(service(cfg, id), u.Username)
}

func Set(cfg *cli.Config, id label, secret string) error {
	u, err := user.Current()
	if err != nil {
		return err
	}

	return keyring.Set(service(cfg, id), u.Username, secret)
}

func service(cfg *cli.Config, id label) string {
	return cfg.API.URL + " " + string(id)
}
