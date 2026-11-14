package checkinstalledline

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// versionPairs are the compare cases; TestCmpMatchesTheShellLib replays them on the bash comparator.
var versionPairs = []struct {
	a, b string
	want string // what vv_cmp prints
}{
	{"2.9.2", "2.10.0", "-1"}, {"2.10.0", "2.9.2", "1"}, {"2.8.0", "2.8.0", "0"},
	{"1.10", "1.9", "1"}, {"1.3", "1.3.0", "0"}, {"1.03", "1.3", "0"}, {"1.3", "1.3.1", "-1"},
	{"2.10.0-rc", "2.10.0", "incomparable"}, {"1.3-rc", "1.3-rc", "0"}, {"1.4-rc", "1.3-rc", "1"},
	{"", "1.0", "incomparable"}, {"1.0", "", "incomparable"}, {"v1.0", "v1.0", "incomparable"},
	{"1.2.", "1.2.", "0"}, {"7", "007", "0"}, {"12345678901234567890", "12345678901234567891", "-1"},
}

func cmpText(a, b string) string {
	c, ok := Cmp(a, b)
	if !ok {
		return "incomparable"
	}
	return map[int]string{-1: "-1", 0: "0", 1: "1"}[c]
}

func TestCmp(t *testing.T) {
	t.Parallel()
	for _, p := range versionPairs {
		if got := cmpText(p.a, p.b); got != p.want {
			t.Errorf("Cmp(%q, %q) = %s, want %s", p.a, p.b, got, p.want)
		}
	}
}

var sourceTexts = []struct{ name, text, want string }{
	{"the call", "niagaraModule {\n  defaultModuleVersion(\"2.10.0\")\n}\n", "2.10.0"},
	{"a comment naming it is skipped", "  // defaultModuleVersion sets \"vendorVersion\"\n\tdefaultModuleVersion( \"1.3\" )\n", "1.3"},
	{"the first call wins", "defaultModuleVersion(\"1.0\")\ndefaultModuleVersion(\"2.0\")\n", "1.0"},
	{"a commented call is skipped", "// defaultModuleVersion(\"9.9\")\n", ""},
	{"no call", "plugins { }\n", ""},
	{"a variable argument is not a version", "defaultModuleVersion(moduleVersion)\n", ""},
	{"no final newline", "defaultModuleVersion(\"4.5.6\")", "4.5.6"},
	{"an empty version reads as absent", "defaultModuleVersion(\"\")\ndefaultModuleVersion(\"2.0\")\n", ""},
}

func TestSourceVersion(t *testing.T) {
	t.Parallel()
	for _, c := range sourceTexts {
		if got := SourceVersion(c.text); got != c.want {
			t.Errorf("%s: SourceVersion = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestMatchesTheShellLib pins this package to toolbelt/lib/vendor-version.sh, the reader and comparator
// build.sh and ng-deploy.sh still source: vv_cmp and vv_source_version must answer every case above the
// same way, so the Go guard and the bash gates never disagree on a version.
func TestMatchesTheShellLib(t *testing.T) {
	t.Parallel()
	lib, err := filepath.Abs("../../../build-n4-module-kit/toolbelt/lib/vendor-version.sh")
	if err != nil {
		t.Fatal(err)
	}
	bash := func(script string, stdin string, args ...string) string {
		cmd := exec.Command("bash", append([]string{"-c", `. "$1"; shift; ` + script, "_", lib}, args...)...)
		cmd.Stdin = strings.NewReader(stdin)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("bash %s: %v", script, err)
		}
		return strings.TrimRight(string(out), "\n")
	}
	for _, p := range versionPairs {
		if p.a == "12345678901234567890" {
			continue // beyond awk's exact integers; Cmp compares the digits exactly
		}
		if got := bash(`vv_cmp "$1" "$2"`, "", p.a, p.b); got != cmpText(p.a, p.b) {
			t.Errorf("vv_cmp %q %q = %s, Cmp = %s", p.a, p.b, got, cmpText(p.a, p.b))
		}
	}
	for _, c := range sourceTexts {
		if got := bash(`vv_source_version`, c.text); got != SourceVersion(c.text) {
			t.Errorf("%s: vv_source_version = %q, SourceVersion = %q", c.name, got, SourceVersion(c.text))
		}
	}
}
