package niagaratest

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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
// 31 and the day count is floored, so a profile is safe on every date only when validity - refresh >= 32.
const (
	signingProfileProp  = "niagaraSigningProfile"
	localProfileType    = "LocalSigningProfile"
	defaultValidityDays = 365 // the plugin's default profile (niagara.signing.validity)
	defaultRefreshDays  = 183 // the plugin's default profile (niagara.signing.refresh)
	minValidityMargin   = 32  // 31 (the last day of a month) + 1 (the floored day count)
)

// signingProfileRefusal returns why the build's signing profile would re-mint its key at every signing, or "" when
// it would not (no profile property, a profile type that never mints a key, or a safe validity/refresh). The
// profile is the niagaraSigningProfile gradle property, from GRADLE_USER_HOME/gradle.properties first (gradle's
// precedence) and then the gradle root's; a relative path resolves against the gradle root, as the kit's root
// signing block does (layout.projectDirectory.file). A property passed with -P is not seen: the runner passes none.
func signingProfileRefusal(gradleRoot, gradleUserHome string) string {
	path := ""
	for _, f := range []string{filepath.Join(gradleUserHome, "gradle.properties"), filepath.Join(gradleRoot, "gradle.properties")} {
		if v, ok := gradleProperty(f, signingProfileProp); ok {
			path = v
			break
		}
	}
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(gradleRoot, path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("signing profile %s (%s): cannot read it: %v", path, signingProfileProp, err)
	}
	entries, err := profileEntries(b)
	if err != nil {
		return fmt.Sprintf("signing profile %s: not a properties XML file: %v", path, err)
	}
	if t, ok := entries["niagara.signing.profileType"]; ok && !strings.HasSuffix(strings.TrimSpace(t), "."+localProfileType) {
		return ""
	}
	validity, refresh := defaultValidityDays, defaultRefreshDays
	for _, e := range []struct {
		key string
		dst *int
	}{{"niagara.signing.validity", &validity}, {"niagara.signing.refresh", &refresh}} {
		v, ok := entries[e.key]
		if !ok {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Sprintf("signing profile %s: %s=%q is not a whole number of days", path, e.key, v)
		}
		*e.dst = n
	}
	if validity-refresh >= minValidityMargin {
		return ""
	}
	return fmt.Sprintf("signing profile %s: niagara.signing.validity=%d, niagara.signing.refresh=%d: the signing plugin "+
		"sets a new key's notAfter to day %d of the current month, so the key has validity minus today's day of the month "+
		"left and is re-minted at every signing once that is below the refresh; the jars then carry a signature block no "+
		"JVM verifies and the NRE drops them. Keep validity - refresh >= %d (the plugin defaults are %d/%d)",
		path, validity, refresh, validity, minValidityMargin, defaultValidityDays, defaultRefreshDays)
}

// gradleProperty reads one key of a java.util.Properties text file the way gradle does for the simple forms the
// kit writes: "key=value", "key:value" or "key value", # and ! comment lines, whitespace around the separator.
// Line continuations and escapes are not handled. A missing file has no property.
func gradleProperty(file, key string) (string, bool) {
	f, err := os.Open(file)
	if err != nil {
		return "", false
	}
	defer f.Close()
	found, val := false, ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		ln := strings.TrimLeft(sc.Text(), " \t\f")
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
		found, val = true, strings.TrimSpace(rest) // the last definition wins, as in java.util.Properties
	}
	return val, found
}

// profileEntries parses the java.util.Properties XML form (<properties><entry key="k">v</entry>…</properties>).
func profileEntries(b []byte) (map[string]string, error) {
	var doc struct {
		Entries []struct {
			Key   string `xml:"key,attr"`
			Value string `xml:",chardata"`
		} `xml:"entry"`
	}
	d := xml.NewDecoder(strings.NewReader(string(b)))
	d.Strict = false // the file carries a DOCTYPE that is never fetched
	if err := d.Decode(&doc); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(doc.Entries))
	for _, e := range doc.Entries {
		out[e.Key] = e.Value
	}
	return out, nil
}
