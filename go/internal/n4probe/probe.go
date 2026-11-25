// Package n4probe is the n4-probe command: live probes of a station's HTTP API (`stall`, `pulse`).
// Every probe is capped at 4 Hz, keeps one request in flight, runs for a fixed time and reads its
// credentials only from the operator env file (never from a flag or the process environment).
package n4probe

import (
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/url"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/envfile"
)

// Exit codes of n4-probe.
const (
	ExitOK          = 0
	ExitFailed      = 1 // every stall request failed; the pulse press or baseline could not be sent
	ExitUsage       = 2
	ExitNotReleased = 3 // pulse: not back at the baseline within the duration
)

const (
	maxRate     = 4.0
	maxDuration = 10 * time.Minute
	keyUser     = "N4_PROBE_USER"
	keyPass     = "N4_PROBE_PASSWORD"
)

//go:embed usage.txt
var usage string

// common is the flags every subcommand shares.
type common struct {
	envFile    string
	insecure   bool
	rate       float64
	allowAbove bool
	duration   time.Duration
}

func (c *common) register(fs *flag.FlagSet) {
	fs.StringVar(&c.envFile, "env-file", "", "env file with the credentials")
	fs.BoolVar(&c.insecure, "insecure-tls", false, "accept a self-signed certificate")
	fs.Float64Var(&c.rate, "rate", maxRate, "requests per second")
	fs.BoolVar(&c.allowAbove, "allow-rate-above-4", false, "lift the 4 Hz cap")
	fs.DurationVar(&c.duration, "duration", 30*time.Second, "run length")
}

// Main implements `n4-probe stall|pulse ...`.
func Main(args []string, s cli.Stdio) int {
	if len(args) == 0 {
		fmt.Fprint(s.Err, usage)
		return ExitUsage
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprint(s.Out, usage)
		return ExitOK
	case "stall":
		return runStall(args[1:], s)
	case "pulse":
		return runPulse(args[1:], s)
	}
	return usageErr(s, "unknown subcommand %q (stall or pulse)", args[0])
}

func usageErr(s cli.Stdio, format string, a ...any) int {
	fmt.Fprintf(s.Err, "n4-probe: "+format+"\nsee n4-probe --help\n", a...)
	return ExitUsage
}

// parse reads args into fs; done is true when the command must exit with code now.
func parse(fs *flag.FlagSet, args []string, s cli.Stdio) (code int, done bool) {
	fs.SetOutput(io.Discard)
	switch err := fs.Parse(args); {
	case errors.Is(err, flag.ErrHelp):
		fmt.Fprint(s.Out, usage)
		return ExitOK, true
	case err != nil:
		return usageErr(s, "%v", err), true
	case fs.NArg() > 0:
		return usageErr(s, "unexpected argument %q", fs.Arg(0)), true
	}
	return 0, false
}

// client checks the common flags and builds the client with the env file's credentials.
func (c *common) client() (*client, error) {
	switch {
	case math.IsNaN(c.rate) || c.rate <= 0:
		return nil, errors.New("--rate must be greater than 0")
	case c.rate > maxRate && !c.allowAbove:
		return nil, fmt.Errorf("--rate %g is above the 4 Hz cap (--allow-rate-above-4 lifts it)", c.rate)
	case c.duration <= 0:
		return nil, errors.New("--duration must be greater than 0")
	case c.duration > maxDuration:
		return nil, fmt.Errorf("--duration %s is above the %s cap", c.duration, maxDuration)
	case c.envFile == "":
		return nil, errors.New("--env-file is required: credentials are read only from that file")
	}
	user, pass, err := credentials(c.envFile)
	if err != nil {
		return nil, err
	}
	return newClient(user, pass, c.insecure), nil
}

// credentials reads N4_PROBE_USER and N4_PROBE_PASSWORD from the env file (the later assignment wins, as a
// sourced file); the process environment is not consulted and no value is ever put in an error.
func credentials(path string) (user, pass string, err error) {
	none := func(string) (string, bool) { return "", false }
	entries, found, err := envfile.Load(path, none)
	if err != nil {
		return "", "", fmt.Errorf("env file: %w", err)
	}
	if !found {
		return "", "", errors.New("env file not found")
	}
	for _, e := range entries {
		switch e.Key {
		case keyUser:
			user = e.Value
		case keyPass:
			pass = e.Value
		}
	}
	if user == "" || pass == "" {
		return "", "", fmt.Errorf("the env file must set %s and %s", keyUser, keyPass)
	}
	return user, pass, nil
}

// checkURL is why raw cannot be probed. It never echoes raw: a URL may carry a secret.
func checkURL(flagName, raw string) error {
	u, err := url.Parse(raw)
	switch {
	case raw == "":
		return fmt.Errorf("%s is required", flagName)
	case err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "":
		return fmt.Errorf("%s needs an http:// or https:// URL with a host", flagName)
	case u.User != nil:
		return fmt.Errorf("%s must not carry credentials (user:password@); they come from --env-file", flagName)
	}
	return nil
}

// label is the URL without its query, for output.
func label(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "?"
	}
	return u.Scheme + "://" + u.Host + u.Path
}
