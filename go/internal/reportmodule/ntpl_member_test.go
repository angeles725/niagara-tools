package reportmodule

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// ntplArchive is a station template archive whose manifest lists deps.
func ntplArchive(t *testing.T, deps ...string) string {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	manifest := "<template isStation=\"true\">\n  <dependencies>\n"
	for _, d := range deps {
		manifest += "    <dependency name=\"" + d + "\"/>\n"
	}
	manifest += "  </dependencies>\n</template>\n"
	for name, body := range map[string]string{"template.bog": "<bajaObjectGraph version=\"4.0\"/>\n", "template-manifest.xml": manifest} {
		w, err := z.Create(name)
		must(t, err)
		_, err = w.Write([]byte(body))
		must(t, err)
	}
	must(t, z.Close())
	return b.String()
}

// ntplRun runs report-module --lint-only with args before the module root and env added to the process
// environment, with lint-ntpl-manifest-deps the only real member.
func ntplRun(t *testing.T, root string, env []string, args ...string) (int, string) {
	t.Helper()
	repo, err := filepath.Abs("../../..")
	must(t, err)
	dir := t.TempDir()
	tb := filepath.Join(dir, "toolbelt")
	must(t, os.MkdirAll(tb, 0o755))
	must(t, os.WriteFile(filepath.Join(tb, ".stub-rest"), nil, 0o644))
	must(t, os.Symlink(filepath.Join(repo, "build-n4-module-kit", "toolbelt", ntplScript+".sh"), filepath.Join(tb, ntplScript+".sh")))
	toolbeltCopy(t, repo, dir)
	var o, e bytes.Buffer
	environ := []string{}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "NIAGARA_HOME=") {
			environ = append(environ, kv)
		}
	}
	s := cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string {
		if k == ToolbeltEnv {
			return tb
		}
		return os.Getenv(k)
	}}
	code := mainEnv(append(append([]string{"--lint-only"}, args...), root), s, append(environ, env...))
	return code, o.String() + e.String()
}

// TestTemplateLintIsAModuleMember (K2, K8, K3): report-module runs lint-ntpl-manifest-deps once on the module
// root with the installation it is given (--niagara-home, else NIAGARA_HOME in the members' environment). A
// station template that lists stock modules the installation ships passes; one that needs a module nobody
// declares or ships is a FAIL row and exit 1; without an installation the rows are WARN and do not block.
func TestTemplateLintIsAModuleMember(t *testing.T) {
	if testing.Short() {
		t.Skip("runs toolbelt members as subprocesses")
	}
	nh := t.TempDir()
	for _, jar := range []string{"baja.jar", "fox-rt.jar", "web-rt.jar"} {
		p := filepath.Join(nh, "modules", jar)
		must(t, os.MkdirAll(filepath.Dir(p), 0o755))
		must(t, os.WriteFile(p, nil, 0o644))
	}
	const station = "ModA-rt/src/newStations/Plant.ntpl"
	build := map[string]string{"ModA-rt/ModA-rt.gradle.kts": "dependencies { api(\":local-rt\") }\n"}
	tree := func(deps ...string) string {
		files := map[string]string{station: ntplArchive(t, deps...)}
		for k, v := range build {
			files[k] = v
		}
		return moduleTree(t, files)
	}
	stock := tree("baja", "fox-rt", "web-rt", "local-rt")
	if code, out := ntplRun(t, stock, nil, "--niagara-home", nh); code != ExitClean || !strings.Contains(out, "(module)  PASS  lint-ntpl-manifest-deps  clean") {
		t.Errorf("stock modules with --niagara-home: exit %d, want 0 and the PASS row:\n%s", code, out)
	}
	ghost := tree("baja", "ghost-rt")
	wantFail := "(module)  FAIL  ntpl-deps  " + ghost + "/" + station + ": manifest dependencies neither declared"
	if code, out := ntplRun(t, ghost, nil, "--niagara-home", nh); code != ExitFail || !strings.Contains(out, wantFail) || !strings.Contains(out, "installation "+nh+": ghost-rt") {
		t.Errorf("a missing module with --niagara-home: exit %d, want 1 and %q:\n%s", code, wantFail, out)
	}
	if code, out := ntplRun(t, ghost, []string{"NIAGARA_HOME=" + nh}); code != ExitFail || !strings.Contains(out, wantFail) {
		t.Errorf("a missing module with NIAGARA_HOME: exit %d, want 1 and %q:\n%s", code, wantFail, out)
	}
	for name, args := range map[string][]string{"no installation": nil, "an empty --niagara-home is absent": {"--niagara-home", ""}} {
		code, out := ntplRun(t, ghost, nil, args...)
		if code != ExitClean || !strings.Contains(out, "(module)  WARN  ntpl-deps  "+ghost+"/"+station+": manifest dependencies not declared") ||
			!strings.Contains(out, "(module)  PASS  lint-ntpl-manifest-deps  clean") {
			t.Errorf("%s: exit %d, want 0 with the WARN row relayed and the PASS row:\n%s", name, code, out)
		}
	}
}
