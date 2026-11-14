package skillinstall

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

const (
	installerText = "#!/usr/bin/env bash\n# install-skill.sh — fixture header.\nset -u\n" +
		"SKILLS=\"build-n4-module:build-n4-module-kit/skill/SKILL.md mcp-n4:mcp-n4-kit/skill/SKILL.md\"\n"
	driftText   = "#!/usr/bin/env bash\n# check-skill-drift.sh — fixture header.\n#\n#   second line\nset -u\n"
	wantUsage   = "usage: install-skill.sh [--skill <build-n4-module|mcp-n4>] [--home <dir>] [--dry-run] [--force]\n"
	buildSkill  = "stub build skill\n"
	mcpSkill    = "stub mcp skill\n"
	kitStateLog = "kit-state.log"
)

// fixture is a throwaway repository tree (scripts/, the two tracked launchers), a home and a stub kit;
// nothing in it is the real repository or the real home.
type fixture struct {
	t                     *testing.T
	root, home, kit       string
	installer, driftCheck string
	env                   map[string]string
}

func write(t *testing.T, path, text string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	base := t.TempDir()
	f := &fixture{t: t, root: filepath.Join(base, "repo"), home: filepath.Join(base, "home"), kit: filepath.Join(base, "stubkit")}
	f.installer = write(t, filepath.Join(f.root, "scripts", "install-skill.sh"), installerText)
	f.driftCheck = write(t, filepath.Join(f.root, "scripts", "check-skill-drift.sh"), driftText)
	write(t, f.tracked("build-n4-module"), buildSkill)
	write(t, f.tracked("mcp-n4"), mcpSkill)
	write(t, filepath.Join(f.kit, "METHODOLOGY.md"), "")
	// HOME is a path that must never be written: every case passes --home or sets it on purpose.
	f.env = map[string]string{"HOME": filepath.Join(base, "never-home"), "BUILD_N4_KIT": f.kit}
	return f
}

func (f *fixture) tracked(skill string) string {
	return filepath.Join(f.root, skill+"-kit", "skill", "SKILL.md")
}

func (f *fixture) installed(skill string) string {
	return filepath.Join(f.home, ".claude", "skills", skill, "SKILL.md")
}

// kitState gives the stub kit a toolbelt/kit-state.sh that logs its arguments and exits with code.
func (f *fixture) kitState(kit, code string) {
	write(f.t, filepath.Join(kit, "toolbelt", "kit-state.sh"),
		"#!/usr/bin/env bash\nprintf 'kit-state: wrote %s\\n' \"$2\"\nprintf '%s\\n' \"$*\" >> \"$2/"+kitStateLog+"\"\nexit "+code+"\n")
}

func (f *fixture) stdio(launcher string, out, errb *bytes.Buffer) cli.Stdio {
	return cli.Stdio{Out: out, Err: errb, Getenv: func(name string) string {
		if name == installtools.LauncherEnv {
			return launcher
		}
		return f.env[name]
	}}
}

// install runs install-skill with --home <fixture home> ahead of args.
func (f *fixture) install(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Main(append([]string{"--home", f.home}, args...), f.stdio(f.installer, &out, &errb))
	return code, out.String(), errb.String()
}

// drift runs check-skill-drift with --home <fixture home> ahead of args.
func (f *fixture) drift(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := DriftMain(append([]string{"--home", f.home}, args...), f.stdio(f.driftCheck, &out, &errb))
	return code, out.String(), errb.String()
}

func expect(t *testing.T, name string, code int, out, errText string, wantCode int, wantOut, wantErr string) {
	t.Helper()
	if code != wantCode || out != wantOut || errText != wantErr {
		t.Errorf("%s:\n got exit %d\n stdout %q\n stderr %q\nwant exit %d\n stdout %q\n stderr %q",
			name, code, out, errText, wantCode, wantOut, wantErr)
	}
}

// absent fails when path exists.
func absent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); err == nil {
		t.Errorf("%s exists; nothing should have been written", path)
	}
}

// driftNoHome runs check-skill-drift without --home, so the base is $HOME.
func (f *fixture) driftNoHome(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := DriftMain(args, f.stdio(f.driftCheck, &out, &errb))
	return code, out.String(), errb.String()
}
