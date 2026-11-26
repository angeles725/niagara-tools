package signinginventory

import (
	"crypto/x509"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/keytool"
	"github.com/angeles725/niagara-tools/go/internal/signingdoctor"
	"github.com/angeles725/niagara-tools/go/internal/signingprofile"
)

// warnDays: a certificate with fewer days left than this is a WARN.
const warnDays = 60

// alias is a signing alias found in a listed store.
type alias struct {
	Name, Store string
	Expired     bool
}

// profile lists the keystore one profile names. The profile's own store password opens the store through
// the keytool environment; it is never printed, so a store that cannot be listed is one WARN row that names
// its path and a fixed reason, nothing keytool or the file said.
func (r *run) profile(path string) {
	data, err := readProfile(path)
	if errors.Is(err, errNotProfileFile) {
		r.add("WARN", "profile", "%s: %v", path, err)
		return
	}
	if err != nil {
		r.add("WARN", "profile", "%s: unreadable", path)
		return
	}
	entries, err := signingprofile.Entries(data)
	if err != nil {
		r.add("WARN", "profile", "%s: %v", path, err)
		return
	}
	r.add("INFO", "profile", "%s", path)
	store, kind := entries[signingprofile.StorePathKey], entries[signingprofile.StoreTypeKey]
	if store == "" {
		r.add("WARN", "store", "%s: the profile names no keystore", path)
		return
	}
	if !filepath.IsAbs(store) {
		store = filepath.Join(filepath.Dir(path), store)
	}
	if kind == "" {
		kind = signingprofile.DefaultStoreType
	}
	r.add("INFO", "store", "%s (%s)", store, kind)
	if st, err := os.Stat(store); err != nil || !st.Mode().IsRegular() {
		r.locked++
		r.add("WARN", "store-unreadable", "%s: no such file", store)
		return
	}
	pass := entries[signingprofile.StorePassKey]
	bin, err := keytool.Find(r.javaHome)
	if pass != "" && err != nil {
		r.locked++
		r.add("WARN", "store-unreadable", "%s: %v", store, err)
		return
	}
	list, err := keytool.List(bin, store, kind, pass)
	switch {
	case errors.Is(err, keytool.ErrNoPassword):
		r.locked++
		r.add("WARN", "store-locked", "%s: the profile holds no store password", store)
	case err != nil:
		r.locked++
		r.add("WARN", "store-unreadable", "%s: %v", store, err)
	default:
		r.listed++
		for _, e := range list {
			r.entry(store, e)
		}
	}
}

// entry adds the rows of one keystore entry. The code-signing, CA and validity rules are signingdoctor's
// (CheckX509); a trusted certificate (a CA a key chains to) is listed but not judged as a signing key.
func (r *run) entry(store string, e keytool.Entry) {
	if e.Cert == nil {
		r.add("WARN", "alias", "%s: no certificate in %s", e.Alias, store)
		return
	}
	c := e.Cert
	kind := "trusted certificate"
	if e.KeyEntry {
		kind = "signing key"
	}
	r.add("INFO", "alias", "%s  %s in %s  subject %s  issuer %s", e.Alias, kind, store, name(c.Subject.CommonName, c.Subject.String()),
		name(c.Issuer.CommonName, c.Issuer.String()))
	a := alias{Name: e.Alias, Store: store, Expired: r.now.After(c.NotAfter)}
	if e.KeyEntry {
		r.found = append(r.found, a)
	}
	r.rows = append(r.rows, r.validity(e.Alias, c))
	if !e.KeyEntry {
		return
	}
	for _, row := range signingdoctor.CheckX509(c, r.now) {
		if row.Check == "cert-ca" || row.Check == "cert-usage" {
			r.add(row.Status, strings.TrimPrefix(row.Check, "cert-"), "%s: %s", e.Alias, row.Detail)
		}
	}
}

// validity is the status, check id and detail of the validity row: FAIL expired or not yet valid, WARN
// under warnDays left, else PASS.
func (r *run) validity(alias string, c *x509.Certificate) signingdoctor.Row {
	day := c.NotAfter.UTC().Format("2006-01-02")
	left := int(c.NotAfter.Sub(r.now).Hours() / 24)
	switch {
	case r.now.Before(c.NotBefore):
		return row("FAIL", alias+": not valid before "+c.NotBefore.UTC().Format("2006-01-02"))
	case r.now.After(c.NotAfter):
		return row("FAIL", alias+": expired "+day+" ("+days(int(r.now.Sub(c.NotAfter).Hours()/24))+" ago)")
	case left < warnDays:
		return row("WARN", alias+": valid until "+day+" ("+days(left)+" left)")
	}
	return row("PASS", alias+": valid until "+day+" ("+days(left)+" left)")
}

func row(status, detail string) signingdoctor.Row {
	return signingdoctor.Row{Status: status, Check: "validity", Detail: detail}
}

func days(n int) string {
	if n == 1 {
		return "1 day"
	}
	return strconv.Itoa(n) + " days"
}

// name is the common name, else the whole distinguished name.
func name(cn, full string) string {
	if cn != "" {
		return cn
	}
	return full
}
