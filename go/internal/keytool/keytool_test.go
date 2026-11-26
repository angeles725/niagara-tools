package keytool

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func certPEM(t *testing.T, cn string) string {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: cn},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &k.PublicKey, k)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func TestParseReadsAliasTypeAndTheFirstCertificateOfTheChain(t *testing.T) {
	text := "Keystore type: JCEKS\n\nYour keystore contains 3 entries\n\n" +
		"Alias name: mine\nCreation date: Nov 26, 2026\nEntry type: PrivateKeyEntry\nCertificate chain length: 2\n" +
		"Certificate[1]:\n" + certPEM(t, "leaf") + "Certificate[2]:\n" + certPEM(t, "root") + "\n*****\n" +
		"Alias name: trusted, with comma\nEntry type: trustedCertEntry\n" + certPEM(t, "ca") + "\n" +
		"Alias name: broken\nEntry type: PrivateKeyEntry\n-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n"
	got := parse(text)
	if len(got) != 3 {
		t.Fatalf("entries: %+v", got)
	}
	if got[0].Alias != "mine" || !got[0].KeyEntry || got[0].Cert == nil || got[0].Cert.Subject.CommonName != "leaf" {
		t.Errorf("first: %+v", got[0])
	}
	if got[1].Alias != "trusted, with comma" || got[1].KeyEntry || got[1].Cert.Subject.CommonName != "ca" {
		t.Errorf("second: %+v", got[1])
	}
	if got[2].Alias != "broken" || got[2].Cert != nil {
		t.Errorf("a certificate that does not parse is none: %+v", got[2])
	}
}

func TestListWithoutAPasswordRunsNothing(t *testing.T) {
	if _, err := List("/nonexistent/keytool", "/x", "JCEKS", ""); !errors.Is(err, ErrNoPassword) {
		t.Errorf("err = %v", err)
	}
	if _, err := List("/nonexistent/keytool", "/x", "JCEKS", "p"); !errors.Is(err, ErrUnreadable) {
		t.Errorf("a keytool that cannot run: err = %v", err)
	}
}

func TestFindPrefersJavaHomeThenPath(t *testing.T) {
	t.Setenv("PATH", "")
	if _, err := Find("/nonexistent"); !errors.Is(err, ErrNoKeytool) {
		t.Errorf("err = %v", err)
	}
	home := t.TempDir()
	if err := writeTool(home + "/bin/keytool"); err != nil {
		t.Fatal(err)
	}
	if p, err := Find(home); err != nil || p != home+"/bin/keytool" {
		t.Errorf("Find = %q, %v", p, err)
	}
}

func writeTool(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755)
}
