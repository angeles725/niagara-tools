// Package orientguard is the BUILD-LOOP §0.a location guard of toolbelt/orient-guard.sh: a module root
// must sit under the one legal build prefix before anything is built or edited. The rules and their
// mutations are listed in the header of that script.
package orientguard

// Exit codes of orient-guard.sh (unchanged from the bash tool; its header lists them).
const (
	ExitOK    = 0 // a PASS row, or a WARN row under the override
	ExitFail  = 1 // a FAIL row
	ExitUsage = 3 // usage error, or no HOME to derive the default prefix from
)

// The environment the guard reads.
const (
	// PrefixEnv replaces the legal prefix. It is a test seam: a WARN row always says it is in use.
	PrefixEnv = "BUILD_N4_LEGAL_ROOT_PREFIX"
	// OverrideEnv set to exactly OverrideOn turns a FAIL into a WARN; any other value changes nothing.
	OverrideEnv = "BUILD_N4_CLIENTE_OVERRIDE"
	OverrideOn  = "1"
)

// legalSuffix is the legal prefix below $HOME.
const legalSuffix = "/modulos_niagara_n4/Cliente/"

// Row statuses.
const (
	pass = "PASS"
	fail = "FAIL"
	warn = "WARN"
)

// rowFormat is one output row: status, tool, detail (the row convention of preflight.sh).
const rowFormat = "%-4s  %-12s  %s\n"

const tool = "orient-guard"

// Texts: the row details are fmt formats, the other two go to stderr as they are.
const (
	msgUsage    = "usage: orient-guard.sh <module-root>\n"
	msgNoHome   = "orient-guard: HOME is not set; cannot derive the legal root\n"
	detailSeam  = "non-default legal root %s (test seam)"
	detailOther = "non-standard path: %s"
	detailFail  = "%s — must be under %s; set " + OverrideEnv + "=" + OverrideOn + " to override"
)
