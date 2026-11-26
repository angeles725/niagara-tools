package signinginventory

import (
	"path/filepath"
	"strings"
	"testing"
)

const moduleXML = `<module name="Mod-rt"><dependencies><dependency name="baja" vendor="Tridium" vendorVersion="4.14"/>` +
	`</dependencies></module>`

func envelope(module string, fields ...string) string {
	return "<!-- build-state.v1 -->\nmodule: " + module + "\n" + strings.Join(append(fields, ""), "\n") + "<!-- /build-state.v1 -->\n"
}

// modFx is a module tree with one built jar signed by signer ("" is unsigned), a build-state file and a
// profile over the one-alias PKCS12 store (good) or the four-alias JCEKS one.
type modFx struct {
	root, state, profile string
}

func newMod(t *testing.T, signer, record string, jceks bool) modFx {
	t.Helper()
	root := filepath.Join(t.TempDir(), "Mod")
	files := map[string]string{"META-INF/module.xml": moduleXML}
	if signer != "" {
		files["META-INF/"+signer+".SF"], files["META-INF/"+signer+".RSA"] = "sf", "rsa"
	}
	jar(t, filepath.Join(root, "Mod-rt/build/libs/Mod-rt.jar"), files)
	// jars the walk must not report: outside build/libs and under a vendored tree
	jar(t, filepath.Join(root, "lib/other.jar"), files)
	jar(t, filepath.Join(root, "node_modules/x/build/libs/x.jar"), files)
	store, kind := fx.p12, "PKCS12"
	if jceks {
		store, kind = fx.jceks, "JCEKS"
	}
	return modFx{root, write(t, root, "../state.md", record),
		write(t, root, "../p.xml", profileXML(store, kind, canaryStore))}
}

func (m modFx) run(t *testing.T, extra ...string) (int, string, string) {
	t.Helper()
	return inv(t, nil, append([]string{"--profile", m.profile, "--module", m.root, "--build-state", m.state}, extra...)...)
}

func TestModuleRecordedSignerInTheInventoryPasses(t *testing.T) {
	need(t)
	m := newMod(t, "GOOD", envelope("Mod", "expected_signer: Good"), false)
	code, out, errs := m.run(t)
	if code != 0 || errs != "" {
		t.Fatalf("exit %d, stderr %q\n%s", code, errs, out)
	}
	has(t, out, "INFO", "expected-signer", "Mod: Good (recorded in "+m.state+")")
	has(t, out, "PASS", "expected-signer-known", "good is in "+fx.p12+" and in date")
	has(t, out, "INFO", "jar", "Mod-rt/build/libs/Mod-rt.jar", "signed by GOOD", "compiled for Niagara 4.14")
	if strings.Count(out, "INFO  jar") != 1 {
		t.Errorf("only the build/libs jar outside node_modules is a built jar:\n%s", out)
	}
	quiet(t, out, errs)
}

func TestModuleFailRows(t *testing.T) {
	need(t)
	for name, c := range map[string]struct {
		signer, record string
		jceks          bool
		want           [][]string
	}{
		"alias in no store": {"GHOST", envelope("Mod", "expected_signer: ghost"), false, [][]string{
			{"FAIL", "expected-signer-known", "ghost is in none of the 1 store(s) listed (0 locked or unreadable)"}}},
		"alias only expired": {"OLD", envelope("Mod", "expected_signer: old"), true, [][]string{
			{"FAIL", "expected-signer-known", "old is only in expired form (in " + fx.jceks + ")"}}},
		"jar signed by another": {"OTHER", envelope("Mod", "expected_signer: good"), false, [][]string{
			{"PASS", "expected-signer-known"},
			{"FAIL", "jar-signer", "Mod-rt/build/libs/Mod-rt.jar: signed by OTHER, expected good (META-INF/GOOD.SF)"}}},
		"jar unsigned": {"", envelope("Mod", "expected_signer: good"), false, [][]string{
			{"FAIL", "jar-signer", "Mod-rt.jar: unsigned, expected good"}}},
		"malformed record": {"GOOD", envelope("Mod", "expected_signer: bad alias!"), false, [][]string{
			{"FAIL", "expected-signer", "Mod: envelope (Mod) expected_signer invalid: 'bad alias!'"}}},
	} {
		m := newMod(t, c.signer, c.record, c.jceks)
		code, out, errs := m.run(t)
		if code != 1 || errs != "" {
			t.Errorf("%s: exit %d, stderr %q\n%s", name, code, errs, out)
		}
		for _, w := range c.want {
			has(t, out, w...)
		}
	}
}

func TestModuleWithNoRecordPrintsInfoOnly(t *testing.T) {
	need(t)
	for name, rec := range map[string]string{"no envelope": "# nothing\n", "unknown": envelope("Mod", "expected_signer: unknown")} {
		m := newMod(t, "OTHER", rec, false)
		code, out, errs := m.run(t)
		if code != 0 || errs != "" || strings.Contains(out, "FAIL") {
			t.Errorf("%s: exit %d, stderr %q\n%s", name, code, errs, out)
		}
		has(t, out, "INFO", "expected-signer", "Mod: none recorded in "+m.state)
		has(t, out, "INFO", "jar", "signed by OTHER", "compiled for Niagara 4.14")
	}
}

