package reportmodule

import (
	"fmt"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/atomicfile"
)

// writeBaseline is --write-baseline, after the summary: it replaces <root>/.report-baseline with the keys of
// the run's FAIL rows, sorted and each once, floor check rows left out (and named on stderr), and returns the
// exit code of the run; a file that cannot be written is an env fault. Without the option it returns code.
func (r *report) writeBaseline(code int) int {
	if !r.o.writeBaseline {
		return code
	}
	b := r.bl
	keys := append([]string(nil), b.keys...)
	sort.Strings(keys)
	var body strings.Builder
	body.WriteString(baselineHeader)
	n := 0
	for i, key := range keys {
		if i == 0 || key != keys[i-1] {
			body.WriteString(key + "\n")
			n++
		}
	}
	if err := atomicfile.Write(b.path, []byte(body.String())); err != nil {
		fmt.Fprintf(r.errw, msgCannotWrite, b.path, err)
		return ExitEnv
	}
	what := counted(n, "FAIL key")
	if b.floorN > 0 {
		what += fmt.Sprintf(msgFloorLeftOut, counted(b.floorN, "floor check row"))
	}
	fmt.Fprintf(r.errw, msgWrote, b.path, what)
	return code
}

// counted is "1 <what>" or "<n> <what>s".
func counted(n int, what string) string {
	if n == 1 {
		return "1 " + what
	}
	return itoa(n) + " " + what + "s"
}
