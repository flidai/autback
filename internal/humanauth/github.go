package humanauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/flidai/autback/internal/control"
)

const (
	defaultGitHubAuthorizeURL = "https://github.com/login/oauth/authorize"
	defaultGitHubTokenURL     = "https://github.com/login/oauth/access_token"
	defaultGitHubAPIURL       = "https://api.github.com"
)

type GitHubConfig struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
	AuthorizeURL string
	TokenURL     string
	APIURL       string
	HTTPClient   *http.Client
}

type GitHub struct {
	clientID     string
	clientSecret string
	callbackURL  string
	authorizeURL string
	tokenURL     string
	apiURL       string
	httpClient   *http.Client
}

func NewGitHub(config GitHubConfig) (*GitHub, error) {
	if strings.TrimSpace(config.ClientID) == "" || strings.TrimSpace(config.ClientSecret) == "" {
		return nil, errors.New("GitHub client ID and secret are required")
	}
	callback, err := url.Parse(config.CallbackURL)
	if err != nil || callback.Scheme != "https" || callback.Host == "" {
		return nil, errors.New("GitHub callback URL must use HTTPS")
	}
	if config.AuthorizeURL == "" {
		config.AuthorizeURL = defaultGitHubAuthorizeURL
	}
	if config.TokenURL == "" {
		config.TokenURL = defaultGitHubTokenURL
	}
	if config.APIURL == "" {
		config.APIURL = defaultGitHubAPIURL
	}
	for _, endpoint := range []string{config.AuthorizeURL, config.TokenURL, config.APIURL} {
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return nil, errors.New("GitHub endpoints must be absolute URLs")
		}
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &GitHub{
		clientID: config.ClientID, clientSecret: config.ClientSecret, callbackURL: callback.String(),
		authorizeURL: config.AuthorizeURL, tokenURL: config.TokenURL, apiURL: strings.TrimRight(config.APIURL, "/"), httpClient: config.HTTPClient,
	}, nil
}

func (g *GitHub) AuthorizationURL(state, codeChallenge string) string {
	query := url.Values{
		"client_id": {g.clientID}, "redirect_uri": {g.callbackURL}, "state": {state},
		"code_challenge": {codeChallenge}, "code_challenge_method": {"S256"}, "allow_signup": {"false"},
	}
	return g.authorizeURL + "?" + query.Encode()
}

func (g *GitHub) Exchange(ctx context.Context, code, verifier string) (GitHubIdentity, error) {
	form := url.Values{
		"client_id": {g.clientID}, "client_secret": {g.clientSecret}, "code": {code},
		"redirect_uri": {g.callbackURL}, "code_verifier": {verifier},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, g.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return GitHubIdentity{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	response, err := g.httpClient.Do(request)
	if err != nil {
		return GitHubIdentity{}, fmt.Errorf("exchange GitHub authorization: %w", err)
	}
	defer response.Body.Close()
	var token struct {
		AccessToken      string `json:"access_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if response.StatusCode != http.StatusOK || json.NewDecoder(response.Body).Decode(&token) != nil || token.AccessToken == "" {
		return GitHubIdentity{}, fmt.Errorf("GitHub authorization exchange failed: %s", firstNonEmpty(token.ErrorDescription, token.Error, response.Status))
	}
	return g.identity(ctx, "/user", token.AccessToken)
}

func (g *GitHub) Lookup(ctx context.Context, login string) (GitHubIdentity, error) {
	login = strings.TrimSpace(login)
	if login == "" || strings.Contains(login, "/") {
		return GitHubIdentity{}, errors.New("GitHub login is required")
	}
	return g.identity(ctx, "/users/"+url.PathEscape(login), "")
}

func (g *GitHub) Resolve(ctx context.Context, login string) (control.ExternalIdentity, error) {
	identity, err := g.Lookup(ctx, login)
	if err != nil {
		return control.ExternalIdentity{}, err
	}
	return control.ExternalIdentity{Provider: "github", Subject: identity.Subject, Login: identity.Login}, nil
}

func (g *GitHub) identity(ctx context.Context, path, token string) (GitHubIdentity, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, g.apiURL+path, nil)
	if err != nil {
		return GitHubIdentity{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "autback")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := g.httpClient.Do(request)
	if err != nil {
		return GitHubIdentity{}, fmt.Errorf("read GitHub identity: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return GitHubIdentity{}, fmt.Errorf("read GitHub identity: %s", response.Status)
	}
	var user struct {
		ID    json.Number `json:"id"`
		Login string      `json:"login"`
		Name  string      `json:"name"`
	}
	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&user); err != nil {
		return GitHubIdentity{}, fmt.Errorf("decode GitHub identity: %w", err)
	}
	subject := user.ID.String()
	if _, err := strconv.ParseInt(subject, 10, 64); err != nil || strings.TrimSpace(user.Login) == "" {
		return GitHubIdentity{}, errors.New("GitHub identity response is incomplete")
	}
	return GitHubIdentity{Subject: subject, Login: user.Login, Name: user.Name}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return "unknown error"
}
