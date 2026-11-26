package stationpull

import (
	"regexp"
	"strconv"
	"strings"
)

// Row is one `plat flist` entry: the file name and its CRC32 (0 = the daemon did not compute one).
type Row struct {
	Name string
	CRC  uint32
}

var (
	crcField  = regexp.MustCompile(`^\d[\d,]*$`)
	columnGap = regexp.MustCompile(`\t| {2,}`)
	// a trailing column that is not part of a name: size, date, time, AM/PM.
	columnTok = regexp.MustCompile(`^([\d,.]+|\d{1,4}[-/.]\d{1,2}[-/.]\d{1,4}|\d{1,2}:\d{2}(:\d{2})?([.,]\d+)?|[AaPp][Mm])$`)
)

// ParseFlist reads the rows of a `plat flist -recurse` answer. The CRC is the last field of a row (digits
// with thousands commas). The name is the text before the first column gap (a tab or two spaces), so it
// may hold single spaces; a row with single-spaced columns loses its trailing size/date/time columns
// instead. Lines without a CRC (boot log, header, stack trace) and directory rows (name ending in /) are
// skipped.
func ParseFlist(text string) []Row {
	rows, _ := parseFlist(text)
	return rows
}

// parseFlist is ParseFlist plus the rows whose CRC field is digits but no CRC32 (too large): such a
// row is a listed file that cannot be checked, so the pull fails rather than skip it.
func parseFlist(text string) (rows []Row, bad []string) {
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r", ""), "\n") {
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)
		if len(fields) < 2 || !crcField.MatchString(fields[len(fields)-1]) {
			continue
		}
		v, err := strconv.ParseUint(strings.ReplaceAll(fields[len(fields)-1], ",", ""), 10, 32)
		name := rowName(line, fields)
		if err != nil {
			bad = append(bad, name)
			continue
		}
		if name == "" || strings.HasSuffix(name, "/") {
			continue
		}
		rows = append(rows, Row{Name: name, CRC: uint32(v)})
	}
	return rows, bad
}

func rowName(line string, fields []string) string {
	if parts := columnGap.Split(line, -1); len(parts) >= 2 {
		return strings.TrimSpace(parts[0])
	}
	n := len(fields)
	for n > 1 && columnTok.MatchString(fields[n-1]) {
		n--
	}
	return strings.Join(fields[:n], " ")
}
