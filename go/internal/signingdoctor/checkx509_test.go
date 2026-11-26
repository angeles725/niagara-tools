package signingdoctor

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
	"time"
)

// CheckX509 is what CheckCert runs on a certificate that parses: same rows, same order (signing-inventory
// calls it on the certificates of a keystore).
func TestCheckX509IsTheRowsOfCheckCert(t *testing.T) {
	path := certFile(t, &x509.Certificate{ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning}}, false)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(data)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	got, want := CheckX509(cert, now), CheckCert(data, now)
	if len(got) != 4 || len(want) != 4 {
		t.Fatalf("rows: %v / %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("row %d: %v != %v", i, got[i], want[i])
		}
	}
}
