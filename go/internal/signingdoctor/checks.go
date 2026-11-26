package signingdoctor

import (
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
)

// Row is one check: Status PASS, WARN or FAIL, the check id and what it found.
type Row struct{ Status, Check, Detail string }

// CheckCert checks a PEM or DER certificate: it parses, is not a CA, carries the code-signing
// usage and is in date at now.
func CheckCert(data []byte, now time.Time) []Row {
	der := data
	if block, _ := pem.Decode(data); block != nil {
		if block.Type != "CERTIFICATE" {
			return []Row{{"FAIL", "cert-parse", "the PEM block is " + block.Type + ", not a CERTIFICATE"}}
		}
		der = block.Bytes
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return []Row{{"FAIL", "cert-parse", "not a certificate (PEM or DER): " + err.Error()}}
	}
	return CheckX509(cert, now)
}

// CheckX509 checks a parsed certificate: its subject (cert-parse), that it is not a CA, that it carries the
// code-signing usage and that it is in date at now. signing-inventory checks the certificates of a keystore
// with it.
func CheckX509(cert *x509.Certificate, now time.Time) []Row {
	rows := []Row{{"PASS", "cert-parse", "subject " + cert.Subject.String()}}
	if cert.BasicConstraintsValid && cert.IsCA {
		rows = append(rows, Row{"FAIL", "cert-ca", "the certificate is a CA (CA:TRUE): export the code-signing certificate " +
			"of the signing key, not the CA that issued it"})
	} else {
		rows = append(rows, Row{"PASS", "cert-ca", "not a CA"})
	}
	rows = append(rows, usageRow(cert))
	switch {
	case now.Before(cert.NotBefore):
		rows = append(rows, Row{"FAIL", "cert-validity", "not valid before " + cert.NotBefore.UTC().Format(time.RFC3339)})
	case now.After(cert.NotAfter):
		rows = append(rows, Row{"FAIL", "cert-validity", "expired " + cert.NotAfter.UTC().Format(time.RFC3339)})
	default:
		rows = append(rows, Row{"PASS", "cert-validity", "valid until " + cert.NotAfter.UTC().Format(time.RFC3339)})
	}
	return rows
}

// usageRow is the code-signing usage row: an extended key usage list without code signing, or a
// key usage without digital signature, is a FAIL; no usage extension at all is a WARN (any purpose).
func usageRow(cert *x509.Certificate) Row {
	if cert.KeyUsage != 0 && cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		return Row{"FAIL", "cert-usage", "the key usage has no digitalSignature: no code-signing usage"}
	}
	if len(cert.ExtKeyUsage) == 0 && len(cert.UnknownExtKeyUsage) == 0 {
		return Row{"WARN", "cert-usage", "no extended key usage: the certificate does not say it is for code signing"}
	}
	for _, u := range cert.ExtKeyUsage {
		if u == x509.ExtKeyUsageCodeSigning || u == x509.ExtKeyUsageAny {
			return Row{"PASS", "cert-usage", "code signing"}
		}
	}
	return Row{"FAIL", "cert-usage", fmt.Sprintf("no code-signing extended key usage (%d other usage(s) listed)",
		len(cert.ExtKeyUsage)+len(cert.UnknownExtKeyUsage))}
}

var (
	oidPBES2 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 13}
	// AES-GCM schemes (NIST CSOR 2.16.840.1.101.3.4.1.{6,26,46}); OpenSSL 3 cannot decrypt PBES2 with them.
	gcmSchemes = map[string]string{"2.16.840.1.101.3.4.1.6": "AES-128-GCM", "2.16.840.1.101.3.4.1.26": "AES-192-GCM",
		"2.16.840.1.101.3.4.1.46": "AES-256-GCM"}
)

type algorithm struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue `asn1:"optional"`
}

type encryptedKey struct {
	Algo algorithm
	Data []byte
}

type pbes2Params struct {
	KDF    algorithm
	Scheme algorithm
}

// CheckKey checks a private key export: the form is recognised and, for a PBES2 password-protected
// one, its cipher is not AES-GCM, which the Workbench export uses and OpenSSL 3 cannot read.
func CheckKey(data []byte) []Row {
	block, _ := pem.Decode(data)
	if block == nil {
		return []Row{{"FAIL", "key-format", "not a PEM private key export: nothing here OpenSSL can read"}}
	}
	if block.Type != "ENCRYPTED PRIVATE KEY" {
		if strings.HasSuffix(block.Type, "PRIVATE KEY") {
			return []Row{{"PASS", "key-format", "an unencrypted " + block.Type + " block"}}
		}
		return []Row{{"FAIL", "key-format", "the PEM block is " + block.Type + ", not a private key"}}
	}
	var info encryptedKey
	if rest, err := asn1.Unmarshal(block.Bytes, &info); err != nil || len(rest) != 0 {
		return []Row{{"FAIL", "key-format", "the ENCRYPTED PRIVATE KEY block is not a valid EncryptedPrivateKeyInfo"}}
	}
	if !info.Algo.Algorithm.Equal(oidPBES2) {
		return []Row{{"PASS", "key-format", "password-protected, not PBES2 (" + info.Algo.Algorithm.String() + ")"}}
	}
	var params pbes2Params
	if rest, err := asn1.Unmarshal(info.Algo.Parameters.FullBytes, &params); err != nil || len(rest) != 0 {
		return []Row{{"FAIL", "key-format", "the PBES2 parameters cannot be read"}}
	}
	if name, gcm := gcmSchemes[params.Scheme.Algorithm.String()]; gcm {
		return []Row{{"FAIL", "key-format", "PBES2 with " + name + " (the Workbench private key export): OpenSSL 3 cannot read it; " +
			"convert it with Java, which can, never with OpenSSL"}}
	}
	return []Row{{"PASS", "key-format", "PBES2 with cipher " + params.Scheme.Algorithm.String() + ", readable by OpenSSL 3"}}
}
