// Package kitresolve is the Go side of scripts/kit-resolve.sh: the fail-closed choice of the
// build-n4-module kit a session may run. The script header is the contract (candidates, the success
// line, the refusals, the kit-main fast-forward).
//
// One file per concern: layout.go (names, paths and exit codes, data), options.go (the command line),
// candidate.go (which kit wins), repo.go (the questions asked of git), command.go (the rules, in order).
package kitresolve

const (
	tool       = "kit-resolve"
	kitDirName = "build-n4-module-kit" // the kit inside a niagara-tools checkout
	kitMarker  = "METHODOLOGY.md"      // what makes a directory a kit

	envKit     = "BUILD_N4_KIT"      // candidate 1: a kit directory
	envKitMain = "BUILD_N4_KIT_MAIN" // overrides the kit-main worktree location

	// Under $HOME: the kit-main worktree (candidate 2) and the default checkout's kit (candidate 3).
	homeKitMain    = "/modulos_niagara_n4/niagara-tools-worktrees/kit-main"
	homeDefaultKit = "/modulos_niagara_n4/niagara-tools/" + kitDirName

	freshRef = "origin/main" // the ref a kit must be at
)

// Exit codes of kit-resolve (unchanged from the bash tool).
const (
	exitOK      = 0
	exitRefused = 2 // stale, dirty, ahead, diverged, not a git work tree, origin unreachable, unreadable
	exitUsage   = 3 // usage, or no candidate
)

// The candidate that won; it selects the remedy printed with a refusal.
const (
	sourceEnv     = "env"
	sourceKitMain = "kit-main"
	sourceDefault = "default"
)
