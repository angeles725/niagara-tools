package changelog

import (
	"fmt"
	"os/exec"
	"strings"
)

// gitRunner runs one git command and returns its stdout; tests may swap it.
type gitRunner func(args ...string) (string, error)

func realGit(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if ee, ok := err.(*exec.ExitError); ok {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(ee.Stderr)))
	}
	return string(out), err
}
