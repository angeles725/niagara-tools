package platread

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// What station-pull (issue #1169) reuses: the env-file credential, the masked command line, the plat
// executable lookup, the output cleaner and the Java-exception detector.

// Masked is what replaces the password in anything printed.
const Masked = masked

// Cred is the daemon credential read from the env file.
type Cred = cred

// LoadEnv reads and checks the mode-0600 env file at path.
func LoadEnv(path string) (Cred, error) { return loadEnv(path) }

// DefaultEnv is NIAGARA_PLAT_ENV, else ~/.config/niagara-private/lab-platform.env ("" when no home).
func DefaultEnv(s cli.Stdio) string {
	if p := s.Env("NIAGARA_PLAT_ENV"); p != "" {
		return p
	}
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".config", "niagara-private", "lab-platform.env")
	}
	return ""
}

// Conn is the connection part of a plat command line; shown masks the password.
func (c cred) Conn(shown bool) []string {
	pwd := c.password
	if shown {
		pwd = masked
	}
	return []string{"-h:ip:" + c.host, "-p:" + c.port, "-secure", "-noinput", "-usr:" + c.user, "-pwd:" + pwd}
}

// Mask replaces the password in s.
func (c cred) Mask(s string) string { return strings.ReplaceAll(s, c.password, masked) }

// PlatExe is <home>/bin/plat.exe, else <home>/bin/plat.
func PlatExe(home string) string { return call{home: home}.exe() }

// Failure names why plat failed when it exited 0 with a stack trace, or "" when it printed no exception.
func Failure(output string) string { return failure(output) }

// Clean drops the CRs and the NRE boot noise from plat output.
func Clean(output string) string {
	var kept []string
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r", ""), "\n") {
		if !noise.MatchString(line) {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}
