package verifymodule

import (
	"fmt"
	"strings"
)

// proceed is parseFlags' code when the run goes on.
const proceed = -1

// valueFlags are the gate's flags that take a value, each with the field of the run it sets.
func (r *run) valueFlags() map[string]*string {
	return map[string]*string{"--target-version": &r.target, "--src": &r.src, "--expected-signer": &r.expected}
}

// parseFlags reads the gate's command line into r and returns the jars. code is proceed, or the exit code
// after the usage or a message was printed. Flags and jars may come in any order.
func (r *run) parseFlags(args []string) (jars []string, code int) {
	values := r.valueFlags()
	for len(args) > 0 {
		switch a := args[0]; {
		case values[a] != nil:
			if len(args) < 2 {
				fmt.Fprint(r.s.Err, usage)
				return nil, 2
			}
			if a == "--expected-signer" && args[1] == "" {
				// An empty alias (an unset variable of the caller) must not turn the comparison off.
				fmt.Fprint(r.s.Err, msgNoAlias, usage)
				return nil, 2
			}
			*values[a] = args[1]
			args = args[1:]
		case a == "--stored":
			r.stored = true
		case a == "--strict":
			r.strict = true
		case a == "-h" || a == "--help":
			fmt.Fprint(r.s.Out, usage)
			return nil, 0
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(r.s.Err, "verify-module: unknown flag %s\n%s", a, usage)
			return nil, 2
		default:
			jars = append(jars, a)
		}
		args = args[1:]
	}
	return jars, proceed
}
