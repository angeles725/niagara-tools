package buildmodule

import (
	"bytes"
	"context"
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/modjar"
	"github.com/angeles725/niagara-tools/go/internal/signingprofile"
)

// logNoPrivateKey: the plugin found no key for the alias in the profile's keystore (#1121).
var logNoPrivateKey = regexp.MustCompile(`No private key for (\S+) found`)

// keytoolEntry is one entry line of `keytool -list`: "<alias>, <date>, <entry type>,". The date holds a comma
// of its own, and the alias is everything before it.
var keytoolEntry = regexp.MustCompile(`^(.+?), [^,]+, \d{4}, (?:PrivateKeyEntry|SecretKeyEntry|trustedCertEntry),`)

// keytoolWait bounds the keytool run; a test may shorten it.
var keytoolWait = 30 * time.Second

// storePassEnv carries the store password to keytool (-storepass:env): not on its command line, which any
// local user can read, and never in the output of this tool.
const storePassEnv = "BUILD_MODULE_STOREPASS"

// explainNoPrivateKey says what a "No private key for <alias> found" failure leaves unsaid: the signing
// profile the build used, the keystore it names and the aliases that store holds, so the alias can be
// corrected (a sibling project's alias with what looks like the same profile). Only alias names are printed:
// when keytool cannot list them without a password this tool does not have, the command to run is printed.
func (b *build) explainNoPrivateKey(alias string) {
	profile := b.signingProfile()
	if profile == "" {
		b.warn(msgNoKeyNoProfile, alias)
		return
	}
	b.warn(msgNoKeyIn, alias, profile)
	entries, err := readProfileEntries(profile)
	if err != nil {
		b.warn(msgNoKeyProfileUnreadable, err)
		return
	}
	store, kind := entries["niagara.signing.storepath"], entries["niagara.signing.storetype"]
	if store == "" {
		b.warn(msgNoKeyNoStore)
		return
	}
	if kind == "" {
		kind = "JCEKS"
	}
	b.warn(msgNoKeyStore, store, kind)
	aliases, why := b.listAliases(store, kind, entries["niagara.signing.storepass"])
	if len(aliases) > 0 {
		b.warn(msgNoKeyAliases, strings.Join(aliases, ", "))
		return
	}
	b.warn(msgNoKeyCommand, store, kind, why)
}

// signingProfile is the niagaraSigningProfile the build used, resolved as modjar.ConfiguredAlias resolves the
// alias, and made absolute against the gradle root as the root block's layout.projectDirectory.file does.
func (b *build) signingProfile() string {
	path := ""
	for _, p := range b.opt.gradleProps {
		if v, ok := strings.CutPrefix(p, "-P"+signingprofile.Property+"="); ok {
			path = v
		}
	}
	if path == "" {
		path = b.s.Env(gradleEnvPrefix + signingprofile.Property)
	}
	for _, dir := range []string{b.gradleUserHome(), b.gradleRoot} {
		if data, err := os.ReadFile(dir + "/" + modjar.GradleProperties); err == nil && path == "" {
			path, _ = signingprofile.GradleProperty(data, signingprofile.Property)
		}
	}
	if path != "" && !filepath.IsAbs(path) {
		path = filepath.Join(b.gradleRoot, path)
	}
	return path
}

// readProfileEntries reads a java.util.Properties XML file: <entry key="k">v</entry>.
func readProfileEntries(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Entries []struct {
			Key   string `xml:"key,attr"`
			Value string `xml:",chardata"`
		} `xml:"entry"`
	}
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}
	entries := map[string]string{}
	for _, e := range doc.Entries {
		entries[e.Key] = strings.TrimSpace(e.Value)
	}
	return entries, nil
}

// listAliases is the aliases of the keystore, read by `keytool -list` of the JDK 8 the build runs on (else the
// one on PATH) with the password given through the environment. why says what stopped it when there are none.
func (b *build) listAliases(store, kind, pass string) (aliases []string, why string) {
	if pass == "" {
		return nil, "the profile holds no store password"
	}
	keytool := b.java8 + "/bin/keytool"
	if _, err := os.Stat(keytool); err != nil {
		if keytool, err = exec.LookPath("keytool"); err != nil {
			return nil, "no keytool found"
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), keytoolWait)
	defer cancel()
	cmd := exec.CommandContext(ctx, keytool, "-list", "-keystore", store, "-storetype", kind, "-storepass:env", storePassEnv)
	cmd.Env = append(b.environ(), storePassEnv+"="+pass, "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		return nil, "keytool could not read the keystore with the profile's password"
	}
	for _, line := range strings.Split(string(out), "\n") {
		if m := keytoolEntry.FindStringSubmatch(line); m != nil {
			aliases = append(aliases, m[1])
		}
	}
	if len(aliases) == 0 {
		return nil, "the keystore lists no entries"
	}
	return aliases, ""
}
