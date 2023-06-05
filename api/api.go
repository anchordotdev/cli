package api

//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen --package=api -generate=types -o ./openapi.gen.go ../../config/openapi.yml

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/keyring"
	"golang.org/x/exp/slices"
)

var (
	ErrSignedOut = errors.New("sign in required")
)

// TODO: maybe make this return a real client, not an http.Client
func Client(cfg *cli.Config) (*http.Client, error) {
	client := &http.Client{
		Transport: urlRewriter{
			RoundTripper: responseChecker{
				RoundTripper: new(http.Transport),
			},
			URL: cfg.API.URL,
		},
	}

	apiToken := cfg.API.Token
	if apiToken == "" {
		var (
			kr = keyring.Keyring{Config: cfg}

			err error
		)

		if apiToken, err = kr.Get(keyring.APIToken); err == keyring.ErrNotFound {
			return client, ErrSignedOut
		} else if err != nil {
			return nil, fmt.Errorf("reading PAT token from keyring failed: %w", err)
		}
		if !strings.HasPrefix(apiToken, "ap0_") || len(apiToken) != 64 {
			return nil, fmt.Errorf("read invalid PAT token from keyring")
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

type responseChecker struct {
	http.RoundTripper
}

var jsonMediaTypes = mediaTypes{
	"application/json",
	"application/problem+json",
}

func (r responseChecker) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := r.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("request error %s %s: %w", req.Method, req.URL.Path, err)
	}

	switch res.StatusCode {
	case http.StatusForbidden:
		return nil, ErrSignedOut
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if contentType := res.Header.Get("Content-Type"); !jsonMediaTypes.Matches(contentType) {
		return nil, fmt.Errorf("non-json response received: %q: %w", contentType, err)
	}
	return res, nil
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
	req.URL = u.JoinPath(req.URL.Path)

	return r.RoundTripper.RoundTrip(req)
}

type mediaTypes []string

func (s mediaTypes) Matches(val string) bool {
	media, _, err := mime.ParseMediaType(val)
	if err != nil {
		return false
	}
	return slices.Contains(s, media)
}
