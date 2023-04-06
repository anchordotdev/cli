package api

//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen --package=api -generate=types -o ./openapi.gen.go ../../config/openapi.yml

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/keyring"
)

var (
	ErrSignedOut = errors.New("sign in required")
)

// TODO: maybe make this return a real client, not an http.Client
func Client(cfg *cli.Config) (*http.Client, error) {
	client := &http.Client{
		Transport: urlRewriter{
			RoundTripper: new(http.Transport),
			URL:          cfg.API.URL,
		},
	}

	apiToken := cfg.API.Token
	if apiToken == "" {
		var err error
		if apiToken, err = keyring.Get(cfg, keyring.APIToken); err == keyring.ErrNotFound {
			return client, ErrSignedOut
		} else if err != nil {
			return nil, err
		}
	}

	client.Transport = basicAuther{
		RoundTripper: client.Transport,
		PAT:          apiToken,
	}

	return client, nil
}

type basicAuther struct {
	http.RoundTripper

	PAT string
}

func (r basicAuther) RoundTrip(req *http.Request) (*http.Response, error) {
	if r.PAT != "" {
		req.SetBasicAuth(r.PAT, "")
	}

	return r.RoundTripper.RoundTrip(req)
}

type urlRewriter struct {
	http.RoundTripper

	URL string
}

func (r urlRewriter) RoundTrip(req *http.Request) (*http.Response, error) {
	u, err := url.Parse(r.URL)
	if err != nil {
		return nil, err
	}
	if u, err = u.Parse(req.URL.Path); err != nil {
		return nil, err
	}
	req.URL = u

	return r.RoundTripper.RoundTrip(req)
}
