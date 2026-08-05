package dataplane_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/flidai/autback/internal/control/pki"
	"github.com/flidai/autback/internal/dataplane"
)

func TestProxyUsesRefreshedCredentialForNewUpstreamConnections(t *testing.T) {
	authority, err := pki.Ensure(filepath.Join(t.TempDir(), "pki"), []string{"127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	serials := make(chan string, 2)
	target := tlsEchoServer(t, authority.ServerTLSConfig(pki.OperationBuild, func(kind pki.Operation, id string) bool {
		return kind == pki.OperationBuild && id == "build-1"
	}), serials)
	first, err := authority.Issue(pki.OperationBuild, "build-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	second, err := authority.Issue(pki.OperationBuild, "build-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	proxy, err := dataplane.Start(context.Background(), credential(target, first))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = proxy.Close() })

	assertEcho(t, proxy.Address(), "first")
	if got, want := <-serials, certificateSerial(t, first.CertificatePEM); got != want {
		t.Fatalf("first client certificate serial = %q, want %q", got, want)
	}
	if err := proxy.Update(credential(target, second)); err != nil {
		t.Fatal(err)
	}
	assertEcho(t, proxy.Address(), "second")
	if got, want := <-serials, certificateSerial(t, second.CertificatePEM); got != want {
		t.Fatalf("refreshed client certificate serial = %q, want %q", got, want)
	}
}

func TestProxyRejectsInvalidCredentialUpdateAndKeepsServing(t *testing.T) {
	authority, err := pki.Ensure(filepath.Join(t.TempDir(), "pki"), []string{"127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	serials := make(chan string, 2)
	target := tlsEchoServer(t, authority.ServerTLSConfig(pki.OperationJob, func(kind pki.Operation, id string) bool {
		return kind == pki.OperationJob && id == "job-1"
	}), serials)
	issued, err := authority.Issue(pki.OperationJob, "job-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	proxy, err := dataplane.Start(context.Background(), credential(target, issued))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = proxy.Close() })

	invalid := credential(target, issued)
	invalid.PrivateKeyPEM = []byte("not a private key")
	if err := proxy.Update(invalid); err == nil {
		t.Fatal("invalid credential update succeeded")
	}
	assertEcho(t, proxy.Address(), "still-valid")
	if got, want := <-serials, certificateSerial(t, issued.CertificatePEM); got != want {
		t.Fatalf("client certificate serial after rejected update = %q, want %q", got, want)
	}
}

func TestProxyClosesListenerWhenParentIsCancelled(t *testing.T) {
	authority, err := pki.Ensure(filepath.Join(t.TempDir(), "pki"), []string{"127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	target := tlsEchoServer(t, authority.ServerTLSConfig(pki.OperationJob, func(pki.Operation, string) bool { return true }), make(chan string, 1))
	issued, err := authority.Issue(pki.OperationJob, "job-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	proxy, err := dataplane.Start(ctx, credential(target, issued))
	if err != nil {
		t.Fatal(err)
	}
	address := proxy.Address()
	cancel()
	if err := proxy.Close(); err != nil {
		t.Fatal(err)
	}
	if connection, err := net.DialTimeout("tcp", address, 100*time.Millisecond); err == nil {
		_ = connection.Close()
		t.Fatal("proxy listener still accepts connections after parent cancellation")
	}
	if err := proxy.Update(credential(target, issued)); err == nil {
		t.Fatal("closed proxy accepted a credential update")
	}
}

func credential(endpoint string, issued pki.Credential) dataplane.Credential {
	return dataplane.Credential{
		Endpoint: endpoint, ServerName: issued.ServerName, CAPEM: issued.CAPEM,
		CertificatePEM: issued.CertificatePEM, PrivateKeyPEM: issued.PrivateKeyPEM,
		ExpiresAt: issued.ExpiresAt,
	}
}

func tlsEchoServer(t *testing.T, config *tls.Config, serials chan<- string) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			connection, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			go func() {
				defer connection.Close()
				tlsConnection := tls.Server(connection, config)
				if handshakeErr := tlsConnection.Handshake(); handshakeErr != nil {
					return
				}
				state := tlsConnection.ConnectionState()
				if len(state.PeerCertificates) == 0 {
					return
				}
				serials <- state.PeerCertificates[0].SerialNumber.String()
				_, _ = io.Copy(tlsConnection, tlsConnection)
			}()
		}
	}()
	return listener.Addr().String()
}

func assertEcho(t *testing.T, address, message string) {
	t.Helper()
	connection, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if _, err := connection.Write([]byte(message)); err != nil {
		t.Fatal(err)
	}
	response := make([]byte, len(message))
	if _, err := io.ReadFull(connection, response); err != nil {
		t.Fatal(err)
	}
	if string(response) != message {
		t.Fatalf("echo = %q, want %q", response, message)
	}
}

func certificateSerial(t *testing.T, certificatePEM []byte) string {
	t.Helper()
	block, _ := pem.Decode(certificatePEM)
	if block == nil {
		t.Fatal("certificate PEM contains no certificate")
	}
	parsed, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	return parsed.SerialNumber.String()
}
