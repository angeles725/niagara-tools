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

// TestMainRefusesEachBracketSubGuard pins the three sub-guards of the bracket scan, one smallest line each:
// a class opener with no closer, a `]` that is a member because it comes first (so the backslash after it
// is still inside the bracket expression), and a POSIX class name written as an equivalence class.
func TestMainRefusesEachBracketSubGuard(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, line, construct string }{
		{"a class opener that is never closed", `[[:]`, `[:`},
		{"a backslash after a leading member bracket", `[]a\]]`, `\]`},
		{"a POSIX class name between equals signs", `[[=alpha=]]`, `[=alpha=]`},
	}
	for _, c := range cases {
		tap, allow := fixture(t, oneSkip, c.line+"\n")
		want := "check-bats-skips: allowlist line uses " + c.construct +
			", outside the dialect (POSIX ERE as Go regexp.CompilePOSIX reads it; tests/bats-skip-allowlist.txt header): " + c.line + "\n"
		if code, out, errText := run(tap, allow); code != ExitEnv || out != "" || errText != want {
			t.Errorf("%s: exit %d stdout %q stderr %q, want exit 3 stderr %q", c.name, code, out, errText, want)
		}
	}
}

// TestMainRefusesWhatGNUGrepReadsDifferently: three constructs Go compiles and GNU grep -E reads another
// way are outside the dialect, each named: `{,2}` (a literal for Go, the interval 0 to 2 for GNU), a
// class name in single brackets (a set of those letters for Go; GNU refuses it) and `{}` (a literal for
// Go; GNU refuses it).
func TestMainRefusesWhatGNUGrepReadsDifferently(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, line, construct string }{
		{"an interval with no lower bound", `x{,2}`, `{,2}`},
		{"a class name in single brackets", `[:alpha:]+`, `[:alpha:]`},
		{"an empty interval", `a{}`, `{}`},
		{"a negated class name in single brackets", `^tool [^:alpha:] not installed$`, `[^:alpha:]`},
		{"a brace that opens no interval", `^tool x{ not installed$`, `{`},
	}
	for _, c := range cases {
		tap, allow := fixture(t, oneSkip, c.line+"\n")
		want := "check-bats-skips: allowlist line uses " + c.construct +
			", outside the dialect (POSIX ERE as Go regexp.CompilePOSIX reads it; tests/bats-skip-allowlist.txt header): " + c.line + "\n"
		if code, out, errText := run(tap, allow); code != ExitEnv || out != "" || errText != want {
			t.Errorf("%s: exit %d stdout %q stderr %q, want exit 3 stderr %q", c.name, code, out, errText, want)
		}
	}
}

// TestMainKeepsBracesAndColonsOfTheDialect: what sits next to the refused constructs still matches as
// before: the three interval forms, an escaped brace, a brace or a colon as a bracket member, and a
// bracket expression that holds a class between two colons.
func TestMainKeepsBracesAndColonsOfTheDialect(t *testing.T) {
	t.Parallel()
	cases := []struct{ reason, line string }{
		{"aa bbb cc", `^a{2} b{2,} c{1,3}$`},
		{"x{,2} {}", `^x\{,2\} \{\}$`},
		{"{:} a:", `^[{][:][}] [:a][a:]$`},
		{":b:", `^[:[:alpha:]:]{3}$`},
	}
	for _, c := range cases {
		tap, allow := fixture(t, "1..1\nok 1 a # skip "+c.reason+"\n", c.line+"\n")
		if code, out, errText := run(tap, allow); code != ExitOK || out != "check-bats-skips: 1 skip(s), all allowlisted\n" || errText != "" {
			t.Errorf("%s: exit %d stdout %q stderr %q", c.line, code, out, errText)
		}
	}
}
