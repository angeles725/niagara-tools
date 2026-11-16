package niagaratest

import (
	"strings"
	"time"
)

// runConfig is the parsed command line of run.
type runConfig struct {
	install, gradleRoot, part, module, scratch string
	jdk8, gradleUserHome, logDir               string
	reuseHome                                  string // --reuse-home: boot again on a dir a --keep-home run kept
	timeout                                    time.Duration
	keepHome, verifyLow, restoreRoot           bool
	excludeJars                                []string // --exclude-jar globs, in flag order
	allow                                      IgnoredAllowance
}

// plan is a validated run: the install, %TEMP%, %USERPROFILE%, the scratch parent and, once created, the
// scratch dir.
type plan struct {
	runConfig
	install, temp, profile, parent, dir winDir
	windows                             bool   // a Windows-side step started: cleanup must look for its processes
	sysPropsSum                         string // --module-verification low: sha256 of the install's system.properties
	excluded                            []jarExclusion
	root                                rootSnapshot // --gradle-root's git status before the copy
	rootTop                             string       // its worktree top, where the status paths start
	rows                                *lineRing    // the last rows the run printed, for the run log
	logDirSet                           bool         // --log-dir was given: a green run keeps a log too
	runID                               string       // this run's id, in its marker and its log (reusehome.go)
	marker                              marker       // the marker this run wrote in its scratch dir
	logPath                             string       // the run log once written
}

func (p *plan) home() winDir        { return winDir{p.dir.wsl + "/home", p.dir.win + `\home`} }
func (p *plan) sub(n string) string { return p.dir.wsl + "/" + n }

func (p *plan) subWin(n string) string {
	return p.dir.win + `\` + strings.ReplaceAll(n, "/", `\`)
}
