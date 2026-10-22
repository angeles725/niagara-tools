package cisteps

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unicode"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// ExitBad is the exit code for a usage error, an unreadable file or an unreadable form; the
// records are written only when the whole file parsed.
const ExitBad = 2

// Main implements `ci-steps <workflow.yml>`: one record per step on stdout, three NUL-terminated
// fields <kind> <name> <body>, read by scripts/ci-local.sh.
func Main(args []string, s cli.Stdio) int {
	if len(args) != 1 {
		fmt.Fprint(s.Err, "usage: ci-steps <workflow.yml>\n")
		return ExitBad
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintln(s.Err, osError(args[0], err))
		return ExitBad
	}
	steps, err := Parse(args[0], data)
	if err != nil {
		fmt.Fprintln(s.Err, err)
		return ExitBad
	}
	for _, st := range steps {
		fmt.Fprintf(s.Out, "%s\x00%s\x00%s\x00", st.Kind, st.Name, st.Body)
	}
	return 0
}

// osError formats a read error as Python's OSError does (`[Errno 2] No such file or directory: 'p'`),
// so the message ci-local.sh shows did not change with the port.
func osError(path string, err error) string {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return err.Error()
	}
	msg := []rune(errno.Error())
	msg[0] = unicode.ToUpper(msg[0])
	return fmt.Sprintf("[Errno %d] %s: '%s'", int(errno), string(msg), path)
}
