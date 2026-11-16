package checkbatsskips

import "testing"

const oneSkip = "1..1\nok 1 a # skip tool x not installed\n"

// TestMainRefusesALineOutsideTheDialect: an escape or a class the stated dialect does not hold is exit 3
// naming the construct and the line, even though Go compiles it; matched, it would mean something else
// than the author wrote (\< a literal `<`, \n a newline, [[=a=]] a set of four characters) and the skip
// would read UNLISTED or, worse, allowlisted (retro go-check-scripts-ports Δ3).
func TestMainRefusesALineOutsideTheDialect(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, line, construct string }{
		{"a GNU word boundary", `^\<tool x not installed$`, `\<`},
		{"a GNU word end", `^tool\> x not installed$`, `\>`},
		{"a control escape", `^tool x not installed\n?$`, `\n`},
		{"a hex escape", `^tool \x78 not installed$`, `\x`},
		{"an octal escape", `^tool \170 not installed$`, `\1`},
		{"a backslash in a bracket expression", `^tool [a-z\]]+ not installed$`, `\]`},
		{"a class name Go adds", `^tool [[:word:]]+ not installed$`, `[:word:]`},
		{"an equivalence class", `^tool [[=x=]] not installed$`, `[=x=]`},
		{"a collating symbol", `^tool [[.x.]] not installed$`, `[.x.]`},
		{"the first one is named", `^tool [[:word:]] \< not installed$`, `[:word:]`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tap, allow := fixture(t, oneSkip, "# why\n^tool [a-z]+ not installed$\n"+c.line+"\n")
			want := "check-bats-skips: allowlist line uses " + c.construct +
				", outside the dialect (POSIX ERE as Go regexp.CompilePOSIX reads it; tests/bats-skip-allowlist.txt header): " + c.line + "\n"
			if code, out, errText := run(tap, allow); code != ExitEnv || out != "" || errText != want {
				t.Errorf("exit %d\nstdout %q\nstderr %q\nwant exit %d\nstderr %q", code, out, errText, ExitEnv, want)
			}
		})
	}
}

// TestMainKeepsEveryConstructOfTheDialect: escaped ERE operators, the twelve POSIX classes, a bracket
// expression that opens with `]` or holds `[`, intervals and alternation still match as before.
func TestMainKeepsEveryConstructOfTheDialect(t *testing.T) {
	t.Parallel()
	cases := []struct{ reason, line string }{
		{`tool (x) [y] {z} a.b*c+d?e|f^g$h\i`, `^tool \(x\) \[y\] \{z\} a\.b\*c\+d\?e\|f\^g\$h\\i$`},
		{"aZ9 \t!x", `^[[:alpha:]][[:upper:]][[:digit:]][[:blank:]][[:space:]][[:punct:]][[:lower:]]$`},
		{"a1 F~", `^[[:alnum:]][[:xdigit:]][[:print:]][[:graph:]][^[:cntrl:]]$`},
		{"]-[", `^[]a-]+[[]$`},
		{"^a", `^[a^]{2}$`},
		{"abab x", `^(ab){2,3} (x|y){1,}$`},
	}
	for _, c := range cases {
		tap, allow := fixture(t, "1..1\nok 1 a # skip "+c.reason+"\n", c.line+"\n")
		if code, out, errText := run(tap, allow); code != ExitOK || out != "check-bats-skips: 1 skip(s), all allowlisted\n" || errText != "" {
			t.Errorf("%s: exit %d stdout %q stderr %q", c.line, code, out, errText)
		}
	}
}

// TestMainKeepsTheInvalidLineMessage: a line Go does not compile keeps its message, whatever it holds.
func TestMainKeepsTheInvalidLineMessage(t *testing.T) {
	t.Parallel()
	for _, line := range []string{`^tool \w+ not installed$`, `^(tool) \1$`, `*tool`, `^tool \<( x$`, `^tool x\`, `^tool x not \installed$`, `^tool x \— not installed$`} {
		tap, allow := fixture(t, oneSkip, line+"\n")
		want := "check-bats-skips: allowlist line is not a valid ERE: " + line + "\n"
		if code, out, errText := run(tap, allow); code != ExitEnv || out != "" || errText != want {
			t.Errorf("%s: exit %d stdout %q stderr %q, want exit 3 stderr %q", line, code, out, errText, want)
		}
	}
}
