package commissioningverify

import (
	"fmt"
	"regexp"
	"strings"
)

// owedCols is the width of the values-owed table: Slot | Owed by | Unit | Safe default | Status. One constant for
// the malformed-row rule and the wide-block-after-the-Slot-table rule, so the two stay in step.
const owedCols = 5

// sepCell is one cell of a markdown alignment row: ---, :---, :---:, ---:.
var sepCell = regexp.MustCompile(`^:?-+:?$`)

// Block states of the values-owed reader.
const (
	outside   = iota // between pipe blocks
	owedBlock        // rows are owed rows
	foreign          // a foreign table (revisions, notes): skipped
	pending          // first row of a block buffered: the second row decides
)

// owedRows returns one MANUAL detail per open row of the values-owed doc text (file names the doc in the
// no-table row), in file order; none means every value is provided. It fails closed (toolbelt/commissioning-verify.sh
// header): the values-owed table is the block whose header row starts with Slot; another block is foreign only
// when its second row is an alignment row and it is not a block of owedCols or more columns after the Slot
// table; a row with fewer than owedCols cells or an empty Slot cell is a "malformed" row.
func owedRows(file, text string) []string {
	var out []string
	st, pend, pendN, seen, pipes := outside, "", 0, false, false
	owed := func(r string) {
		c, blank, sep := cells(r)
		switch {
		case blank || sep:
		case len(c) < owedCols || c[0] == "":
			out = append(out, "malformed values-owed row (needs Slot | Owed by | Unit | Safe default | Status): "+trim(r))
		case strings.HasPrefix(lower(c[4]), "filled") || strings.HasPrefix(lower(c[4]), "provided"):
		default:
			out = append(out, fmt.Sprintf("%s: owed by %s, unit %s, still at safe default %s", c[0], c[1], c[2], c[3]))
		}
	}
	flush := func() { // a lone buffered row is an owed row
		if pend != "" {
			owed(pend)
		}
		pend = ""
	}
	for _, r := range records(text) {
		if !strings.HasPrefix(strings.TrimLeft(r, " \t"), "|") {
			flush()
			st = outside
			continue
		}
		pipes = true
		c, _, sep := cells(r)
		switch st {
		case outside: // first row of a block
			if len(c) > 0 && lower(c[0]) == "slot" {
				st, seen = owedBlock, true
				continue
			}
			pend, pendN, st = r, len(c), pending
			continue
		case pending:
			if sep && !(seen && pendN >= owedCols) {
				pend, st = "", foreign
				continue
			}
			flush() // headerless or wide block after the Slot table: owed rows
			st = owedBlock
		}
		if st == owedBlock {
			owed(r)
		}
	}
	flush()
	if pipes && !seen {
		out = append(out, "no values-owed table (header Slot | Owed by | Unit | Safe default | Status) in "+file)
	}
	return out
}

// records splits text into awk records: lines without their "\n", a last line without one included.
func records(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

// cells splits a pipe row like the awk reader: trimmed, one outer pipe dropped at each end, each cell trimmed with
// its backticks removed. blank means every cell is empty, sep that every cell is an alignment cell (both hold
// for a row with no cell).
func cells(r string) (c []string, blank, sep bool) {
	line := strings.TrimSuffix(strings.TrimPrefix(trim(r), "|"), "|")
	if line == "" {
		return nil, true, true
	}
	c, blank, sep = strings.Split(line, "|"), true, true
	for i := range c {
		c[i] = strings.ReplaceAll(trim(c[i]), "`", "")
		blank = blank && c[i] == ""
		sep = sep && sepCell.MatchString(c[i])
	}
	return c, blank, sep
}

func trim(s string) string { return strings.Trim(s, " \t") }

// lower is awk tolower under LC_ALL=C: ASCII letters only.
func lower(s string) string {
	b := []byte(s)
	for i, ch := range b {
		if 'A' <= ch && ch <= 'Z' {
			b[i] = ch + 'a' - 'A'
		}
	}
	return string(b)
}
