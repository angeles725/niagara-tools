package niagaratest

import (
	"regexp"
	"time"
)

// Exit codes of run, beyond verdict's.
const (
	exitEnv     = 3
	exitTimeout = 4
)

const (
	defaultTimeout = 30 * time.Minute
	maxTimeout     = 24 * time.Hour
	cleanupTimeout = 2 * time.Minute
	defaultJDK8    = "/usr/lib/jvm/java-8-openjdk-amd64"
)

// copiedDirs are the install dirs the home copy takes whole; modules/ is copied jar by jar (copyJars).
var copiedDirs = []string{"bin", "etc", "jre", "lib", "security", "defaults"}

var (
	partRe    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*(:[A-Za-z0-9][A-Za-z0-9_.-]*)*$`)
	moduleRe  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	versionRe = regexp.MustCompile(`^[0-9]+(\.[0-9]+)+$`)
)
