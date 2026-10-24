package slotcoverage

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

var (
	propName = regexp.MustCompile(`name[[:space:]]*=[[:space:]]*"([^"]*)"`)
	rangeTag = grepLongest(`@Range\("[^"]+"\)`)
)

// xargsWords is how `printf '%s\n' "$files" | xargs` splits the file list: on blanks and newlines.
func xargsWords(files []string) []string {
	return strings.FieldsFunc(strings.Join(files, "\n"), func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' })
}

// slots runs the awk scan of the per-slot mode over words, one awk over all of them: an annotation open at
// the end of one file carries into the next, a directory is skipped, and the first word awk cannot open
// ends the scan (gawk's fatal "cannot open file"). all holds every named annotation, op the OPERATOR ones.
func slots(words []string) (all, op []string) {
	scan := javasrc.PropertyScan{}
	for _, w := range words {
		if isDir(w) {
			continue
		}
		data, err := os.ReadFile(w)
		if err != nil {
			break
		}
		for i, ln := range lines(data) {
			text, _, ok := scan.Line(ln, i+1)
			if !ok {
				continue
			}
			m := propName.FindStringSubmatch(text)
			if m == nil || m[1] == "" {
				continue
			}
			all = append(all, m[1])
			if strings.Contains(text, "OPERATOR") || strings.Contains(text, `"o"`) {
				op = append(op, m[1])
			}
		}
	}
	return all, op
}

// rangeTags are the @Range("tag") values grep -oh finds in words; a word grep cannot read is skipped.
func rangeTags(words []string) []string {
	var out []string
	for _, w := range words {
		if isDir(w) {
			continue
		}
		for _, m := range grepO(rangeTag, readQuiet(w)) {
			out = append(out, strings.TrimSuffix(strings.TrimPrefix(m, `@Range("`), `")`))
		}
	}
	return out
}

// filterDot is grep '\.' (dotted) or grep -v '\.' over the lines of v.
func filterDot(v string, dotted bool) string {
	var out []string
	for _, k := range here(v) {
		if strings.Contains(k, ".") == dotted {
			out = append(out, k)
		}
	}
	return subst(out)
}

// perSlot is the per-slot mode: OPERATOR @NiagaraProperty slots under src against bare lexicon keys.
func perSlot(xmlPath, lexPath, src string, s cli.Stdio) int {
	for _, p := range []string{xmlPath, lexPath} {
		if !isFile(p) {
			fmt.Fprintf(s.Err, "slot-coverage: not found: %s\n", p)
			return 3
		}
	}
	if !isDir(src) {
		fmt.Fprintf(s.Err, "slot-coverage: not a directory: %s\n", src)
		return 3
	}
	files, err := lint.Walk(src, lint.Selection{Names: []string{"*.java"}})
	if err != nil {
		fmt.Fprintf(s.Err, "slot-coverage: cannot list every file under %s: %s\n", src, err)
		return 3
	}
	words := xargsWords(files)
	allSlots, opSlots := slots(words)
	lexKeys := subst(sortU(trimKeys(lexiconKeys(grepRead(lexPath, 1, s)))))
	lexSlots, lexDotted := filterDot(lexKeys, false), filterDot(lexKeys, true)
	types := typeNames(grepRead(xmlPath, 1, s))
	known := sortU(nonEmpty(append(append(append([]string(nil), allSlots...), types...), rangeTags(words)...)))
	opSorted := sortU(opSlots)
	var missing, stale []string
	if lexSlots != "" {
		missing, stale = minus(opSorted, here(lexSlots)), minus(here(lexSlots), known)
	} else {
		missing = opSorted
	}
	missing, stale = nonEmpty(missing), nonEmpty(stale)
	if len(opSorted) == 0 {
		fmt.Fprintln(s.Out, "pct=N/A (per-slot)")
	} else {
		fmt.Fprintf(s.Out, "pct=%s (per-slot)\n", tenths(len(opSorted)-len(missing), len(opSorted)))
	}
	for _, k := range missing {
		fmt.Fprintf(s.Out, "MISSING %s\n", k)
	}
	for _, k := range stale {
		fmt.Fprintf(s.Out, "STALE %s\n", k)
	}
	if lexDotted != "" {
		for _, dk := range here(lexDotted) {
			ds := dk[strings.LastIndex(dk, ".")+1:]
			if !contains(here(subst(allSlots)), ds) || contains(here(lexSlots), ds) {
				continue
			}
			fmt.Fprintf(s.Out, "DOTTED %s (never read: the framework reads the bare key %s)\n", dk, ds)
		}
	}
	if len(missing) > 0 {
		return 1
	}
	return 0
}
