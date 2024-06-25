package api

//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen --package=api -generate=types -o ./openapi.gen.go ../../config/openapi.yml

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/keyring"
	"github.com/anchordotdev/cli/version"
	"golang.org/x/exp/slices"
)

var (
	ErrSignedOut            = errors.New("sign in required")
	ErrTransient            = errors.New("transient error encountered, please retry")
	ErrGnomeKeyringRequired = fmt.Errorf("gnome-keyring required for secure credential storage: %w", ErrSignedOut)
)

// NB: can't call this Client since the name is already taken by an openapi
// generated type. It's more like a session anyways, since it caches some
// current user info.

type Session struct {
	*http.Client

	cfg *cli.Config

	userInfo *Root
}

// TODO: rename to NewSession
func NewClient(ctx context.Context, cfg *cli.Config) (*Session, error) {
	anc := &Session{
		Client: &http.Client{
			Transport: urlRewriter{
				RoundTripper: responseChecker{
					RoundTripper: userAgentSetter{
						RoundTripper: preferSetter{
							cfg:          cfg,
							RoundTripper: new(http.Transport),
						},
					},
				},
				URL: cfg.API.URL,
			},
		},

		cfg: cfg,
	}

	apiToken := cfg.API.Token
	if apiToken == "" {
		var (
			kr = keyring.Keyring{Config: cfg}

			err error
		)

		if apiToken, err = kr.Get(keyring.APIToken); err == keyring.ErrNotFound {
			return anc, ErrSignedOut
		}
		if err != nil && gnomeKeyringMissing(cfg) {
			return anc, ErrGnomeKeyringRequired
		}

		if apiToken != "" {
			if !strings.HasPrefix(apiToken, "ap0_") || len(apiToken) != 64 {
				return nil, fmt.Errorf("read invalid PAT token from keyring")
			}
		}
	}

	anc.Client.Transport = basicAuther{
		RoundTripper: anc.Client.Transport,
		PAT:          apiToken,
	}

	if info, err := anc.UserInfo(ctx); err == nil {
		if err := version.MinimumVersionCheck(info.MinimumCliVersion); err != nil {
			return nil, err
		}
	}

	return anc, nil
}

func attachServicePath(orgSlug, serviceSlug string) string {
	return "/orgs/" + url.QueryEscape(orgSlug) + "/services/" + url.QueryEscape(serviceSlug) + "/actions/attach"
}

func (s *Session) AttachService(ctx context.Context, chainSlug string, domains []string, orgSlug, realmSlug, serviceSlug string) (*ServicesXtach200, error) {
	attachInput := AttachOrgServiceJSONRequestBody{
		Domains: domains,
	}
	attachInput.Relationships.Chain.Slug = chainSlug
	attachInput.Relationships.Realm.Slug = realmSlug

	var attachOutput ServicesXtach200
	if err := s.post(ctx, attachServicePath(orgSlug, serviceSlug), attachInput, &attachOutput); err != nil {
		return nil, err
	}
	return &attachOutput, nil
}

func getServiceAttachmentsPath(orgAPID, serviceAPID string) string {
	return fmt.Sprintf("/orgs/%s/services/%s/attachments", url.QueryEscape(orgAPID), url.QueryEscape(serviceAPID))
}

func (s *Session) GetServiceAttachments(ctx context.Context, orgAPID string, serviceAPID string) ([]Attachment, error) {
	var attachments Attachments
	if err := s.get(ctx, getServiceAttachmentsPath(orgAPID, serviceAPID), &attachments); err != nil {
		return nil, err
	}
	return attachments.Items, nil
}

