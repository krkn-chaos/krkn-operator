package identity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"testing"
	"time"
)

func TestNewHTTPClient(t *testing.T) {
	t.Run("uses shared default client without custom TLS settings", func(t *testing.T) {
		got, err := NewHTTPClient("", false)
		if err != nil {
			t.Fatal(err)
		}
		if got != http.DefaultClient {
			t.Fatal("expected shared default client")
		}
	})

	t.Run("rejects invalid CA PEM", func(t *testing.T) {
		if _, err := NewHTTPClient("not a certificate", false); err == nil {
			t.Fatal("expected invalid CA error")
		}
	})

	t.Run("trusts custom CA and reuses the cached client", func(t *testing.T) {
		caPEM, cert := testCACertificate(t)
		first, err := NewHTTPClient(caPEM, false)
		if err != nil {
			t.Fatal(err)
		}
		second, err := NewHTTPClient(caPEM, false)
		if err != nil {
			t.Fatal(err)
		}
		if first != second {
			t.Fatal("expected the configured client to be reused")
		}
		transport, ok := first.Transport.(*http.Transport)
		if !ok || transport.TLSClientConfig == nil || transport.TLSClientConfig.RootCAs == nil {
			t.Fatal("expected custom root CAs on the transport")
		}
		for _, subject := range transport.TLSClientConfig.RootCAs.Subjects() {
			if string(subject) == string(cert.RawSubject) {
				return
			}
		}
		t.Fatal("custom CA was not added to the root pool")
	})

	t.Run("allows explicitly skipped verification and reuses its client", func(t *testing.T) {
		first, err := NewHTTPClient("", true)
		if err != nil {
			t.Fatal(err)
		}
		second, err := NewHTTPClient("", true)
		if err != nil {
			t.Fatal(err)
		}
		if first != second {
			t.Fatal("expected insecure configured client to be reused")
		}
		transport, ok := first.Transport.(*http.Transport)
		if !ok || transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
			t.Fatal("expected TLS verification to be disabled on the transport")
		}
	})
}

func testCACertificate(t *testing.T) (string, *x509.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: "krkn test CA"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), cert
}
