package retrorows

import (
	"fmt"
	"strings"
)

// AppendIssue appends " #N" to the delta cell of the row on the given 1-based line and returns the new text. The line
// count, the other cells and a CRLF ending stay as they are. A row that already carries the same number is a no-op; a
// line that is not a delta row, or a row that carries another number, is an error.
func AppendIssue(data string, line, n int) (string, error) {
	lines := strings.Split(data, "\n")
	if line < 1 || line > len(lines) {
		return "", fmt.Errorf("line %d is outside the retro", line)
	}
	var row *Row
	for _, r := range Parse(data) {
		if r.Line == line {
			r := r
			row = &r
		}
	}
	if row == nil {
		return "", fmt.Errorf("line %d is not a delta row", line)
	}
	if row.Issue == n {
		return data, nil
	}
	if row.Issue != 0 {
		return "", fmt.Errorf("delta %s already carries #%d, not #%d", row.ID, row.Issue, n)
	}
	raw := lines[line-1]
	cr := ""
	if strings.HasSuffix(raw, "\r") {
		raw, cr = strings.TrimSuffix(raw, "\r"), "\r"
	}
	// byte offsets of the unescaped pipes
	var pipes []int
	for i := 0; i < len(raw); i++ {
		if raw[i] == '\\' {
			i++
		} else if raw[i] == '|' {
			pipes = append(pipes, i)
		}
	}
	if len(pipes) < 3 {
		return "", fmt.Errorf("line %d has no delta cell", line)
	}
	cellText := strings.TrimRight(raw[pipes[1]+1:pipes[2]], " ")
	lines[line-1] = raw[:pipes[1]+1] + cellText + fmt.Sprintf(" #%d ", n) + raw[pipes[2]:] + cr
	return strings.Join(lines, "\n"), nil
}
