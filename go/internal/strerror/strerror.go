// Package strerror words an OS error as the C library's strerror() does in the C locale, the text GNU
// find and awk print after a path. It is shared by the lint engine and jarversion, which sit side by
// side and must not import each other.
package strerror

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
)

// Text is the C-locale strerror text of err's errno (Go's errno strings are glibc's, lower-cased; the
// first letter is upper-cased back), else err's own text unchanged.
func Text(err error) string {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return err.Error()
	}
	s := errno.Error()
	if s != "" && s[0] >= 'a' && s[0] <= 'z' {
		s = string(s[0]-'a'+'A') + s[1:]
	}
	return s
}

// FindQuote spells a path as GNU find quotes it in the C locale (quotearg locale style): always inside
// '...', with \' and \\, C escapes for \a \b \f \n \r \t \v, and 3-digit octal for every other byte
// outside printable ASCII (UTF-8 too: each byte is escaped on its own).
func FindQuote(p string) string {
	var b strings.Builder
	b.WriteByte('\'')
	for i := 0; i < len(p); i++ {
		c := p[i]
		switch esc := strings.IndexByte("\a\b\f\n\r\t\v", c); {
		case c == '\'' || c == '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		case esc >= 0:
			b.WriteByte('\\')
			b.WriteByte("abfnrtv"[esc])
		case c < 0x20 || c >= 0x7f:
			fmt.Fprintf(&b, "\\%03o", c)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('\'')
	return b.String()
}