func (s *Session) CreatePATToken(ctx context.Context, deviceCode string) (string, error) {
	reqBody := CreateCliTokenJSONRequestBody{
		DeviceCode: deviceCode,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(reqBody); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "/cli/pat-tokens", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := s.Do(req)
	if err != nil {
		return "", err
	}

	requestId := res.Header.Get("X-Request-Id")

	switch res.StatusCode {
	case http.StatusOK:
		var patTokens *AuthCliPatTokensResponse
		if err = json.NewDecoder(res.Body).Decode(&patTokens); err != nil {
			return "", err
		}
		return patTokens.PatToken, nil
	case http.StatusServiceUnavailable:
		return "", ErrTransient
	case http.StatusBadRequest:
		var errorsRes *Error
		if err = json.NewDecoder(res.Body).Decode(&errorsRes); err != nil {
			return "", err
		}
		switch errorsRes.Type {
		case "urn:anchordev:api:cli-auth:authorization-pending":
			return "", ErrTransient
		case "urn:anchordev:api:cli-auth:expired-device-code":
			return "", fmt.Errorf("Your authorization request has expired, please try again.")
		case "urn:anchordev:api:cli-auth:incorrect-device-code":
			return "", fmt.Errorf("Your authorization request was not found, please try again.")
		default:
			return "", fmt.Errorf("request [%s]: unexpected error: %s", requestId, errorsRes.Detail)
		}
	default:
		return "", fmt.Errorf("request [%s]: unexpected response code: %d", requestId, res.StatusCode)
	}
}

func (s *Session) CreateEAB(ctx context.Context, chainSlug, orgSlug, realmSlug, serviceSlug, subCASlug string) (*Eab, error) {
	var eabInput CreateEabTokenJSONRequestBody
	eabInput.Relationships.Chain.Slug = chainSlug
	eabInput.Relationships.Organization.Slug = orgSlug
	eabInput.Relationships.Realm.Slug = realmSlug
	eabInput.Relationships.Service = &RelationshipsServiceSlug{
		Slug: serviceSlug,
	}
	eabInput.Relationships.SubCa.Slug = subCASlug

	var eabOutput Eab
	if err := s.post(ctx, "/acme/eab-tokens", eabInput, &eabOutput); err != nil {
		return nil, err
	}
	return &eabOutput, nil
}

func (s *Session) CreateService(ctx context.Context, orgSlug, serviceSlug, serverType string, localhostPort *int) (*Service, error) {
	serviceInput := CreateServiceJSONRequestBody{
		Name:          serviceSlug,
		ServerType:    serverType,
		LocalhostPort: localhostPort,
	}
	serviceInput.Relationships.Organization.Slug = orgSlug

	var serviceOutput Service
	if err := s.post(ctx, "/services", serviceInput, &serviceOutput); err != nil {
		return nil, err
	}
	return &serviceOutput, nil
}

func fetchCredentialsPath(orgSlug, realmSlug string) string {
	return "/orgs/" + url.QueryEscape(orgSlug) + "/realms/" + url.QueryEscape(realmSlug) + "/x509/credentials"
}

func (s *Session) FetchCredentials(ctx context.Context, orgSlug, realmSlug string) ([]Credential, error) {
	var creds struct {
		Items []Credential `json:"items,omitempty"`
	}

	if err := s.get(ctx, fetchCredentialsPath(orgSlug, realmSlug), &creds); err != nil {
		return nil, err
	}
	return creds.Items, nil
}

func (s *Session) UserInfo(ctx context.Context) (*Root, error) {
	if s.userInfo != nil {
		return s.userInfo, nil
	}

	if err := s.get(ctx, "", &s.userInfo); err != nil {
		return nil, err
	}
	return s.userInfo, nil
}

func (s *Session) GenerateUserFlowCodes(ctx context.Context, source string) (*AuthCliCodesResponse, error) {
	var codes AuthCliCodesResponse
	if err := s.post(ctx, "/cli/codes", nil, &codes); err != nil {
		return nil, err
	}

	// TODO: should the request POST the signup source instead?
	if source != "" {
		codes.VerificationUri += "?signup_src=" + source
	}
	return &codes, nil
}

func (s *Session) GetOrgs(ctx context.Context) ([]Organization, error) {
	var orgs Organizations
	if err := s.get(ctx, "/orgs", &orgs); err != nil {
		return nil, err
	}
	return orgs.Items, nil
}

func getOrgRealmsPath(orgApid string) string {
	return "/orgs/" + url.QueryEscape(orgApid) + "/realms"
}

func (s *Session) GetOrgRealms(ctx context.Context, orgApid string) ([]Realm, error) {
	var realms Realms
	if err := s.get(ctx, getOrgRealmsPath(orgApid), &realms); err != nil {
		return nil, err
	}
	return realms.Items, nil
}

func getOrgServicesPath(orgSlug string) string {
	return "/orgs/" + url.QueryEscape(orgSlug) + "/services"
}

func (s *Session) GetOrgServices(ctx context.Context, orgSlug string) ([]Service, error) {
	var svc Services
	if err := s.get(ctx, getOrgServicesPath(orgSlug), &svc); err != nil {
		return nil, err
	}
	return svc.Items, nil
}

func getServicePath(orgSlug, serviceSlug string) string {
	return "/orgs/" + url.QueryEscape(orgSlug) + "/services/" + url.QueryEscape(serviceSlug)
}

func (s *Session) GetService(ctx context.Context, orgSlug, serviceSlug string) (*Service, error) {
	var svc Service
	if err := s.get(ctx, getServicePath(orgSlug, serviceSlug), &svc); err != nil {
		if errors.Is(err, NotFoundErr) {
			return nil, nil
		}
		return nil, err
	}
	return &svc, nil
}

func (s *Session) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := s.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		var errorsRes *Error
		if err = json.NewDecoder(res.Body).Decode(&errorsRes); err != nil {
			return err
		}
		requestId := res.Header.Get("X-Request-Id")
		return fmt.Errorf("request [%s]: %w: %s", requestId, StatusCodeError(res.StatusCode), errorsRes.Title)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (s *Session) post(ctx context.Context, path string, in, out any) error {
	var buf bytes.Buffer
	if in != nil {
		if err := json.NewEncoder(&buf).Encode(in); err != nil {
			return err
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", path, &buf)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := s.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		var errorsRes *Error
		if err = json.NewDecoder(res.Body).Decode(&errorsRes); err != nil {
			return err
		}
		requestId := res.Header.Get("X-Request-Id")
		return fmt.Errorf("request [%s]: %w: %s", requestId, StatusCodeError(res.StatusCode), errorsRes.Title)
	}
	return json.NewDecoder(res.Body).Decode(out)
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

	requestId := res.Header.Get("X-Request-Id")

	switch res.StatusCode {
	case http.StatusForbidden:
		return nil, ErrSignedOut
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("request [%s] failed: 500 Internal Server Error", requestId)
	}
	if contentType := res.Header.Get("Content-Type"); !jsonMediaTypes.Matches(contentType) {
		return nil, fmt.Errorf("request [%s]: %d response, expected json content-type, got: %q", requestId, res.StatusCode, contentType)
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

type preferSetter struct {
	http.RoundTripper

	cfg *cli.Config
}

func (s preferSetter) RoundTrip(req *http.Request) (*http.Response, error) {
	path := req.URL.Path

	var value []string

	if s.cfg.Test.Prefer[path].Code != 0 {
		value = append(value, fmt.Sprintf("code=%d", s.cfg.Test.Prefer[path].Code))
	}

	if s.cfg.Test.Prefer[path].Dynamic {
		value = append(value, fmt.Sprintf("dynamic=%t", s.cfg.Test.Prefer[path].Dynamic))
	}

	if s.cfg.Test.Prefer[path].Example != "" {
		value = append(value, fmt.Sprintf("example=%s", s.cfg.Test.Prefer[path].Example))
	}

	if len(value) > 0 {
		req.Header.Set("Prefer", strings.Join(value, " "))
	}

	return s.RoundTripper.RoundTrip(req)
}

type userAgentSetter struct {
	http.RoundTripper
}

func (s userAgentSetter) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", cli.UserAgent())

	return s.RoundTripper.RoundTrip(req)
}

type mediaTypes []string

func (s mediaTypes) Matches(val string) bool {
	media, _, err := mime.ParseMediaType(val)
	if err != nil {
		return false
	}
	return slices.Contains(s, media)
}

type StatusCodeError int

const NotFoundErr = StatusCodeError(http.StatusNotFound)

func (err StatusCodeError) StatusCode() int { return int(err) }
func (err StatusCodeError) Error() string   { return fmt.Sprintf("unexpected %d status response", err) }

func gnomeKeyringMissing(cfg *cli.Config) bool {
	if cfg.GOOS() != "linux" {
		return false
	}
	if path, _ := exec.LookPath("gnome-keyring-daemon"); path != "" {
		return false
	}
	return true
}
