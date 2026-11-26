package signinginventory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUsageErrorsExit2AndPrintNothingOnStdout(t *testing.T) {
	for name, args := range map[string][]string{
		"unknown option":             {"--nope"},
		"a stray argument":           {"extra"},
		"module-name without module": {"--module-name", "X"},
		"build-state without module": {"--build-state", "x"},
		"empty profile":              {"--profile", ""},
		"empty home":                 {"--home", ""},
		"empty module":               {"--module", ""},
		"missing profile file":       {"--profile", "/nonexistent/p.xml"},
		"missing home":               {"--home", "/nonexistent/home"},
		"missing module root":        {"--module", "/nonexistent/mod"},
	} {
		code, out, errs := inv(t, nil, args...)
		if code != 2 || out != "" || errs == "" {
			t.Errorf("%s: exit %d, stdout %q, stderr %q", name, code, out, errs)
		}
	}
	_, _, errs := inv(t, nil, "--nope")
	if !strings.Contains(errs, "usage: signing-inventory") || !strings.Contains(errs, "Places searched") {
		t.Errorf("usage text missing the searched places:\n%s", errs)
	}
}

func TestListsEveryAliasWithItsCertificateAndTheFileItLivesIn(t *testing.T) {
	need(t)
	prof := write(t, t.TempDir(), "p.xml", profileXML(fx.jceks, "JCEKS", canaryStore))
	code, out, errs := inv(t, nil, "--profile", prof)
	if code != 1 || errs != "" {
		t.Fatalf("exit %d (want 1: an expired key and a CA), stderr %q\n%s", code, errs, out)
	}
	has(t, out, "INFO", "profile", prof)
	has(t, out, "INFO", "store", fx.jceks, "(JCEKS)")
	has(t, out, "INFO", "alias", "good", "signing key", "in "+fx.jceks, "subject good signer", "issuer good signer")
	has(t, out, "PASS", "validity", "good: valid until", "days left")
	has(t, out, "PASS", "usage", "good: code signing")
	has(t, out, "PASS", "ca", "good: not a CA")
	has(t, out, "WARN", "validity", "soon: valid until", "days left")
	has(t, out, "FAIL", "validity", "old: expired 2020-01-3", "ago")
	has(t, out, "FAIL", "ca", "ca: the certificate is a CA")
	quiet(t, out, errs)
}

func TestCleanProfileExits0(t *testing.T) {
	need(t)
	prof := write(t, t.TempDir(), "p.xml", profileXML(fx.p12, "PKCS12", canaryStore))
	code, out, errs := inv(t, nil, "--profile", prof)
	if code != 0 || errs != "" || strings.Contains(out, "FAIL") {
		t.Fatalf("exit %d, stderr %q\n%s", code, errs, out)
	}
	has(t, out, "INFO", "store", fx.p12, "(PKCS12)")
	has(t, out, "PASS", "validity", "good: valid until")
}

func TestAStoreItCannotOpenIsOneWarnRowWithItsPathOnly(t *testing.T) {
	need(t)
	dir := t.TempDir()
	for name, c := range map[string]struct{ prof, check, why string }{
		"no store password":  {profileXML(fx.jceks, "JCEKS", ""), "store-locked", "the profile holds no store password"},
		"wrong password":     {profileXML(fx.jceks, "JCEKS", "wrong-Canary-1"), "store-unreadable", "could not read the keystore"},
		"wrong store type":   {profileXML(fx.jceks, "PKCS12", canaryStore), "store-unreadable", "could not read the keystore"},
		"store file missing": {profileXML(dir+"/none.jceks", "JCEKS", canaryStore), "store-unreadable", "no such file"},
	} {
		code, out, errs := inv(t, nil, "--profile", write(t, dir, strings.ReplaceAll(name, " ", "-")+".xml", c.prof))
		if code != 0 || errs != "" {
			t.Errorf("%s: exit %d (a locked store is a WARN), stderr %q", name, code, errs)
		}
		has(t, out, "WARN", c.check, c.why)
		if strings.Contains(out, "alias") || strings.Contains(out, "wrong-Canary-1") {
			t.Errorf("%s: the store's content or a password was printed:\n%s", name, out)
		}
		quiet(t, out, errs)
	}
}

