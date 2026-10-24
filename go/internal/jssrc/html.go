package jssrc

import (
	"regexp"
	"strings"
)

var (
	scriptOpen  = regexp.MustCompile(`(?i)<script(\s[^>]*)?>`)
	scriptClose = regexp.MustCompile(`(?i)</script\s*>`)
)

// HTMLScripts is the lines of an HTML file with every byte outside its <script ...>...</script> elements
// replaced by a blank (line breaks kept), so Code or Tokens read only the page's JavaScript and each line keeps
// its length. A <script> left open runs to the end of the file. The tags themselves are blanked too.
func HTMLScripts(lines []string) []string {
	src := strings.Join(lines, "\n")
	b := []byte(src)
	keep := make([]bool, len(b))
	for at := 0; at < len(src); {
		open := scriptOpen.FindStringIndex(src[at:])
		if open == nil {
			break
		}
		from := at + open[1]
		to := len(src)
		if shut := scriptClose.FindStringIndex(src[from:]); shut != nil {
			to = from + shut[0]
			at = from + shut[1]
		} else {
			at = len(src)
		}
		for k := from; k < to; k++ {
			keep[k] = true
		}
	}
	for k := range b {
		if !keep[k] && b[k] != '\n' {
			b[k] = ' '
		}
	}
	return strings.Split(string(b), "\n")
}
