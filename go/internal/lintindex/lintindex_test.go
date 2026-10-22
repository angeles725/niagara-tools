package lintindex

import "testing"

// lint builds a header the way tests/gen-lint-index.bats _mklint does.
func lint(name, desc, usage, exit string) string {
	s := "#!/usr/bin/env bash\n# " + name + " — " + desc + "\n#\n"
	if usage != "" {
		s += "# Usage:  " + usage + "\n"
	}
	if exit != "" {
		s += "#   Exit: " + exit + "\n"
	}
	return s + "exit 0\n"
}

func TestExtract(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, file, text string
		want             Row
		problem          string
	}{
		{"GLI-fields", "lint-gamma.sh",
			lint("lint-gamma.sh", "flags gamma [ev: retro some-retro-stem Δ2]", "lint-gamma.sh [--strict] <src-root>", "0 no WARN · 1 any WARN under --strict · 3 usage/env"),
			Row{Desc: "flags gamma", Usage: "lint-gamma.sh [--strict] <src-root>", Exit: "0 no WARN · 1 any WARN under --strict · 3 usage/env", Ev: "[ev: retro some-retro-stem Δ2]"}, ""},
		{"GLI-wrap: description continued on line 3", "lint-delta.sh",
			"#!/usr/bin/env bash\n# lint-delta.sh — flags delta things that\n# span two header lines.\n#\n# Usage: lint-delta.sh <src>\n# Exits: 0 clean · 3 usage\n",
			Row{Desc: "flags delta things that span two header lines.", Usage: "lint-delta.sh <src>", Exit: "0 clean · 3 usage"}, ""},
		{"GLI-pipe: a literal | is escaped", "lint-pipe.sh",
			lint("lint-pipe.sh", "flags a || b", "lint-pipe.sh <src>", "0 clean | 1 any FAIL | 3 usage"),
			Row{Desc: `flags a \|\| b`, Usage: "lint-pipe.sh <src>", Exit: `0 clean \| 1 any FAIL \| 3 usage`}, ""},
		{"GLI-contract: no Usage line", "lint-nousage.sh", lint("lint-nousage.sh", "no usage line.", "", "0"),
			Row{}, `no "# Usage:" line in the header (lines 1-60)`},
		{"GLI-contract-exit: no Exit line", "lint-noexit.sh", lint("lint-noexit.sh", "no exit line.", "lint-noexit.sh <src>", ""),
			Row{}, `no "# Exit:"/"# Exits:" line in the header (lines 1-60)`},
		{"GLI-wrap-exit: aligned continuation joined, explanation is not", "lint-wrap.sh",
			"#!/usr/bin/env bash\n# lint-wrap.sh — wraps its exit contract.\n#\n# Usage:  lint-wrap.sh <src>\n#   Scans the tree (explanation, not part of the usage).\n# Exits:  0 clean · 1 any FAIL, or any\n#         STALE under --strict · 3 usage/env\n# VCS-free by design.\n",
			Row{Desc: "wraps its exit contract.", Usage: "lint-wrap.sh <src>", Exit: "0 clean · 1 any FAIL, or any STALE under --strict · 3 usage/env"}, ""},
		{"GLI-multi: every Usage and Exit line, in order", "lint-two.sh",
			"#!/usr/bin/env bash\n# lint-two.sh — has two usages and two exit lines.\n#\n# Usage: lint-two.sh <src>\n# Usage: lint-two.sh --strict <src>\n# Exit: 0 clean · 1 any FAIL\n# Exit: 2 usage · 3 env\n",
			Row{Desc: "has two usages and two exit lines.", Usage: "lint-two.sh <src><br>lint-two.sh --strict <src>", Exit: "0 clean · 1 any FAIL<br>2 usage · 3 env"}, ""},
		{"GLI-cap: description joins header lines 2-5 only", "lint-long.sh",
			"#!/usr/bin/env bash\n# lint-long.sh — one\n# two\n# three\n# four\n# five-is-past-the-cap\n#\n# Usage: lint-long.sh <src>\n# Exit: 0 clean\n",
			Row{Desc: "one two three four", Usage: "lint-long.sh <src>", Exit: "0 clean"}, ""},
		{"empty file", "lint-empty.sh", "", Row{}, `line 2 is not "# lint-empty.sh — <what it checks>"`},
		{"line 2 names another script", "lint-x.sh", "#!/bin/sh\n# lint-other.sh — d\n# Usage: x\n# Exit: 0\n", Row{}, `line 2 is not "# lint-x.sh — <what it checks>"`},
		{"line 2 without a separator", "lint-n.sh", "#!/bin/sh\n# lint-n.sh no separator\n# Usage: x\n# Exit: 0\n", Row{}, `line 2 is not "# lint-n.sh — <what it checks>"`},
		{"-- and - separators, Exit with no colon", "lint-d.sh", "#!/bin/sh\n# lint-d.sh -- dd\n# Usage: u\n# Exits 0 ok\n", Row{Desc: "dd", Usage: "u", Exit: "0 ok"}, ""},
		{"Usage past line 60 is not read", "lint-late.sh", "#!/bin/sh\n# lint-late.sh — d\n" + filler(58) + "# Usage: late\n# Exit: 0\n", Row{}, `no "# Usage:" line in the header (lines 1-60)`},
		{"line 60 read, line 61 not", "lint-l.sh", "#!/bin/sh\n# lint-l.sh — d\n" + filler(56) + "# Usage: u59\n# Exit: e60\n# Exit: e61\n", Row{Desc: "d filler filler filler", Usage: "u59", Exit: "e60"}, ""},
		{"CRLF header", "lint-crlf.sh", "#!/bin/sh\r\n# lint-crlf.sh — crlf desc\r\n# more\r\n#\r\n# Usage: lint-crlf.sh <x>\r\n#        cont\r\n# Exit: 0 ok\r\n",
			Row{Desc: "crlf desc more", Usage: "lint-crlf.sh <x> cont", Exit: "0 ok"}, ""},
		{"[ev:] tags deduplicated from lines 1-60, stripped from the description only", "lint-ev2.sh",
			"#!/bin/sh\n# lint-ev2.sh — d [ev: a] [ev: a]\n#\n# Usage: `u` here\n# Exit: e\n#   [ev: late tag] Mutation: foo\n",
			Row{Desc: "d", Usage: "`u` here", Exit: "e", Ev: "[ev: a] [ev: late tag]"}, ""},
		{"empty separator rest is joined from line 3", "lint-em.sh", "#!/bin/sh\n# lint-em.sh —\n# continued only\n# Usage: u\n# Exit: e\n", Row{Desc: "continued only", Usage: "u", Exit: "e"}, ""},
		{"empty first Usage/Exit lines are skipped", "lint-u.sh", "#!/bin/sh\n# lint-u.sh — d\n# Usage:\n# Usage: second\n# Exit:\n# Exit: real\n", Row{Desc: "d", Usage: "second", Exit: "real"}, ""},
		{"only empty Usage values", "lint-u2.sh", "#!/bin/sh\n# lint-u2.sh — d\n# Usage:\n# Exit:   \n", Row{}, `no "# Usage:" line in the header (lines 1-60)`},
		{"continuation stops at a Word: line and a less-indented line", "lint-k.sh",
			"#!/bin/sh\n# lint-k.sh — d\n# Usage: a b\n#        Note: c\n#        cont c\n#    less\n# Exit: e\n#   Row: x\n#       more\n# Usage: z\n",
			Row{Desc: "d", Usage: "a b<br>z", Exit: "e"}, ""},
		{"a Usage on line 3 stops the description join; Exitless is no Exit line", "lint-j3.sh", "#!/bin/sh\n# lint-j3.sh — d1\n# Usage: u on line 3\n# Exitless\n# Exit: e\n", Row{Desc: "d1", Usage: "u on line 3", Exit: "e"}, ""},
		{"tabs collapse to one space", "lint-t.sh", "#!/bin/sh\n# lint-t.sh — tab\there   spaced\n# Usage:\tu\tv\n# Exit:\t0\n", Row{Desc: "tab here spaced", Usage: "u v", Exit: "0"}, ""},
		{"no final newline", "lint-nonl.sh", "#!/bin/sh\n# lint-nonl.sh — d\n# Usage: u\n# Exit: e", Row{Desc: "d", Usage: "u", Exit: "e"}, ""},
		{"R2/R3/R4-gli-exit: a # Exit/# Usage comment after the header block does not join", "lint-stray.sh",
			"#!/bin/sh\n# lint-stray.sh — d\n# Usage: u\n# Exit: 0 clean\nset -u\n# Exit early when the tree is empty\n# Usage: not a usage\nexit 0\n",
			Row{Desc: "d", Usage: "u", Exit: "0 clean"}, ""},
		{"a header with its Exit line only after code is a contract violation", "lint-late-exit.sh",
			"#!/bin/sh\n# lint-late-exit.sh — d\n# Usage: u\nset -u\n# Exit: 0\n", Row{}, `no "# Exit:"/"# Exits:" line in the header (lines 1-60)`},
		{"description of [ev:] tags only keeps the columns in place", "lint-ev.sh",
			"#!/bin/sh\n# lint-ev.sh — [ev: only tag]\n# Usage: u [ev: in usage]\n# Exit: e [ev: only tag] [ev: z]\n",
			Row{Desc: "", Usage: "u [ev: in usage]", Exit: "e [ev: only tag] [ev: z]", Ev: "[ev: only tag] [ev: in usage] [ev: z]"}, ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, problem := Extract(tt.file, tt.text)
			tt.want.Name = tt.file
			if problem != tt.problem {
				t.Fatalf("problem = %q, want %q", problem, tt.problem)
			}
			if problem == "" && got != tt.want {
				t.Errorf("got %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func filler(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += "# filler\n"
	}
	return s
}

func TestInvokedByReport(t *testing.T) {
	t.Parallel()
	report := "#!/usr/bin/env bash\n# lint-beta.sh is documented here but never run.\n" +
		"a_out=$(\"$TOOLBELT/lint-alpha.sh\" \"$ADIR/src\" 2>&1) || a_exit=$?\n" +
		"echo \"run lint-gamma.sh manually\"\n  # \"$TOOLBELT/lint-gamma.sh\" indented comment\n" +
		"x=$(\"${TOOLBELT}/lint-d.sh\")\ny=$TOOLBELT/lint-e.sh\nz='$TOOLBELT/lint-q.sh'\r\nw=$TOOLBELT/lint-dot.sh\n" +
		"echo \"run $TOOLBELT/lint-m.sh now\"\nprintf '%s\\n' \"$TOOLBELT/lint-p.sh\"\n  \"$TOOLBELT/lint-s.sh\" \"$ADIR\"\n" +
		"if \"$TOOLBELT/lint-i.sh\" x; then :; fi\ntrue && \"$TOOLBELT/lint-and.sh\" x\nfind . | \"$TOOLBELT/lint-pipe.sh\"\n" +
		":; $TOOLBELT/lint-semi.sh\nbash \"$TOOLBELT/lint-bash.sh\" x\n"
	cases := []struct {
		name string
		want bool
	}{
		{"lint-alpha.sh", true}, {"lint-d.sh", true},
		{"lint-e.sh", false}, // an assignment, not an invocation
		{"lint-q.sh", false}, // a single-quoted literal
		{"lint-m.sh", false}, // R3-gli-auto-message-string-overclaim: a message string
		{"lint-p.sh", false}, // a printf argument
		{"lint-s.sh", true},  // a command at the start of a line
		{"lint-i.sh", true},  // after if
		{"lint-and.sh", true}, {"lint-pipe.sh", true}, {"lint-semi.sh", true}, {"lint-bash.sh", true},
		{"lint-beta.sh", false},  // GLI-auto: a comment
		{"lint-gamma.sh", false}, // a message string and an indented comment
		{"lint-al.sh", false},    // a prefix of an invoked name
		{"lint-dotXsh", false},   // the dot is literal
	}
	for _, tt := range cases {
		if got := InvokedByReport(report, tt.name); got != tt.want {
			t.Errorf("InvokedByReport(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestRenderRow(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		row  Row
		want string
	}{
		{"one usage, no evidence", Row{Name: "lint-a.sh", Desc: "d", Usage: "u", Exit: "e"}, "| `toolbelt/lint-a.sh` | d | `u` | e | no | — |"},
		{"GLI-multi: one code span per usage", Row{Name: "lint-b.sh", Desc: "d", Usage: "u1<br>u2", Exit: "e", Auto: true, Ev: "[ev: x]"},
			"| `toolbelt/lint-b.sh` | d | `u1`<br>`u2` | e | yes | [ev: x] |"},
		{"a usage already in backticks is kept", Row{Name: "lint-c.sh", Desc: "d", Usage: "`u` x<br>y", Exit: "e"}, "| `toolbelt/lint-c.sh` | d | `u` x<br>y | e | no | — |"},
	}
	for _, tt := range cases {
		if got := tt.row.Markdown(); got != tt.want {
			t.Errorf("%s:\ngot  %q\nwant %q", tt.name, got, tt.want)
		}
	}
}
