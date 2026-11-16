package ngdeploy

import (
	"strings"
	"testing"
)

// builtAt is the entries of a module jar as two builds of the same sources at different times produce it:
// the build time in module.xml, the manifest's per-entry digests and the signature files differ, and
// nothing else. change, when given, replaces or adds entries (the content gone removes one).
// gone, as the content of a changed entry, removes it.
const gone = "\x00gone"

func builtAt(millis string, change map[string]string) map[string]string {
	entries := map[string]string{
		"META-INF/": "",
		"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\r\nImplementation-Vendor: Acme\r\nSealed: true\r\n\r\n" +
			"Name: com/acme/BDemo.class\r\nSHA-256-Digest: aaaa\r\n\r\n" +
			"Name: META-INF/module.xml\r\nSHA-256-Digest: digest-of-" + millis + "-that-is-long-enough-to-be-folded-on-the-next\r\n line\r\n\r\n",
		"META-INF/ACMESIGN.SF":  "Signature-Version: 1.0\r\nSHA-256-Digest-Manifest: " + millis + "\r\n",
		"META-INF/ACMESIGN.RSA": "block " + millis,
		"META-INF/module.xml": `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
			`<module name="demo-rt" vendor="Acme" vendorVersion="1.0" buildMillis="` + millis + `" buildHost="HOST" moduleName="demo">` + "\n <types/>\n</module>\n",
		"com/acme/BDemo.class": "class bytes",
		"module.palette":       "palette",
	}
	for name, content := range change {
		if entries[name] = content; content == gone {
			delete(entries, name)
		}
	}
	return entries
}

// sameVersion deploys a built rt jar over a deployed one, both at vendorVersion 1.0.
func sameVersion(t *testing.T, deployed, built map[string]string) (f *fixture, code int) {
	t.Helper()
	f = newFixture(t)
	f.zipJar(f.deployedJar("rt"), deployed)
	f.zipJar(f.builtJar("rt"), built)
	t.Setenv("FAKE_OLD_VERSION", "1.0")
	t.Setenv("FAKE_NEW_VERSION", "1.0")
	return f, f.run("--no-backup", "--mode", "C")
}

// A rebuild of unchanged sources is not a version to bump: the jars differ only in the build time of
// module.xml, the manifest's per-entry digests and the signature files, so the deploy goes through with
// the same vendorVersion, also when the rebuild is signed by another alias or not signed at all.
func TestAnUnchangedRebuildPassesTheVersionPrecheck(t *testing.T) {
	for name, built := range map[string]map[string]string{
		"a later build":                 builtAt("1788609309999", nil),
		"signed by another alias":       builtAt("1788609309999", map[string]string{"META-INF/ACMESIGN.SF": gone, "META-INF/ACMESIGN.RSA": gone, "META-INF/OTHER.SF": "sf", "META-INF/OTHER.EC": "ec"}),
		"signature files in lower case": builtAt("1788609309999", map[string]string{"META-INF/ACMESIGN.SF": gone, "META-INF/ACMESIGN.RSA": gone, "META-INF/acmesign.sf": "sf", "META-INF/acmesign.dsa": "dsa"}),
	} {
		f, code := sameVersion(t, builtAt("1788609301540", nil), built)
		f.want(code, ExitOK, "[ng-deploy] version-bump ok (demo-rt.jar: vendorVersion still 1.0, and the same content as the deployed jar apart from the build time and host, the manifest digests and the signature files)\n", "[ng-deploy] copy ok\n")
		if f.err != warnNoBackup || f.read(f.deployedJar("rt")) != f.read(f.builtJar("rt")) {
			t.Errorf("%s: stderr %q", name, f.err)
		}
	}
}

// Anything else that differs under the same vendorVersion is still exit 51 with the line it always had: a
// changed class, an entry added or removed, and a difference inside the three files the comparison reads
// in part (module.xml outside buildMillis, the manifest's main section, a per-entry attribute that is not a
// digest, a signature-like file outside META-INF/ or below a directory of it).
func TestChangedContentUnderTheSameVersionIsStillExit51(t *testing.T) {
	later := "1788609309999"
	for name, change := range map[string]map[string]string{
		"one changed class":                          {"com/acme/BDemo.class": "other bytes"},
		"an added entry":                             {"com/acme/BNew.class": "new"},
		"a removed entry":                            {"module.palette": gone},
		"an added directory":                         {"com/acme/extra/": ""},
		"module.xml elsewhere":                       {"META-INF/module.xml": strings.ReplaceAll(builtAt(later, nil)["META-INF/module.xml"], `vendor="Acme"`, `vendor="Other"`)},
		"the manifest's main section":                {"META-INF/MANIFEST.MF": strings.ReplaceAll(builtAt(later, nil)["META-INF/MANIFEST.MF"], "Sealed: true", "Sealed: false")},
		"a per-entry attribute that is not a digest": {"META-INF/MANIFEST.MF": strings.ReplaceAll(builtAt(later, nil)["META-INF/MANIFEST.MF"], "Name: com/acme/BDemo.class\r\n", "Name: com/acme/BDemo.class\r\nContent-Type: other\r\n")},
		"a .SF file outside META-INF":                {"rc/ACMESIGN.SF": "not a signature"},
		"a .SF file below a directory of META-INF":   {"META-INF/services/X.SF": "not a signature"},
	} {
		f, code := sameVersion(t, builtAt("1788609301540", nil), builtAt(later, change))
		want := warnNoBackup + "[ng-deploy] ERROR version-bump precheck: demo-rt.jar bytes changed but vendorVersion is still 1.0 — bump defaultModuleVersion and rebuild\n"
		if code != ExitVersionSame || f.err != want || f.read(f.deployedJar("rt")) == f.read(f.builtJar("rt")) || strings.Contains(f.calls(), "vm.sh") {
			t.Errorf("%s: exit %d, stderr %q", name, code, f.err)
		}
	}
}

// A jar whose content cannot be read is never "the same content": with the same vendorVersion the precheck
// cannot verify (exit 52) and says which jar and why, whichever of the two it is.
func TestAJarThatCannotBeReadIsNotTheSameContent(t *testing.T) {
	good := builtAt("1788609301540", nil)
	for name, tc := range map[string]struct {
		arrange func(f *fixture)
		why     string // with JAR for the jar the message names
	}{
		"the deployed jar is not a zip": {func(f *fixture) { f.write(f.deployedJar("rt"), "not a zip") }, "DEPLOYED is not a readable jar ("},
		"the built jar is not a zip":    {func(f *fixture) { f.write(f.builtJar("rt"), "not a zip either") }, "BUILT is not a readable jar ("},
		"a jar with no entry":           {func(f *fixture) { f.zipJar(f.builtJar("rt"), nil) }, "BUILT holds no entry"},
	} {
		f := newFixture(t)
		f.zipJar(f.deployedJar("rt"), good)
		f.zipJar(f.builtJar("rt"), builtAt("1788609309999", nil))
		tc.arrange(f)
		t.Setenv("FAKE_OLD_VERSION", "1.0")
		t.Setenv("FAKE_NEW_VERSION", "1.0")
		before := f.read(f.deployedJar("rt"))
		code := f.run("--no-backup", "--mode", "C")
		want := warnNoBackup + "[ng-deploy] ERROR version-bump precheck cannot verify demo-rt.jar: cannot compare its content with the deployed jar: " +
			strings.NewReplacer("DEPLOYED", f.deployedJar("rt"), "BUILT", f.builtJar("rt")).Replace(tc.why)
		if code != ExitVersionUnknown || !strings.HasPrefix(f.err, want) || strings.Count(f.err, "\n") != 2 || f.read(f.deployedJar("rt")) != before || strings.Contains(f.out, "copy ok") {
			t.Errorf("%s: exit %d, stderr %q", name, code, f.err)
		}
	}
}

// descriptor is a module.xml with a root element of the given attributes and a body.
func descriptor(root, body string) map[string]string {
	return map[string]string{"META-INF/module.xml": `<?xml version="1.0" encoding="UTF-8"?>` + "\n<module " + root + ">\n" + body + "</module>\n"}
}

// The build time and the build host are the two things of module.xml a rebuild of unchanged sources
// changes: both are left out wherever they stand among the attributes of the root element . The same names on another element, and any other attribute or byte, are content (exit 51).
func TestTheBuildTimeAndHostOfTheDescriptorAreLeftOutWhereverTheyStand(t *testing.T) {
	deployed := builtAt("1", descriptor(`name="demo-rt" buildMillis="1" buildHost="ONE" vendorVersion="1.0"`, ` <dependency name="x>y" buildMillis="7"/>`+"\n"))
	for name, tc := range map[string]struct {
		root, body string
		code       int
	}{
		"built on another machine":             {`name="demo-rt" buildMillis="2" buildHost="TWO" vendorVersion="1.0"`, ` <dependency name="x>y" buildMillis="7"/>` + "\n", ExitOK},
		"the same attributes in another order": {`buildMillis="2" buildHost="TWO" name="demo-rt" vendorVersion="1.0"`, ` <dependency name="x>y" buildMillis="7"/>` + "\n", ExitVersionSame},
		"buildMillis of a nested element":      {`name="demo-rt" buildMillis="2" buildHost="TWO" vendorVersion="1.0"`, ` <dependency name="x>y" buildMillis="8"/>` + "\n", ExitVersionSame},
		"another attribute of the root":        {`name="demo-rt" buildMillis="2" buildHost="TWO" vendorVersion="1.0" autoload="true"`, ` <dependency name="x>y" buildMillis="7"/>` + "\n", ExitVersionSame},
		"a byte after the root element's tag":  {`name="demo-rt" buildMillis="2" buildHost="TWO" vendorVersion="1.0"`, ` <dependency name="x>y" buildMillis="7"/> ` + "\n", ExitVersionSame},
	} {
		f, code := sameVersion(t, deployed, builtAt("2", descriptor(tc.root, tc.body)))
		if code != tc.code || (code == ExitOK) != strings.Contains(f.out, "copy ok") {
			t.Errorf("%s: exit %d, want %d, stderr %q", name, code, tc.code, f.err)
		}
	}
	first := builtAt("1", descriptor(`buildMillis="1" name="demo-rt" buildHost="ONE"`, ""))
	if f, code := sameVersion(t, first, builtAt("2", descriptor(`buildMillis="2" name="demo-rt" buildHost="TWO"`, ""))); code != ExitOK {
		t.Errorf("buildMillis as the first attribute: exit %d, stderr %q", code, f.err)
	}
}

// The precheck reads a jar in pieces and refuses to read without end: a jar that declares more content than
// the limit, a descriptor or a manifest over theirs, and a descriptor with no root element cannot be
// verified (exit 52); none is read whole into memory, and none passes.
func TestAJarOverTheLimitsOfThePrecheckCannotBeVerified(t *testing.T) {
	good := builtAt("1788609301540", nil)
	for name, tc := range map[string]struct {
		arrange func(f *fixture)
		why     string
	}{
		"more declared content than the limit": {func(f *fixture) { f.declares(f.builtJar("rt"), "big.bin", 1<<31+1) }, "BUILT declares more than 2147483648 bytes of content, the most the precheck reads"},
		"a descriptor over its limit": {func(f *fixture) {
			f.zipJar(f.builtJar("rt"), builtAt("2", descriptor(`name="demo-rt"`, strings.Repeat(" ", 1<<23))))
		}, "BUILT: META-INF/module.xml is over 8388608 bytes"},
		"a manifest over its limit": {func(f *fixture) {
			f.zipJar(f.builtJar("rt"), builtAt("2", map[string]string{"META-INF/MANIFEST.MF": strings.Repeat("x", 1<<23+1)}))
		}, "BUILT: META-INF/MANIFEST.MF is over 8388608 bytes"},
		"a descriptor with no root element": {func(f *fixture) {
			f.zipJar(f.builtJar("rt"), builtAt("2", map[string]string{"META-INF/module.xml": "not xml <"}))
		}, "BUILT: META-INF/module.xml cannot be read as a descriptor (no root element: "},
	} {
		f := newFixture(t)
		f.zipJar(f.deployedJar("rt"), good)
		tc.arrange(f)
		t.Setenv("FAKE_OLD_VERSION", "1.0")
		t.Setenv("FAKE_NEW_VERSION", "1.0")
		code := f.run("--no-backup", "--mode", "C")
		want := warnNoBackup + "[ng-deploy] ERROR version-bump precheck cannot verify demo-rt.jar: cannot compare its content with the deployed jar: " + strings.ReplaceAll(tc.why, "BUILT", f.builtJar("rt"))
		if code != ExitVersionUnknown || !strings.HasPrefix(f.err, want) || strings.Contains(f.out, "copy ok") {
			t.Errorf("%s: exit %d, stderr %.400q", name, code, f.err)
		}
	}
}
