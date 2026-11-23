package signingdoctor

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Main(args, cli.Stdio{Out: &out, Err: &errb})
	return code, out.String(), errb.String()
}

// certFile writes a certificate made from tmpl, self-signed, as PEM (or DER when der) and returns its path.
func certFile(t *testing.T, tmpl *x509.Certificate, der bool) string {
	t.Helper()
	signer, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl.SerialNumber = big.NewInt(7)
	tmpl.Subject = pkix.Name{CommonName: "test signer"}
	if tmpl.NotBefore.IsZero() {
		tmpl.NotBefore = time.Now().Add(-time.Hour)
		tmpl.NotAfter = time.Now().Add(24 * time.Hour)
	}
	raw, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &signer.PublicKey, signer)
	if err != nil {
		t.Fatal(err)
	}
	data := raw
	if !der {
		data = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: raw})
	}
	path := filepath.Join(t.TempDir(), "cert")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func codeSigning() *x509.Certificate {
	return &x509.Certificate{ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning}, KeyUsage: x509.KeyUsageDigitalSignature}
}

// Retro mcpn4-history-batch-tools Δ5: the certificate exported for signing was the CA.
func TestACodeSigningCertificatePasses(t *testing.T) {
	for _, der := range []bool{false, true} {
		code, out, errOut := run(t, "--cert", certFile(t, codeSigning(), der))
		if code != ExitOK || errOut != "" || strings.Contains(out, "FAIL") {
			t.Errorf("der=%v: exit %d out %q err %q", der, code, out, errOut)
		}
		if !strings.Contains(out, "PASS  cert-ca") || !strings.Contains(out, "PASS  cert-usage") {
			t.Errorf("der=%v: rows missing:\n%s", der, out)
		}
	}
}

func TestACaCertificateIsRefused(t *testing.T) {
	ca := &x509.Certificate{IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	code, out, _ := run(t, "--cert", certFile(t, ca, false))
	if code != ExitFail || !strings.Contains(out, "FAIL  cert-ca") || !strings.Contains(out, "CA:TRUE") ||
		!strings.Contains(out, "not the CA that issued it") {
		t.Errorf("exit %d out:\n%s", code, out)
	}
}

func TestACertificateWithoutCodeSigningUsageIsRefused(t *testing.T) {
	web := &x509.Certificate{ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, KeyUsage: x509.KeyUsageDigitalSignature}
	code, out, _ := run(t, "--cert", certFile(t, web, false))
	if code != ExitFail || !strings.Contains(out, "FAIL  cert-usage") || !strings.Contains(out, "no code-signing") {
		t.Errorf("exit %d out:\n%s", code, out)
	}
	code, out, _ = run(t, "--cert", certFile(t, &x509.Certificate{}, false))
	if code != ExitOK || !strings.Contains(out, "WARN  cert-usage") {
		t.Errorf("no usage at all: exit %d out:\n%s", code, out)
	}
}

func TestAnExpiredCertificateIsRefused(t *testing.T) {
	old := codeSigning()
	old.NotBefore, old.NotAfter = time.Now().Add(-48*time.Hour), time.Now().Add(-24*time.Hour)
	code, out, _ := run(t, "--cert", certFile(t, old, false))
	if code != ExitFail || !strings.Contains(out, "FAIL  cert-validity") {
		t.Errorf("exit %d out:\n%s", code, out)
	}
}

// protectedExport writes a PBES2 password-protected private key export, PEM block type
// "ENCRYPTED PRIVATE KEY", whose encryption scheme is the given OID; the bytes are a structure of the
// right shape, not a usable key.
func protectedExport(t *testing.T, scheme asn1.ObjectIdentifier) string {
	t.Helper()
	type algorithm struct {
		Algorithm  asn1.ObjectIdentifier
		Parameters asn1.RawValue `asn1:"optional"`
	}
	kdfParams, err := asn1.Marshal(struct {
		Salt  []byte
		Iter  int
		KeyID int
	}{[]byte("saltsalt"), 10000, 32})
	if err != nil {
		t.Fatal(err)
	}
	schemeParams, err := asn1.Marshal([]byte("twelve bytes"))
	if err != nil {
		t.Fatal(err)
	}
	pbes2, err := asn1.Marshal(struct {
		KDF    algorithm
		Scheme algorithm
	}{algorithm{asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 12}, asn1.RawValue{FullBytes: kdfParams}},
		algorithm{scheme, asn1.RawValue{FullBytes: schemeParams}}})
	if err != nil {
		t.Fatal(err)
	}
	info, err := asn1.Marshal(struct {
		Algo algorithm
		Data []byte
	}{algorithm{asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 13}, asn1.RawValue{FullBytes: pbes2}}, []byte("ciphertext")})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "wb-export.txt")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "ENCRYPTED PRIVATE KEY", Bytes: info}), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// The Workbench private key export (PBES2 + AES-256-GCM) is unreadable by OpenSSL 3: say so before the build.
func TestAnAesGcmKeyExportIsRefusedWithTheJavaRoute(t *testing.T) {
	code, out, _ := run(t, "--key", protectedExport(t, asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 46}))
	for _, part := range []string{"FAIL  key-format", "AES-256-GCM", "OpenSSL", "Java"} {
		if code != ExitFail || !strings.Contains(out, part) {
			t.Errorf("exit %d missing %q in:\n%s", code, part, out)
		}
	}
}

func TestAnAesCbcKeyExportPasses(t *testing.T) {
	code, out, _ := run(t, "--key", protectedExport(t, asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 42}))
	if code != ExitOK || !strings.Contains(out, "PASS  key-format") {
		t.Errorf("exit %d out:\n%s", code, out)
	}
}

func TestAnUnrecognisedKeyFileIsRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wb-export.txt")
	if err := os.WriteFile(path, []byte("not a key"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, _ := run(t, "--key", path)
	if code != ExitFail || !strings.Contains(out, "FAIL  key-format") {
		t.Errorf("exit %d out:\n%s", code, out)
	}
}

// An option given with an empty value (an unset variable in --cert "$CERT") is a usage error, never
// read as "not given": the other file's rows alone must not pass the gate.
func TestAnEmptyOptionIsAUsageError(t *testing.T) {
	key := filepath.Join(t.TempDir(), "wb-export.txt")
	if err := os.WriteFile(key, []byte("not a key"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"--cert", "", "--key", key}, {"--cert=", "--key", key}, {"--key", "", "--cert", key}} {
		code, out, errOut := run(t, args...)
		if code != ExitUsage || out != "" || !strings.Contains(errOut, "is empty") {
			t.Errorf("%v: exit %d out %q err %q", args, code, out, errOut)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent")
	for _, args := range [][]string{{}, {"extra"}, {"--cert", missing}, {"--key", missing}, {"--nope"}} {
		code, out, errOut := run(t, args...)
		if code != ExitUsage || out != "" || !strings.Contains(errOut, "signing-doctor") {
			t.Errorf("%v: exit %d out %q err %q", args, code, out, errOut)
		}
	}
}
