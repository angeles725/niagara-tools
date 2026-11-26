package buildmodule

import (
	"os"
	"strings"
	"testing"
)

const storePass = "s3cret-pass"

// keytoolStub is a keytool in the JDK 8 directory: it logs its arguments (the password is never one), fails
// unless the password reached it through the environment, and lists two entries.
const keytoolStub = `#!/usr/bin/env bash
echo "keytool $*" >> "LOGDIR/keytool.log"
case " $* " in *" -storepass:env "*) ;; *) exit 2 ;; esac
[ "$BUILD_MODULE_STOREPASS" = "` + storePass + `" ] || { echo "keytool error: Keystore was tampered with, or password was incorrect"; exit 1; }
printf 'Keystore type: JCEKS\nYour keystore contains 2 entries\n\n'
printf 'acme, Nov 25, 2026, PrivateKeyEntry, \nCertificate fingerprint (SHA1): AA:BB\n'
printf 'second signer, Nov 25, 2026, PrivateKeyEntry, \nCertificate fingerprint (SHA1): CC:DD\n'
`

// noKeyBuild runs a failing gradle ("No private key for Wrong found") with profile as niagaraSigningProfile
// (not written when "") and, when stub is not "", a keytool in the JDK directory.
func noKeyBuild(t *testing.T, profile, stub string, extra ...string) *fixture {
	t.Helper()
	f := newFixture(t)
	f.signs()
	f.does("gradlew", `echo "No private key for Wrong found"; exit 1`)
	if stub != "" {
		f.write(f.dir+"/j8/bin/keytool", strings.ReplaceAll(stub, "LOGDIR", f.dir))
		if err := os.Chmod(f.dir+"/j8/bin/keytool", 0o755); err != nil {
			t.Fatal(err)
		}
	}
	args := []string{"--no-drift-check", "--no-report", "--gradle-prop", "niagaraSigningAlias=Wrong"}
	if profile != "" {
		args = append(args, "--gradle-prop", "niagaraSigningProfile="+f.dir+"/prof.xml")
		f.write(f.dir+"/prof.xml", profile)
	}
	f.wants("no private key", f.build(append(args, extra...)...), ExitGradle, "build.sh: gradle failed")
	return f
}

func profileXML(entries string) string {
	return `<?xml version="1.0"?><!DOCTYPE properties SYSTEM "http://java.sun.com/dtd/properties.dtd"><properties>` +
		`<entry key="niagara.signing.storetype">JCEKS</entry>` + entries + `</properties>`
}

// The failure names the profile and the keystore, lists the aliases the keystore holds and never prints the
// password, which keytool gets through its environment, not its command line (#1121).
func TestNoPrivateKeyListsTheAliasesOfTheProfileKeystore(t *testing.T) {
	f := noKeyBuild(t, profileXML(`<entry key="niagara.signing.storepath">/keys/a.jceks</entry>`+
		`<entry key="niagara.signing.storepass">`+storePass+`</entry>`), keytoolStub)
	for _, want := range []string{
		"build.sh: no private key for the alias Wrong in the signing profile " + f.dir + "/prof.xml",
		"keystore /keys/a.jceks", "holds the aliases: acme, second signer",
		"--gradle-prop niagaraSigningAlias=<one of them>",
	} {
		if !strings.Contains(f.err, want) {
			t.Errorf("stderr lacks %q:\n%s", want, f.err)
		}
	}
	if got := f.out + f.err + f.read(f.dir+"/keytool.log"); strings.Contains(got, storePass) {
		t.Errorf("the password is in the output or on the keytool command line:\n%s", got)
	}
	if got := f.read(f.dir + "/keytool.log"); !strings.Contains(got, "-list -keystore /keys/a.jceks -storetype JCEKS") {
		t.Errorf("keytool log %q, want a -list of the profile's store", got)
	}
}

// A keytool that cannot list (here: a wrong password) or a profile without the password leaves the command
// the user can run; the password is not printed there either.
func TestNoPrivateKeyFallsBackToTheCommand(t *testing.T) {
	wrong := profileXML(`<entry key="niagara.signing.storepath">/keys/a.jceks</entry><entry key="niagara.signing.storepass">nope</entry>`)
	none := profileXML(`<entry key="niagara.signing.storepath">/keys/a.jceks</entry>`)
	for _, c := range []struct{ name, profile string }{{"wrong password", wrong}, {"no password in the profile", none}} {
		f := noKeyBuild(t, c.profile, keytoolStub)
		for _, want := range []string{"signing profile " + f.dir + "/prof.xml", "keystore /keys/a.jceks",
			"could not list its aliases", "keytool -list -keystore /keys/a.jceks -storetype JCEKS"} {
			if !strings.Contains(f.err, want) {
				t.Errorf("%s: stderr lacks %q:\n%s", c.name, want, f.err)
			}
		}
		if strings.Contains(f.err, "holds the aliases") || strings.Contains(f.out+f.err, "nope") {
			t.Errorf("%s: stderr claims aliases or prints a password:\n%s", c.name, f.err)
		}
	}
}

// No profile named, a profile that is not there, a profile that names no keystore: each says what it is.
func TestNoPrivateKeyWithoutAUsableProfile(t *testing.T) {
	f := noKeyBuild(t, "", keytoolStub)
	if !strings.Contains(f.err, "no niagaraSigningProfile is configured") {
		t.Errorf("no profile: stderr %q", f.err)
	}
	f = newFixture(t)
	f.signs()
	f.does("gradlew", `echo "No private key for Wrong found"; exit 1`)
	f.wants("missing profile", f.build("--no-drift-check", "--no-report", "--gradle-prop", "niagaraSigningProfile=/nope/p.xml"), ExitGradle,
		"signing profile /nope/p.xml", "cannot be read")
	f = noKeyBuild(t, profileXML(""), keytoolStub)
	if !strings.Contains(f.err, "names no niagara.signing.storepath") {
		t.Errorf("no store: stderr %q", f.err)
	}
}

// Another gradle failure is explained as before: no keystore text.
func TestOtherGradleFailuresStayAsTheyWere(t *testing.T) {
	f := newFixture(t)
	f.does("gradlew", "exit 1")
	f.wants("plain", f.build("--no-drift-check", "--no-report"), ExitGradle, "build.sh: gradle failed")
	if strings.Contains(f.err, "keystore") {
		t.Errorf("stderr %q", f.err)
	}
}
