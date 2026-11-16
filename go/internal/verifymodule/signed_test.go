package verifymodule

import (
	"strings"
	"testing"
)

// signedJar is a one-class module jar plus the META-INF signature entries named in sigs.
func signedJar(sigs ...string) string {
	jar := "META-INF/module.xml=<module name=\"Foo\"><types><type name=\"A\" class=\"com.x.A\"/></types></module>" +
		"|com/x/A.class=\xca\xfe\xba\xbe\x00\x00\x00\x34"
	for _, s := range sigs {
		jar += "|META-INF/" + s + "=x"
	}
	return jar
}

// signedRun runs the gate on one jar built from sigs and returns the exit code, the signed row without its
// status, check and jar columns, the summary line and stderr.
func signedRun(t *testing.T, sigs []string, flags ...string) (code int, status, detail, summary, stderr string) {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, map[string]string{"Foo-rt.jar": signedJar(sigs...)})
	code, out, stderr := runMain(dir, append(flags, "@DIR@/Foo-rt.jar")...)
	for _, line := range strings.Split(out, "\n") {
		for _, st := range []string{"PASS", "FAIL"} {
			if d, ok := strings.CutPrefix(line, st+"  signed     @DIR@/Foo-rt.jar  "); ok {
				status, detail = st, d
			}
		}
		if strings.HasPrefix(line, "verify-module: ") {
			summary = line
		}
	}
	return code, status, detail, summary, stderr
}

// TestExpectedSigner: with --expected-signer ALIAS the signed row is FAIL (exit 1) unless the jar's complete
// signatures (an .SF with its block file) are exactly that one signer. The alias is compared in the form
// jarsigner names the files after: upper-cased, cut to 8 characters, a character outside A-Z 0-9 _ - as "_".
func TestExpectedSigner(t *testing.T) {
	const expects = "; expected only the signer "
	for _, tc := range []struct {
		name, alias  string
		sigs         []string
		code         int
		status, want string
	}{
		{"matches", "ACMESIGN", []string{"ACMESIGN.SF", "ACMESIGN.RSA"}, 0, "PASS",
			"signed by alias ACMESIGN, the expected signer ACMESIGN (META-INF/ACMESIGN.SF + signature block)"},
		{"differs", "ACMESIGN", []string{"NIAGARA4.SF", "NIAGARA4.RSA"}, 1, "FAIL",
			"signed by alias NIAGARA4" + expects + "ACMESIGN (META-INF/ACMESIGN.SF + signature block)"},
		{"two signers", "ACMESIGN", []string{"ACMESIGN.SF", "ACMESIGN.RSA", "NIAGARA4.SF", "NIAGARA4.EC"}, 1, "FAIL",
			"signed by alias ACMESIGN, NIAGARA4" + expects + "ACMESIGN (META-INF/ACMESIGN.SF + signature block)"},
		{"unsigned", "ACMESIGN", nil, 1, "FAIL",
			"no META-INF/*.SF (jar not signed)" + expects + "ACMESIGN (META-INF/ACMESIGN.SF + signature block)"},
		{"block file missing", "ACMESIGN", []string{"ACMESIGN.SF"}, 1, "FAIL",
			"META-INF/ACMESIGN.SF without its .RSA/.DSA/.EC signature block (jar not signed)" + expects +
				"ACMESIGN (META-INF/ACMESIGN.SF + signature block)"},
		{"alias longer than 8 characters", "acmesigner", []string{"ACMESIGN.SF", "ACMESIGN.DSA"}, 0, "PASS",
			"signed by alias ACMESIGN, the expected signer acmesigner (META-INF/ACMESIGN.SF + signature block)"},
		{"alias longer than 8 characters, another signer", "acmesigner", []string{"NIAGARA4.SF", "NIAGARA4.RSA"}, 1, "FAIL",
			"signed by alias NIAGARA4" + expects + "acmesigner (META-INF/ACMESIGN.SF + signature block)"},
		{"alias with characters jarsigner replaces", "a.b c+d", []string{"A_B_C_D.SF", "A_B_C_D.RSA"}, 0, "PASS",
			"signed by alias A_B_C_D, the expected signer a.b c+d (META-INF/A_B_C_D.SF + signature block)"},
		{"a signature file named after the whole alias", "CodeSigning", []string{"CodeSigning.SF", "CodeSigning.RSA"}, 0, "PASS",
			"signed by alias CodeSigning, the expected signer CodeSigning (META-INF/CodeSigning.SF + signature block)"},
		{"a second signer with lower-case extensions", "ACMESIGN", []string{"ACMESIGN.SF", "ACMESIGN.RSA", "NIAGARA4.sf", "NIAGARA4.rsa"}, 1, "FAIL",
			"signed by alias ACMESIGN, NIAGARA4" + expects + "ACMESIGN (META-INF/ACMESIGN.SF + signature block)"},
		{"mixed-case extensions of the expected signer", "ACMESIGN", []string{"ACMESIGN.Sf", "ACMESIGN.Ec"}, 0, "PASS",
			"signed by alias ACMESIGN, the expected signer ACMESIGN (META-INF/ACMESIGN.SF + signature block)"},
		{"the cut comes before the replacement", "acme-sig.ner", []string{"ACME-SIG.SF", "ACME-SIG.RSA"}, 0, "PASS",
			"signed by alias ACME-SIG, the expected signer acme-sig.ner (META-INF/ACME-SIG.SF + signature block)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, status, detail, summary, stderr := signedRun(t, tc.sigs, "--expected-signer", tc.alias)
			verdict := map[int]string{0: "ALL PASS", 1: "FAILED"}[tc.code]
			if code != tc.code || status != tc.status || detail != tc.want || stderr != "" || !strings.HasSuffix(summary, " -> "+verdict) {
				t.Errorf("exit %d, %s %q, summary %q, stderr %q\nwant exit %d, %s %q", code, status, detail, summary, stderr, tc.code, tc.status, tc.want)
			}
		})
	}
}