func TestModuleNameAndBuildStateSelection(t *testing.T) {
	need(t)
	m := newMod(t, "GOOD", "", false)
	rec := envelope("Other-Name", "expected_signer: ghost")
	// --module-name picks the envelope; without it the directory name (Mod) is the module
	if code, out, _ := m.run(t, "--module-name", "Other-Name"); code != 0 || strings.Contains(out, "FAIL") {
		t.Errorf("no record for Other-Name: exit %d\n%s", code, out)
	}
	write(t, filepath.Dir(m.state), "state.md", rec)
	if code, out, _ := m.run(t, "--module-name", "Other-Name"); code != 1 {
		t.Errorf("a ghost signer for Other-Name must FAIL: exit %d\n%s", code, out)
	}
	if code, out, _ := m.run(t); code != 0 || !strings.Contains(out, "Mod: none recorded") {
		t.Errorf("default module name is the directory's: exit %d\n%s", code, out)
	}
	// no --build-state: <root>/BUILD-STATE.md first, else the kit's BUILD-STATE.src.md beside the toolbelt
	kit := t.TempDir()
	write(t, kit, "BUILD-STATE.src.md", envelope("Mod", "expected_signer: kitalias"))
	env := map[string]string{ToolbeltEnv: filepath.Join(kit, "toolbelt")}
	args := []string{"--profile", m.profile, "--module", m.root}
	_, out, _ := inv(t, env, args...)
	has(t, out, "FAIL", "expected-signer-known", "kitalias")
	write(t, m.root, "BUILD-STATE.md", envelope("Mod", "expected_signer: good"))
	code, out, _ := inv(t, env, args...)
	has(t, out, "PASS", "expected-signer-known", "good is in")
	if code != 0 {
		t.Errorf("exit %d\n%s", code, out)
	}
}

func TestModuleUnreadableInputsExit2(t *testing.T) {
	need(t)
	m := newMod(t, "GOOD", "", false)
	code, out, errs := inv(t, nil, "--profile", m.profile, "--module", m.root, "--build-state", m.root+"/none.md")
	if code != 2 || out != "" || !strings.Contains(errs, "none.md") {
		t.Errorf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
	if code, _, _ := inv(t, nil, "--module", m.state); code != 2 {
		t.Errorf("a module root that is a file: exit %d, want 2", code)
	}
}

// Without --module the inventory prints no module row, and a locked store makes the recorded alias a FAIL that
// says so (the alias may sit in the store that could not be listed).
func TestModuleRowsOnlyWithModuleAndLockedStoresAreCounted(t *testing.T) {
	need(t)
	m := newMod(t, "GOOD", envelope("Mod", "expected_signer: good"), false)
	_, out, _ := inv(t, nil, "--profile", m.profile)
	if strings.Contains(out, "jar") || strings.Contains(out, "expected-signer") {
		t.Errorf("module rows without --module:\n%s", out)
	}
	locked := write(t, filepath.Dir(m.state), "locked.xml", profileXML(fx.p12, "PKCS12", ""))
	code, out, _ := inv(t, nil, "--profile", locked, "--module", m.root, "--build-state", m.state)
	if code != 1 {
		t.Errorf("exit %d", code)
	}
	has(t, out, "FAIL", "expected-signer-known", "good is in none of the 0 store(s) listed (1 locked or unreadable)")
}

func TestModuleRecordedNiagaraTargetIsShownAndComparedWithEachJar(t *testing.T) {
	need(t)
	m := newMod(t, "GOOD", envelope("Mod", "expected_signer: Good", "niagara_target: 4.15.6.20"), false)
	code, out, errs := m.run(t)
	if code != 1 {
		t.Fatalf("a jar compiled for 4.14 under a 4.15 target must FAIL (exit 1), got %d\n%s", code, out)
	}
	has(t, out, "INFO", "niagara-target", "Mod: Niagara 4.15.6.20 (recorded in "+m.state+")")
	has(t, out, "FAIL", "jar-target", "Mod-rt/build/libs/Mod-rt.jar: compiled for Niagara 4.14, but the module records niagara_target 4.15.6.20")
	quiet(t, out, errs)

	m = newMod(t, "GOOD", envelope("Mod", "expected_signer: Good", "niagara_target: 4.14.0.162"), false)
	code, out, errs = m.run(t)
	if code != 0 || strings.Contains(out, "jar-target") {
		t.Fatalf("a matching major.minor adds no FAIL row, got exit %d\n%s", code, out)
	}
	has(t, out, "INFO", "niagara-target", "Mod: Niagara 4.14.0.162")
	quiet(t, out, errs)
}
