//go:build !unix

package niagaratest

import "os/exec"

// ownGroup is a no-op off unix: the tool runs from WSL, and cancellation still kills the direct child.
func ownGroup(*exec.Cmd) {}
