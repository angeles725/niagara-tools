package lintheader

import (
	"reflect"
	"strings"
	"testing"
)

func TestLines(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, text string
		want       []string
	}{
		{"shebang and comments up to the first code line", "#!/bin/sh\n# a\n#\n# b\nset -u\n# after\n", []string{"#!/bin/sh", "# a", "#", "# b"}},
		{"no line cap", "#!/bin/sh\n" + strings.Repeat("# f\n", 70) + "x\n", append([]string{"#!/bin/sh"}, strings.Split(strings.Repeat("# f\n", 70), "\n")[:70]...)},
		{"blank lines stay in the header", "#!/bin/sh\n# a\n\n \t\r\n# b\nx\n", []string{"#!/bin/sh", "# a", "", " \t\r", "# b"}},
		{"an indented comment ends the header", "#!/bin/sh\n  # a\n", []string{"#!/bin/sh"}},
		{"CRLF lines keep their CR", "#!/bin/sh\r\n# a\r\nx\r\n", []string{"#!/bin/sh\r", "# a\r"}},
		{"an unterminated last line is read", "#!/bin/sh\n# a", []string{"#!/bin/sh", "# a"}},
		{"all comments", "# a\n# b\n", []string{"# a", "# b"}},
		{"first line is code", "set -u\n# a\n", nil},
		{"a line of spaces then code", "# a\n  x\n", []string{"# a"}},
		{"empty file", "", nil},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Lines(tt.text); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
