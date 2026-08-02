package controlclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/flidai/autback/internal/gen/rtest/v1/autbackv1connect"
)

func New(baseURL, token, caCertFile string) (autbackv1connect.ControlServiceClient, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("autback service URL must be absolute")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return nil, errors.New("autback service URL must use HTTPS or local HTTP")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS13}
	if caCertFile != "" {
		roots, err := x509.SystemCertPool()
		if err != nil || roots == nil {
			roots = x509.NewCertPool()
		}
		data, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, err
		}
		if !roots.AppendCertsFromPEM(data) {
			return nil, errors.New("autback control CA file contains no certificates")
		}
		transport.TLSClientConfig.RootCAs = roots
	}
	var roundTripper http.RoundTripper = transport
	if token != "" {
		roundTripper = authorizationTransport{token: token, next: transport}
	}
	httpClient := &http.Client{Transport: roundTripper}
	return autbackv1connect.NewControlServiceClient(httpClient, parsed.String()), nil
}

type authorizationTransport struct {
	token string
	next  http.RoundTripper
}

func (t authorizationTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header = request.Header.Clone()
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return t.next.RoundTrip(clone)
}
