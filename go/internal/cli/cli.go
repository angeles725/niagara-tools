// Package cli holds what every niagara-tools command shares: injectable stdio and
// the process entry point (modelled on niagara-go's internal/cli, reduced to what
// this module uses).
package cli

import (
	"io"
	"os"
)

// Stdio is the process output a command writes to and the environment it reads;
// tests pass buffers and a Getenv of their own so they never touch the process.
type Stdio struct {
	Out io.Writer
	Err io.Writer
	// Getenv reads one environment variable; nil reads the process (os.Getenv).
	Getenv func(name string) string
}

// Env is the value of the environment variable name ("" when unset).
func (s Stdio) Env(name string) string {
	if s.Getenv == nil {
		return os.Getenv(name)
	}
	return s.Getenv(name)
}

// Main is the signature every command package exports; it returns the exit code.
type Main func(args []string, s Stdio) int

// Run executes m with the process arguments and streams, then exits with its code.
func Run(m Main) {
	os.Exit(m(os.Args[1:], Stdio{Out: os.Stdout, Err: os.Stderr, Getenv: os.Getenv}))
}
