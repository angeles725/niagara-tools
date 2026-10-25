// Package signingprofile is the one rule on a niagara signing profile that the kit enforces in two places:
// run-niagara-test refuses to run on it (go/internal/niagaratest) and lint-structure L16 FAILs a gradle root that
// declares it (go/internal/lint/rules/structure). Both read the profile named by the niagaraSigningProfile gradle
// property, parse it with Parse and judge it with Profile.Refusal.
//
// The niagara signing plugin's LocalSigningProfile (niagara-plugins 7.6.22, LocalSigningProfile.kt, read with
// javap -l) mints its self-signed key in generateCert and re-mints it in getCertInfo:
//
//   - :373-374 refuses validity <= refresh ("The validity period must be greater than the refresh period");
//   - :386-388 builds notAfter as Calendar.getInstance() then cal.set(Calendar.DAY_OF_MONTH, validity): it SETS
//     the day of the month to validity (lenient, so 365 rolls over), it does not add validity days. A key minted
//     on day d of a month therefore has validity - d days left (validity 30 on 25 Oct: notAfter 30 Oct; 365:
//     30 Sep next year);
//   - :216-225 computes the whole days left to notAfter and, when they are below refresh, deletes the chain and
//     mints a new key.
//
// So a fresh key is already due for refresh, and every signing re-mints it, whenever validity - d < refresh. The
// jars of such a build carry a signature block no JVM verifies ("cannot verify signature block file"), the NRE
// drops them and test.exe reports only "invalid test target(s)" (retro r10-scaffold-station-test Δ5). d reaches
// 31 and the day count is floored, so a profile is safe on every date only when validity - refresh >= 32. The
// vendor defect and its re-check on a plugin upgrade: retro r10-signing-margin-lint.
package signingprofile

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

const (
	// Property is the gradle property that names the profile file (templates/signing-root.gradle.kts).
	Property = "niagaraSigningProfile"
	// LocalType is the profile type that mints its own key; a profileType entry naming another type is not judged.
	LocalType = "LocalSigningProfile"
	// DefaultValidity and DefaultRefresh are the plugin's defaults (niagara.signing.validity, .refresh).
	DefaultValidity = 365
	DefaultRefresh  = 183
	// MinMargin is the least safe validity - refresh: 31 (the last day of a month) + 1 (the floored day count).
	MinMargin = 32
)

// Profile is what the rule reads from one profile file: whether it mints its own key and, when it does, the
// validity and refresh in days (the plugin defaults where an entry is absent).
type Profile struct {
	Local             bool
	Validity, Refresh int
}

// Parse reads a java.util.Properties XML profile (<properties><entry key="k">v</entry>…</properties>). A file
// without a niagara.signing.profileType entry is the plugin's LocalSigningProfile. A validity or refresh that is
// not a whole number the plugin's Int can hold is an error, as is a file that is not that XML; a profile of
// another type is returned without reading its days.
func Parse(b []byte) (Profile, error) {
	entries, err := entries(b)
	if err != nil {
		return Profile{}, fmt.Errorf("not a properties XML file: %w", err)
	}
	if t, ok := entries["niagara.signing.profileType"]; ok && !strings.HasSuffix(strings.TrimSpace(t), "."+LocalType) {
		return Profile{}, nil
	}
	p := Profile{Local: true, Validity: DefaultValidity, Refresh: DefaultRefresh}
	for _, e := range []struct {
		key string
		dst *int
	}{{"niagara.signing.validity", &p.Validity}, {"niagara.signing.refresh", &p.Refresh}} {
		v, ok := entries[e.key]
		if !ok {
			continue
		}
		// 32 bits: the plugin's Int, and validity - refresh can then never overflow.
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 32)
		if err != nil {
			return Profile{}, fmt.Errorf("%s=%q is not a whole number of days", e.key, v)
		}
		*e.dst = int(n)
	}
	return p, nil
}

// Safe is the predicate: a profile that mints no key, or one whose validity - refresh is at least MinMargin.
func (p Profile) Safe() bool { return !p.Local || p.Validity-p.Refresh >= MinMargin }

// Refusal is why the profile would re-mint its key at every signing, naming both values, or "" when it is Safe.
func (p Profile) Refusal() string {
	if p.Safe() {
		return ""
	}
	return fmt.Sprintf("niagara.signing.validity=%d, niagara.signing.refresh=%d: the signing plugin "+
		"sets a new key's notAfter to day %d of the current month, so the key has validity minus today's day of the month "+
		"left and is re-minted at every signing once that is below the refresh; the jars then carry a signature block no "+
		"JVM verifies and the NRE drops them. Keep validity - refresh >= %d (the plugin defaults are %d/%d)",
		p.Validity, p.Refresh, p.Validity, MinMargin, DefaultValidity, DefaultRefresh)
}

// GradleProperty reads one key of a java.util.Properties text file the way gradle does for the simple forms the
// kit writes: "key=value", "key:value" or "key value", # and ! comment lines, whitespace around the separator.
// Line continuations and escapes are not handled. The last definition wins, as in java.util.Properties.
func GradleProperty(data []byte, key string) (string, bool) {
	found, val := false, ""
	for _, raw := range bytes.Split(data, []byte("\n")) {
		ln := strings.TrimLeft(strings.TrimSuffix(string(raw), "\r"), " \t\f")
		if ln == "" || ln[0] == '#' || ln[0] == '!' {
			continue
		}
		i := strings.IndexAny(ln, "=: \t\f")
		if i < 0 {
			if ln == key {
				found, val = true, ""
			}
			continue
		}
		if ln[:i] != key {
			continue
		}
		rest := strings.TrimLeft(ln[i:], " \t\f")
		if rest != "" && (rest[0] == '=' || rest[0] == ':') {
			rest = rest[1:]
		}
		found, val = true, strings.TrimSpace(rest)
	}
	return val, found
}

// entries parses the java.util.Properties XML form into its key/value map.
func entries(b []byte) (map[string]string, error) {
	var doc struct {
		Entries []struct {
			Key   string `xml:"key,attr"`
			Value string `xml:",chardata"`
		} `xml:"entry"`
	}
	d := xml.NewDecoder(bytes.NewReader(b))
	d.Strict = false // the file carries a DOCTYPE that is never fetched
	if err := d.Decode(&doc); err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, e := range doc.Entries {
		out[e.Key] = e.Value
	}
	return out, nil
}