// TestExpectedSignerUsage: the option needs a non-empty alias (exit 2, no row); an empty one never turns
// the comparison off.
func TestExpectedSignerUsage(t *testing.T) {
	for name, args := range map[string][]string{"no value": {"@DIR@/Foo-rt.jar", "--expected-signer"}, "empty": {"--expected-signer", "", "@DIR@/Foo-rt.jar"}} {
		dir := t.TempDir()
		write(t, dir, map[string]string{"Foo-rt.jar": signedJar("ACMESIGN.SF", "ACMESIGN.RSA")})
		code, out, stderr := runMain(dir, args...)
		want := usage
		if name == "empty" {
			want = "verify-module: --expected-signer needs an alias\n" + usage
		}
		if code != 2 || out != "" || stderr != want {
			t.Errorf("%s: exit %d %q %q", name, code, out, stderr)
		}
	}
	if !strings.Contains(usage, "--expected-signer ALIAS") {
		t.Error("the usage text does not name --expected-signer")
	}
}

// TestSignedWithoutExpectedSigner: without the option the whole report is what it was before the option
// existed (recorded from the binary built at the commit before it), whatever signs the jar.
func TestSignedWithoutExpectedSigner(t *testing.T) {
	const row = "  signed     @DIR@/Foo-rt.jar  "
	rest := "PASS  types      @DIR@/Foo-rt.jar  1 declared types resolve to classes\n" +
		"SKIP  baja       @DIR@/Foo-rt.jar  no --target-version\n" +
		"SKIP  stored     @DIR@/Foo-rt.jar  no --stored\n"
	for _, c := range []string{"typecount", "facets   ", "facets-req", "ord-literal"} {
		rest += "SKIP  " + c + "  @DIR@/Foo-rt.jar  no --src\n"
	}
	rest += "SKIP  palette    @DIR@/Foo-rt.jar  no module.palette\nSKIP  wb-scaffold  @DIR@/Foo-rt.jar  not a -wb jar\n"
	for _, c := range []string{"phantom-dep", "moduletest", "cross-module-type", "transient-operator", "compact3-import",
		"subscription-leak", "slot-wall", "station-save"} {
		rest += "SKIP  " + c + "  @DIR@/Foo-rt.jar  no --src\n"
	}
	for _, tc := range []struct {
		name            string
		sigs            []string
		code            int
		signed, summary string
	}{
		{"one signer", []string{"NIAGARA4.SF", "NIAGARA4.RSA"}, 0,
			"PASS" + row + "signed by alias NIAGARA4 (META-INF/<ALIAS>.SF + signature block)", "3 passed, 0 failed, 16 skipped, 0 warned -> ALL PASS"},
		{"two signers", []string{"ACMESIGN.SF", "ACMESIGN.RSA", "NIAGARA4.SF", "NIAGARA4.EC"}, 0,
			"PASS" + row + "signed by alias ACMESIGN, NIAGARA4 (META-INF/<ALIAS>.SF + signature block)", "3 passed, 0 failed, 16 skipped, 0 warned -> ALL PASS"},
		// Signature files are read in any case of the extension, as the JDK verifier reads them. Declared
		// difference from the bash tool and from the gate before --expected-signer: these two jars were
		// "no META-INF/*.SF (jar not signed)", exit 1, and "signed by alias ACMESIGN" (one signer).
		{"lower-case extensions", []string{"ACMESIGN.sf", "ACMESIGN.rsa"}, 0,
			"PASS" + row + "signed by alias ACMESIGN (META-INF/<ALIAS>.SF + signature block)", "3 passed, 0 failed, 16 skipped, 0 warned -> ALL PASS"},
		{"a second signer with lower-case extensions", []string{"ACMESIGN.SF", "ACMESIGN.RSA", "NIAGARA4.sf", "NIAGARA4.dsa"}, 0,
			"PASS" + row + "signed by alias ACMESIGN, NIAGARA4 (META-INF/<ALIAS>.SF + signature block)", "3 passed, 0 failed, 16 skipped, 0 warned -> ALL PASS"},
		{"a lower-case .sf without its block", []string{"ACMESIGN.sf"}, 1,
			"FAIL" + row + "META-INF/ACMESIGN.sf without its .RSA/.DSA/.EC signature block (jar not signed)", "2 passed, 1 failed, 16 skipped, 0 warned -> FAILED"},
		{"unsigned", nil, 1, "FAIL" + row + "no META-INF/*.SF (jar not signed)", "2 passed, 1 failed, 16 skipped, 0 warned -> FAILED"},
		{"block file missing", []string{"ACMESIGN.SF"}, 1,
			"FAIL" + row + "META-INF/ACMESIGN.SF without its .RSA/.DSA/.EC signature block (jar not signed)", "2 passed, 1 failed, 16 skipped, 0 warned -> FAILED"},
	} {
		dir := t.TempDir()
		write(t, dir, map[string]string{"Foo-rt.jar": signedJar(tc.sigs...)})
		code, out, stderr := runMain(dir, "@DIR@/Foo-rt.jar")
		want := "PASS  bytecode   @DIR@/Foo-rt.jar  1 classes, all major 52\n" + tc.signed + "\n" + rest + "verify-module: " + tc.summary + "\n"
		if code != tc.code || stderr != "" || out != want {
			t.Errorf("%s: exit %d stderr %q\n%s\nwant\n%s", tc.name, code, stderr, out, want)
		}
	}
}
