// Package keytool lists the certificates of a JKS/JCEKS/PKCS12 keystore by running the JDK's `keytool`: Go's
// standard library cannot read a JKS or JCEKS store. Only certificates are read (`-list -rfc` prints no
// key). The store password goes to keytool through its environment (-storepass:env), never on its command
// line, which any local user can read, and nothing keytool prints on stderr is ever returned: its messages
// can quote what it was given.
package keytool

import (
	"bufio"
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Errors of List and Find; their text names no store content and no password.
var (
	ErrNoKeytool  = errors.New("no keytool found (set JAVA_HOME to a JDK, or put keytool on PATH)")
	ErrNoPassword = errors.New("no store password")
	ErrUnreadable = errors.New("keytool could not read the keystore with the password it was given")
)

// PassEnv carries the store password to keytool.
const PassEnv = "KEYTOOL_STOREPASS"

// Wait bounds one keytool run; a test may shorten it.
var Wait = 30 * time.Second

// Entry is one keystore entry: its alias (as keytool prints it), whether it holds a private key and its
// certificate (the first of the chain, the signing certificate), nil when keytool printed none that parses.
type Entry struct {
	Alias    string
	KeyEntry bool
	Cert     *x509.Certificate
}

// Find is the keytool of javaHome (when given and it has one), else the one on PATH.
func Find(javaHome string) (string, error) {
	if javaHome != "" {
		p := javaHome + "/bin/keytool"
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
			return p, nil
		}
	}
	p, err := exec.LookPath("keytool")
	if err != nil {
		return "", ErrNoKeytool
	}
	return p, nil
}

// List is the entries of the keystore at store, of the given type, opened with pass by the keytool binary bin.
func List(bin, store, kind, pass string) ([]Entry, error) {
	if pass == "" {
		return nil, ErrNoPassword
	}
	ctx, cancel := context.WithTimeout(context.Background(), Wait)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-list", "-rfc", "-keystore", store, "-storetype", kind, "-storepass:env", PassEnv)
	cmd.Env = append(os.Environ(), PassEnv+"="+pass, "LC_ALL=C", "LANG=C")
	var out bytes.Buffer
	cmd.Stdout = &out // cmd.Stderr stays nil: discarded
	if err := cmd.Run(); err != nil {
		return nil, ErrUnreadable
	}
	return parse(out.String()), nil
}

// parse reads the text of `keytool -list -rfc`: "Alias name: <a>", "Entry type: <t>", then PEM certificates.
func parse(text string) (entries []Entry) {
	var pemText strings.Builder
	in := false
	sc := bufio.NewScanner(strings.NewReader(text))
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		switch {
		case in:
			pemText.WriteString(line + "\n")
			if strings.HasPrefix(line, "-----END ") {
				in = false
				if block, _ := pem.Decode([]byte(pemText.String())); block != nil && len(entries) > 0 {
					if c, err := x509.ParseCertificate(block.Bytes); err == nil && entries[len(entries)-1].Cert == nil {
						entries[len(entries)-1].Cert = c
					}
				}
			}
		case strings.HasPrefix(line, "Alias name: "):
			entries = append(entries, Entry{Alias: strings.TrimPrefix(line, "Alias name: ")})
		case strings.HasPrefix(line, "Entry type: ") && len(entries) > 0:
			entries[len(entries)-1].KeyEntry = strings.HasSuffix(line, "PrivateKeyEntry")
		case strings.HasPrefix(line, "-----BEGIN CERTIFICATE-----"):
			in = true
			pemText.Reset()
			pemText.WriteString(line + "\n")
		}
	}
	return entries
}
