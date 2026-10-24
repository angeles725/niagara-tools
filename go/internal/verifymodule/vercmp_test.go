package verifymodule

import "testing"

// TestVerLE pins ver_le against GNU sort -V (coreutils 9.4): le[i][j] is whether
// `printf '%s\n%s\n' vers[i] vers[j] | sort -V | head -1` printed vers[i]. It covers file suffixes
// (.tar.gz, .Final), leading zeros broken by byte order, '~' before the end, '-' after letters, and the
// "." / ".." / dot-name ranks.
func TestVerLE(t *testing.T) {
	vers := []string{"4.14", "4.14.0", "4.14.1", "4.15", "4.9", "4.014", "4.14a", "4.14~rc1", "4.14-rc1", "10.0",
		".", "..", ".hidden", "a.tar.gz", "a.tar", "1.2.3.Final", "1.2.3"}
	le := []string{
		"TTTTFFTFTTFFFTTFF", // 4.14
		"FTTTFFFFFTFFFTTFF", // 4.14.0
		"FFTTFFFFFTFFFTTFF", // 4.14.1
		"FFFTFFFFFTFFFTTFF", // 4.15
		"TTTTTTTTTTFFFTTFF", // 4.9
		"TTTTFTTFTTFFFTTFF", // 4.014
		"FTTTFFTFTTFFFTTFF", // 4.14a
		"TTTTFTTTTTFFFTTFF", // 4.14~rc1
		"FTTTFFFFTTFFFTTFF", // 4.14-rc1
		"FFFFFFFFFTFFFTTFF", // 10.0
		"TTTTTTTTTTTTTTTTT", // .
		"TTTTTTTTTTFTTTTTT", // ..
		"TTTTTTTTTTFFTTTTT", // .hidden
		"FFFFFFFFFFFFFTFFF", // a.tar.gz
		"FFFFFFFFFFFFFTTFF", // a.tar
		"TTTTTTTTTTFFFTTTF", // 1.2.3.Final
		"TTTTTTTTTTFFFTTTT", // 1.2.3
	}
	for i, a := range vers {
		for j, b := range vers {
			if got, want := verLE(a, b), le[i][j] == 'T'; got != want {
				t.Errorf("verLE(%q, %q) = %v, sort -V says %v", a, b, got, want)
			}
		}
	}
	if !verLE("", "x") || verLE("x", "") {
		t.Error("the empty version must sort first")
	}
}
