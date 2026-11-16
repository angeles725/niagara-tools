// Package labsim is the lab-sim command: a field-device simulator the
// laboratory controls and scripts from tests. It only composes its parts: the
// scenario file (scenario), the protocol-neutral devices (model), the Modbus
// register map, codec and server (regmap, modbus, mbserver), the request log
// (reqlog) and the loopback control port (control).
package labsim

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

//go:embed usage.txt
var usage string

// command is one sub-command: the options it takes (each with a value), how
// many arguments, and what runs.
type command struct {
	options  []string
	min, max int
	run      func(p parsed, s cli.Stdio) (int, error)
}

// commands is the table of sub-commands. A run returns exitUsage with the
// error to print for a command line it cannot accept.
var commands = map[string]command{
	"run":    {options: []string{"listen", "control", "log"}, min: 1, max: 1, run: runCommand},
	"check":  {min: 1, max: 1, run: checkCommand},
	"get":    {options: []string{"control"}, min: 2, max: 2, run: getCommand},
	"set":    {options: []string{"control"}, min: 3, max: 3, run: setCommand},
	"status": {options: []string{"control"}, run: statusCommand},
	"stop":   {options: []string{"control"}, run: stopCommand},
	"probe":  {options: []string{"addr", "unit", "fn", "address", "count", "values", "timeout-ms"}, run: probeCommand},
}

// Main is the lab-sim entry point.
func Main(args []string, s cli.Stdio) int {
	if len(args) == 0 {
		fmt.Fprint(s.Err, msgNoCommand)
		return exitUsage
	}
	name := args[0]
	if name == "help" || name == "-h" || name == "--help" {
		fmt.Fprint(s.Out, usage)
		return exitOK
	}
	cmd, ok := commands[name]
	if !ok {
		fmt.Fprintf(s.Err, msgUnknownCommand, name)
		return exitUsage
	}
	p, err := parseArgs(args[1:], cmd.options)
	if err == nil && (len(p.rest) < cmd.min || len(p.rest) > cmd.max) {
		err = fmt.Errorf(msgArgCount)
	}
	if err != nil {
		fmt.Fprintf(s.Err, msgUsage, name, err)
		return exitUsage
	}
	code, err := cmd.run(p, s)
	switch {
	case err != nil && code == exitUsage:
		fmt.Fprintf(s.Err, msgUsage, name, err)
	case err != nil:
		fmt.Fprintf(s.Err, msgFailed, err)
	}
	return code
}

// parsed is a command line split into its options and its arguments.
type parsed struct {
	options map[string]string
	rest    []string
}

// option is the value of an option, or fallback when it was not given.
func (p parsed) option(name, fallback string) string {
	if v, ok := p.options[name]; ok {
		return v
	}
	return fallback
}

// parseArgs reads "--name value" and "--name=value" for the allowed names,
// anywhere on the line; everything else is an argument. Only a leading "--"
// marks an option, so a negative number is an argument.
func parseArgs(args []string, allowed []string) (parsed, error) {
	p := parsed{options: map[string]string{}}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			p.rest = append(p.rest, arg)
			continue
		}
		name, value, inline := strings.Cut(arg[2:], "=")
		known := false
		for _, a := range allowed {
			known = known || a == name
		}
		if !known {
			return p, fmt.Errorf(msgUnknownFlag, arg)
		}
		if !inline {
			if i+1 >= len(args) {
				return p, fmt.Errorf(msgFlagValue, arg)
			}
			i++
			value = args[i]
		}
		p.options[name] = value
	}
	return p, nil
}