func TestProfileThatIsNotXMLNeverQuotesItsText(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"bare text":     canaryStore + "<",
		"bad close tag": "<properties><entry key=\"niagara.signing.storepass\">pa</" + canaryStore + "</entry></properties>",
	} {
		code, out, errs := inv(t, nil, "--profile", write(t, dir, strings.ReplaceAll(name, " ", "-")+".xml", body))
		if code != 0 {
			t.Errorf("%s: exit %d, want 0 (a WARN row)", name, code)
		}
		has(t, out, "WARN", "profile", "not valid properties XML")
		quiet(t, out, errs)
	}
}

func TestSearchedPlaces(t *testing.T) {
	need(t)
	home := t.TempDir()
	write(t, home, ".tridium/security/niagara.signing.xml", profileXML(fx.p12, "PKCS12", canaryStore))
	write(t, home, ".config/n4-signing/acme.signing.xml", profileXML(fx.p12, "PKCS12", canaryStore))
	write(t, home, ".config/n4-signing/notes.xml", "<properties/>")
	niagara := t.TempDir()
	jar(t, filepath.Join(niagara, "modules/baja.jar"),
		map[string]string{"META-INF/module.xml": `<module name="baja" vendor="Tridium" vendorVersion="4.14.0.162"/>`})
	write(t, niagara, "etc/lab.signing.xml", profileXML(fx.p12, "PKCS12", canaryStore))
	other := write(t, t.TempDir(), "given.xml", profileXML(fx.p12, "PKCS12", canaryStore))

	_, out, _ := inv(t, map[string]string{"HOME": home})
	has(t, out, "profile", filepath.Join(home, ".tridium/security/niagara.signing.xml"))
	has(t, out, "profile", filepath.Join(home, ".config/n4-signing/acme.signing.xml"))
	if strings.Contains(out, "notes.xml") || strings.Contains(out, "install") {
		t.Errorf("only *.signing.xml of the user places, no install:\n%s", out)
	}
	_, out, _ = inv(t, map[string]string{"HOME": home}, "--home", niagara)
	has(t, out, "INFO", "install", niagara, "Niagara 4.14.0.162")
	has(t, out, "profile", filepath.Join(niagara, "etc/lab.signing.xml"))
	_, out, _ = inv(t, map[string]string{"HOME": home}, "--profile", other, "--profile", other)
	if strings.Count(out, "INFO  profile") != 1 || strings.Contains(out, ".tridium") {
		t.Errorf("--profile lists just the given file, once:\n%s", out)
	}
	_, out, _ = inv(t, nil, "--home", t.TempDir())
	has(t, out, "WARN", "install", "not a Niagara installation")
	has(t, out, "INFO", "profiles", "no signing profile found")
}

// A keytool that prints the store password on both streams and fails: nothing of it may surface, and the
// password reaches it only through its environment, never its command line.
func TestKeytoolOutputAndArgvNeverReachTheOutput(t *testing.T) {
	java := t.TempDir()
	log := filepath.Join(java, "argv.log")
	write(t, java, "bin/keytool", "#!/bin/sh\necho \"$*\" >> '"+log+"'\n"+
		"echo \"bad: $KEYTOOL_STOREPASS\" >&2\necho \"bad: $KEYTOOL_STOREPASS\"\nexit 1\n")
	if err := os.Chmod(filepath.Join(java, "bin/keytool"), 0o755); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	store := write(t, dir, "s.jceks", "x")
	prof := write(t, dir, "p.xml", profileXML(store, "JCEKS", canaryStore))
	code, out, errs := inv(t, map[string]string{"JAVA_HOME": java}, "--profile", prof)
	if code != 0 {
		t.Errorf("exit %d\n%s%s", code, out, errs)
	}
	has(t, out, "WARN", "store-unreadable", store)
	quiet(t, out, errs)
	argv, err := os.ReadFile(log)
	if err != nil || !strings.Contains(string(argv), "-storepass:env") {
		t.Errorf("keytool not run with -storepass:env: %v %q", err, argv)
	}
	quiet(t, string(argv))
}

func TestNoKeytoolIsAWarnRow(t *testing.T) {
	dir := t.TempDir()
	prof := write(t, dir, "p.xml", profileXML(write(t, dir, "s.jceks", "x"), "JCEKS", canaryStore))
	t.Setenv("PATH", "")
	code, out, _ := inv(t, map[string]string{"JAVA_HOME": dir}, "--profile", prof)
	if code != 0 {
		t.Errorf("exit %d", code)
	}
	has(t, out, "WARN", "store-unreadable", "s.jceks", "no keytool found")
}
