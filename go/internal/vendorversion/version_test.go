package vendorversion

import (
	"testing"
)

// versionPairs are the compare cases; TestCmpMatchesTheShellLib replays them on the bash comparator.
var versionPairs = []struct {
	a, b string
	want string // what the retired vv_cmp printed
}{
	{"2.9.2", "2.10.0", "-1"}, {"2.10.0", "2.9.2", "1"}, {"2.8.0", "2.8.0", "0"},
	{"1.10", "1.9", "1"}, {"1.3", "1.3.0", "0"}, {"1.03", "1.3", "0"}, {"1.3", "1.3.1", "-1"},
	{"2.10.0-rc", "2.10.0", "incomparable"}, {"1.3-rc", "1.3-rc", "0"}, {"1.4-rc", "1.3-rc", "1"},
	{"", "1.0", "incomparable"}, {"1.0", "", "incomparable"}, {"v1.0", "v1.0", "incomparable"},
	{"1.2.", "1.2.", "0"}, {"7", "007", "0"}, {"12345678901234567890", "12345678901234567891", "-1"},
	// the cases of the retired tests/vendor-version.bats that the rows above did not hold
	{"1.3.0", "1.3", "0"}, {"1.3", "1.4", "-1"}, {"2.10.2", "2.10.10", "-1"}, {"2.10.10", "2.10.2", "1"},
	{"1.3.1", "1.3", "1"}, {"1.3-rc", "1.3.0-rc", "0"}, {"1.3-rc", "1.4-rc", "-1"},
	{"1.3-rc", "1.3", "incomparable"}, {"1.4", "1.3-beta", "incomparable"}, {"", "1.3", "incomparable"},
	{"1.3", "", "incomparable"}, {"abc", "abc", "incomparable"},
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
	{"comments, a commented call, then two calls", "niagaraModule {\n  // defaultModuleVersion sets the \"vendorVersion\" attribute on all modules\n" +
		"  // defaultModuleVersion(\"9.9.9\")\n  defaultModuleVersion( \"2.10.0\" )\n  defaultModuleVersion(\"1.0.0\")\n}\n", "2.10.0"},
	{"a build file with no call", "plugins { id(\"x\") }\n", ""},
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
