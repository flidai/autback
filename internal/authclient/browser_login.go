package authclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type BrowserLogin struct {
	DeviceCode              string
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresAt               time.Time
	Interval                time.Duration
}

type BrowserLoginToken struct {
	Token   string
	TokenID string
}

type BrowserLoginClient struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func NewBrowserLoginClient(baseURL string, httpClient *http.Client) (*BrowserLoginClient, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Scheme != "https" && parsed.Scheme != "http" {
		return nil, errors.New("browser login URL must be absolute HTTP(S)")
	}
	if parsed.Scheme == "http" && !isLoopbackHost(parsed.Hostname()) {
		return nil, errors.New("browser login requires HTTPS except on loopback")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &BrowserLoginClient{baseURL: parsed, httpClient: httpClient}, nil
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (c *BrowserLoginClient) Start(ctx context.Context, deviceName string) (BrowserLogin, error) {
	var response struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresInSeconds        int64  `json:"expires_in_seconds"`
		IntervalSeconds         int64  `json:"interval_seconds"`
	}
	status, err := c.post(ctx, "/auth/cli/start", map[string]string{"device_name": deviceName}, &response)
	if err != nil {
		return BrowserLogin{}, err
	}
	if status != http.StatusCreated || response.DeviceCode == "" || response.UserCode == "" || response.VerificationURIComplete == "" || response.ExpiresInSeconds <= 0 {
		return BrowserLogin{}, fmt.Errorf("start browser login: unexpected HTTP %d response", status)
	}
	interval := time.Duration(response.IntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return BrowserLogin{
		DeviceCode: response.DeviceCode, UserCode: response.UserCode, VerificationURI: response.VerificationURI,
		VerificationURIComplete: response.VerificationURIComplete, ExpiresAt: time.Now().UTC().Add(time.Duration(response.ExpiresInSeconds) * time.Second), Interval: interval,
	}, nil
}

func (c *BrowserLoginClient) Poll(ctx context.Context, deviceCode string) (BrowserLoginToken, bool, error) {
	var response struct {
		Status  string `json:"status"`
		Token   string `json:"token"`
		TokenID string `json:"token_id"`
	}
	status, err := c.post(ctx, "/auth/cli/token", map[string]string{"device_code": deviceCode}, &response)
	if err != nil {
		return BrowserLoginToken{}, false, err
	}
	if status == http.StatusAccepted && response.Status == "authorization_pending" {
		return BrowserLoginToken{}, true, nil
	}
	if status != http.StatusOK || response.Status != "authorized" || response.Token == "" {
		return BrowserLoginToken{}, false, fmt.Errorf("complete browser login: HTTP %d %s", status, response.Status)
	}
	return BrowserLoginToken{Token: response.Token, TokenID: response.TokenID}, false, nil
}

func (c *BrowserLoginClient) post(ctx context.Context, path string, input, output any) (int, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return 0, err
	}
	target := c.baseURL.ResolveReference(&url.URL{Path: path})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(output); err != nil {
		return response.StatusCode, fmt.Errorf("decode browser login response: %w", err)
	}
	return response.StatusCode, nil
}
