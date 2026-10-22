package foldaudit

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of sweep-fold-audit (unchanged from the bash tool).
const (
	ExitOK     = 0 // clean, or WARN-only without --strict
	ExitStrict = 1 // uncited rows (or uncited Δ) under --strict
	ExitEnv    = 3 // usage or environment error
)

// Arg0Env names the script path the wrapper was invoked as; it prefixes the two bash runtime
// diagnostics this port reproduces (an octal-invalid deltas cell, an unreadable INDEX).
const Arg0Env = "SWEEP_FOLD_AUDIT_ARG0"

const usage = "usage: sweep-fold-audit.sh [--strict] [--deltas-since <YYYY-MM-DD>] <INDEX.md> <kit-root>"

// Line numbers of the bash statements whose runtime errors are reproduced verbatim.
const (
	bashLineDeltaLoop = 132 // for ((k = 1; k <= ndelta; k++))
	bashLineIndexRead = 152 // done < "$INDEX"
)

// Main implements `sweep-fold-audit [--strict] [--deltas-since <YYYY-MM-DD>] <INDEX.md> <kit-root>`.
func Main(args []string, s cli.Stdio) int {
	o, usageErr := parseArgs(args)
	if usageErr != "" {
		fmt.Fprint(s.Err, usageErr)
		return ExitEnv
	}
	if st, err := os.Stat(o.index); err != nil || !st.Mode().IsRegular() {
		fmt.Fprintf(s.Err, "fold-audit: INDEX not found: %s\n", o.index)
		return ExitEnv
	}
	if st, err := os.Stat(o.root); err != nil || !st.IsDir() {
		fmt.Fprintf(s.Err, "fold-audit: kit-root not found: %s\n", o.root)
		return ExitEnv
	}
	arg0 := s.Env(Arg0Env)
	if arg0 == "" {
		arg0 = "sweep-fold-audit.sh"
	}

	h := harvestCorpus(o.root, o.since != "")
	tokens := h.sortedTokens()

	// Like the bash loop: an unreadable INDEX is reported and audited as empty.
	data, err := os.ReadFile(o.index)
	if err != nil {
		fmt.Fprintf(s.Err, "%s: line %d: %s: %s\n", arg0, bashLineIndexRead, o.index, strerror(err))
	}
	lines := strings.Split(string(data), "\n")
	lines = lines[:len(lines)-1] // `read` never runs the body for a last line without a newline

	var nFolded, nCited, nUncited, dChecked, dMissing int
	warned := false
	for _, line := range lines {
		line = strings.ReplaceAll(line, "\x00", "") // `read` drops NUL bytes
		if !strings.HasPrefix(line, "|") || !datedRow(line) {
			continue
		}
		file := rowFile(line)
		if file == "" || rowStatus(line) != "folded" {
			continue
		}
		nFolded++
		st := stem(file)
		hits := citedBy(st, tokens)

		if o.since != "" && !(prefix(file, 10) < o.since) {
			cell := deltasCell(line)
			if !allDigits(cell) {
				fmt.Fprintf(s.Out, "fold-audit: WARN %s deltas cell unreadable (%s)\n", file, cell)
				dMissing++
				warned = true
			} else if n, ok := bashInt(cell); !ok {
				fmt.Fprintf(s.Err, "%s: line %d: ((: %s: value too great for base (error token is \"%s\")\n",
					arg0, bashLineDeltaLoop, cell, cell)
			} else {
				for k := int64(1); k <= n; k++ {
					dChecked++
					// Mutation: F9 -- substring matching credits another stem's Δk, or Δ10 for Δ1.
					if !h.deltas[st+" Δ"+strconv.FormatInt(k, 10)] {
						fmt.Fprintf(s.Out, "fold-audit: WARN %s Δ%d not cited\n", file, k)
						dMissing++
						warned = true
					}
				}
			}
		}

		if len(hits) == 0 {
			fmt.Fprintf(s.Out, "fold-audit: WARN %s folded with no [ev: retro …] citation\n", file)
			nUncited++
			warned = true
		} else {
			nCited++
			if len(hits) > 1 {
				fmt.Fprintf(s.Out, "fold-audit: NOTE ambiguous citation token credits %s: %s\n", file, strings.Join(hits, " "))
			}
		}
	}

	fmt.Fprintf(s.Out, "fold-audit: %d folded, %d cited, %d uncited\n", nFolded, nCited, nUncited)
	if o.since != "" {
		fmt.Fprintf(s.Out, "fold-audit: deltas: %d checked, %d not cited (folded rows since %s)\n", dChecked, dMissing, o.since)
	}
	if warned && o.strict {
		return ExitStrict
	}
	return ExitOK
}

func prefix(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n]
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// strerror renders err as glibc's strerror does for bash ("Permission denied"): Go's errno text
// with its first letter capitalized.
func strerror(err error) string {
	var errno syscall.Errno
	msg := err.Error()
	if errors.As(err, &errno) {
		msg = errno.Error()
	}
	if msg == "" {
		return msg
	}
	return strings.ToUpper(msg[:1]) + msg[1:]
}
