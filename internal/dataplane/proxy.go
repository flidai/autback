// Package dataplane provides a stable loopback endpoint for operation-scoped
// data-plane connections whose upstream mTLS credentials rotate over time.
package dataplane

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Credential struct {
	Endpoint       string
	ServerName     string
	CAPEM          []byte
	CertificatePEM []byte
	PrivateKeyPEM  []byte
	ExpiresAt      time.Time
}

type upstream struct {
	address   string
	tlsConfig *tls.Config
	expiresAt time.Time
}

// Proxy accepts plaintext connections on an ephemeral loopback address and
// opens a new mTLS upstream connection with the latest validated credential.
// Existing streams remain uninterrupted when Update rotates the credential;
// reconnects automatically use the new generation.
type Proxy struct {
	listener net.Listener
	cancel   context.CancelFunc
	current  atomic.Pointer[upstream]
	closed   atomic.Bool
	done     chan struct{}
	once     sync.Once
	errMu    sync.Mutex
	serveErr error
}

func Start(parent context.Context, credential Credential) (*Proxy, error) {
	if parent == nil {
		return nil, errors.New("data-plane proxy context is required")
	}
	configured, err := validateCredential(credential)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for local data-plane proxy: %w", err)
	}
	ctx, cancel := context.WithCancel(parent)
	proxy := &Proxy{listener: listener, cancel: cancel, done: make(chan struct{})}
	proxy.current.Store(configured)
	go func() {
		serveErr := proxy.serve(ctx)
		proxy.errMu.Lock()
		proxy.serveErr = serveErr
		proxy.errMu.Unlock()
		close(proxy.done)
	}()
	go func() {
		select {
		case <-ctx.Done():
			proxy.closed.Store(true)
			_ = proxy.listener.Close()
		case <-proxy.done:
		}
	}()
	return proxy, nil
}

func (p *Proxy) Address() string {
	if p == nil || p.listener == nil {
		return ""
	}
	return p.listener.Addr().String()
}

// Update validates the complete replacement before publishing it. A rejected
// update leaves the previous credential available for reconnects.
func (p *Proxy) Update(credential Credential) error {
	if p == nil || p.listener == nil || p.closed.Load() {
		return errors.New("data-plane proxy is not running")
	}
	configured, err := validateCredential(credential)
	if err != nil {
		return err
	}
	p.current.Store(configured)
	return nil
}

func (p *Proxy) Close() error {
	if p == nil {
		return nil
	}
	p.once.Do(func() {
		p.closed.Store(true)
		p.cancel()
		_ = p.listener.Close()
	})
	<-p.done
	p.errMu.Lock()
	defer p.errMu.Unlock()
	return p.serveErr
}

func (p *Proxy) serve(ctx context.Context) error {
	var connections sync.WaitGroup
	defer connections.Wait()
	for {
		connection, err := p.listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept local data-plane connection: %w", err)
		}
		connections.Add(1)
		go func() {
			defer connections.Done()
			p.proxyConnection(ctx, connection)
		}()
	}
}

func (p *Proxy) proxyConnection(ctx context.Context, local net.Conn) {
	defer local.Close()
	configured := p.current.Load()
	if configured == nil || !configured.expiresAt.IsZero() && !time.Now().Before(configured.expiresAt) {
		return
	}
	dialer := tls.Dialer{
		NetDialer: &net.Dialer{Timeout: 10 * time.Second},
		Config:    configured.tlsConfig,
	}
	upstreamConnection, err := dialer.DialContext(ctx, "tcp", configured.address)
	if err != nil {
		return
	}
	defer upstreamConnection.Close()
	copied := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(upstreamConnection, local); copied <- struct{}{} }()
	go func() { _, _ = io.Copy(local, upstreamConnection); copied <- struct{}{} }()
	copiedCount := 0
	select {
	case <-ctx.Done():
	case <-copied:
		copiedCount++
	}
	_ = local.Close()
	_ = upstreamConnection.Close()
	for copiedCount < 2 {
		<-copied
		copiedCount++
	}
}

func validateCredential(credential Credential) (*upstream, error) {
	address, err := normalizeEndpoint(credential.Endpoint)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(credential.ServerName) == "" {
		return nil, errors.New("data-plane server name is required")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(credential.CAPEM) {
		return nil, errors.New("data-plane CA contains no certificates")
	}
	certificate, err := tls.X509KeyPair(credential.CertificatePEM, credential.PrivateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("load data-plane client credential: %w", err)
	}
	leaf, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("parse data-plane client certificate: %w", err)
	}
	certificate.Leaf = leaf
	expiresAt := credential.ExpiresAt
	if expiresAt.IsZero() || leaf.NotAfter.Before(expiresAt) {
		expiresAt = leaf.NotAfter
	}
	if !time.Now().Before(expiresAt) {
		return nil, errors.New("data-plane client credential has expired")
	}
	return &upstream{
		address: address,
		tlsConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
			RootCAs:    roots,
			Certificates: []tls.Certificate{
				certificate,
			},
			ServerName: credential.ServerName,
		},
		expiresAt: expiresAt,
	}, nil
}

func normalizeEndpoint(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", errors.New("data-plane endpoint is required")
	}
	if strings.Contains(endpoint, "://") {
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Scheme != "tcp" || parsed.Host == "" || parsed.Path != "" {
			return "", fmt.Errorf("invalid data-plane TCP endpoint %q", endpoint)
		}
		endpoint = parsed.Host
	}
	if _, _, err := net.SplitHostPort(endpoint); err != nil {
		return "", fmt.Errorf("invalid data-plane endpoint %q: %w", endpoint, err)
	}
	return endpoint, nil
}
