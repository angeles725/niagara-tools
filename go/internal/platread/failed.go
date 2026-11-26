package platread

import (
	"regexp"
	"strings"
)

// javaThrow is the first line of a Java stack trace (a class name ending in Exception or Error).
// Live (4.14.0.162): plat exits 0 after printing javax.baja.security.AuthenticationException for a
// wrong daemon password, and after a NullPointerException, so its exit code alone does not tell a
// failure from an answer.
var javaThrow = regexp.MustCompile(`(?m)^([a-z][\w$]*\.)+[A-Z][\w$]*(Exception|Error)\b`)

// failure names why plat failed when it exited 0 with a stack trace, or "" when it answered.
func failure(answer string) string {
	thrown := javaThrow.FindString(answer)
	switch {
	case thrown == "":
		return ""
	case strings.HasSuffix(thrown, "AuthenticationException"):
		return "the daemon refused the user or password of the env file (" + thrown + ")"
	default:
		return "plat failed: " + thrown
	}
}
