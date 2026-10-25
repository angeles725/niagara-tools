package niagaratest

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// root wraps test-method elements in a testng-results document whose root attributes are attrs.
func root(attrs, methods string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>` + "\n<testng-results " + attrs + `><reporter-output/>` +
		`<suite name="S"><test name="T"><class name="test.com.example.ModuleA.BExampleTest">` + methods +
		"</class></test></suite></testng-results>\n"
}

const twoPass = `<test-method status="PASS" name="setup" is-config="true"/>` +
	`<test-method status="PASS" name="a"/><test-method status="PASS" name="b"/>`

// The XML edges of TestNG; the CLI rows of a green run, failures, skips, a missing file, the 20-row cap and a
// console log are the goldens of testdata/ (testdata/README.md). Every document is synthetic, built from the
// element and attribute shapes of a real station-test run.
func TestTestNG(t *testing.T) {
	t.Parallel()
	counts := `skipped="0" failed="0" ignored="0" total="2" passed="2"`
	cases := []struct {
		name, xml string
		reasons   []string
	}{
		{"empty file", "", []string{"no testng-results root (empty file)"}},
		{"another root", "<testsuite tests=\"2\"/>", []string{"root element <testsuite>, not <testng-results>"}},
		{"total 0", root(`skipped="0" failed="0" ignored="0" total="0" passed="0"`, ""), []string{"no tests ran (total=0)"}},
		{"malformed", root(counts, twoPass+`<test-method status="PASS" name="c"></class>`),
			[]string{"malformed XML: XML syntax error on line 2: element <test-method> closed by </class>"}},
		{"truncated", root(counts, twoPass)[:200], []string{"malformed XML: XML syntax error on line 2: unexpected EOF"}},
		{"counts disagree with the methods", root(counts, twoPass+`<test-method status="FAIL" name="c"/>`),
			[]string{"root counts passed 2, failed 0, skipped 0 disagree with the test-methods: 2 PASS, 1 FAIL, 0 SKIP"}},
		{"total is not the sum", root(`skipped="0" failed="0" ignored="1" total="2" passed="2"`, twoPass),
			[]string{"root total 2 is not passed+failed+skipped+ignored (3)", "1 ignored test-method (@Test(enabled = false) or " +
				"excluded; testng-results.xml names none): waive them with --allow-ignored-count 1 --allow-ignored-reason <why>"}},
		{"missing and unparsable attributes", root(`failed="0" total="2x" passed="-1"`, twoPass),
			[]string{`root attribute total="2x" is not a count`, `root attribute passed="-1" is not a count`,
				"root attribute skipped missing"}},
		{"unknown status", root(counts, twoPass+`<test-method status="RUNNING" name="c"/>`),
			[]string{`test-method c has status "RUNNING", not PASS, FAIL or SKIP`}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := TestNG(strings.NewReader(c.xml), IgnoredAllowance{})
			if r.Pass || !reflect.DeepEqual(r.Reasons, c.reasons) {
				t.Errorf("Pass %v, Reasons %q, want false, %q", r.Pass, r.Reasons, c.reasons)
			}
		})
	}
}

// TestTestNGFileTooLarge pins the size cap on a sparse file: nothing past MaxTestNGBytes is read.
func TestTestNGFileTooLarge(t *testing.T) {
	t.Parallel()
	p := filepath.Join(t.TempDir(), "testng-results.xml")
	if err := os.WriteFile(p, []byte(root(`skipped="0" failed="0" ignored="0" total="2" passed="2"`, twoPass)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(p, MaxTestNGBytes+1); err != nil {
		t.Fatal(err)
	}
	r := TestNGFile(p, IgnoredAllowance{})
	want := []string{"file larger than 64 MiB (67108864 bytes): not read"}
	if r.Pass || !reflect.DeepEqual(r.Reasons, want) {
		t.Errorf("Pass %v, Reasons %q, want false, %q", r.Pass, r.Reasons, want)
	}
}
