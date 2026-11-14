package buildmodule

// Exit codes of build.sh (unchanged from the bash tool; its header lists them).
const (
	ExitOK            = 0  // the whole chain passed
	ExitFault         = 1  // a step the chain cannot run without failed (temporary directory, baseline copy)
	ExitUsage         = 2  // usage
	ExitEnv           = 10 // environment: preflight FAIL, no JDK 8, not a niagara_home, a real one without the opt-in, no profile
	ExitGradle        = 30 // gradle failed
	ExitCleanLocked   = 31 // :clean could not delete a locked modules/<jar>
	ExitCopyLocked    = 32 // the plugin's post-jar copy into modules/ was locked
	ExitWhitespace    = 33 // the mirror path or $HOME holds whitespace
	ExitSlotomatic    = 34 // Slot-o-Matic skipped a file in a passing gradle run
	ExitGate          = 50 // the verify gate or report-module FAILed
	ExitDrift         = 51 // deployed-baseline drift
	ExitDriftUnknown  = 52 // the drift gate cannot verify
	ExitReportPartial = 53 // report-module INCOMPLETE (a floor check SKIPped)
)

// proceed is what a step returns when the chain goes on to the next one.
const proceed = -1

// Exit statuses of the tools the chain runs.
const (
	preflightFail    = 1   // preflight.sh: at least one FAIL row
	reportEnvFault   = 3   // report-module.sh: environment error
	reportIncomplete = 4   // report-module.sh: verdict INCOMPLETE
	notExecutable    = 126 // what bash returns for a command it cannot execute
	notFound         = 127 // what bash returns for a command that does not exist
	signalBase       = 128 // bash: 128 + the signal that killed the command
)
