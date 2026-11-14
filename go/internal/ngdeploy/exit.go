package ngdeploy

// Exit codes of ng-deploy.sh; usage.txt lists them for the operator. The bash tool had all but 21 and 41.
const (
	ExitOK             = 0  // deployed, built with --no-deploy, or --help / --version
	ExitFault          = 1  // the backup directory could not be made (mkdir's own status, usually 1)
	ExitEnv            = 10 // a required variable, an invalid path, option value or env file
	ExitNoBaseline     = 12 // schema-risk precheck impossible: no baseline and no --first-deploy
	ExitSlotomatic     = 15 // slotomatic failed or skipped a file, or annotation changes in strict mode
	ExitBackup         = 20 // the backup failed
	ExitPurge          = 21 // the backup was written, but an older backup could not be purged
	ExitBuild          = 30 // gradlew failed
	ExitCopy           = 40 // a jar could not be copied to the station
	ExitRecord         = 41 // the jars were copied, but the deployed commit could not be read or recorded
	ExitVerify         = 50 // type count, a module.xml that cannot be read, cache-buster or verify-module gate
	ExitVersionSame    = 51 // changed jar bytes under an unchanged vendorVersion
	ExitVersionUnknown = 52 // the version-bump precheck cannot verify, or the build left no jar
	ExitSchema         = 53 // schema-risk OUTAGE, LOSSY without --accept-lossy, or a schema-risk fault
	ExitVersionDown    = 54 // the built vendorVersion is lower than the deployed one
)

// proceed is what a phase returns when the deploy goes on to the next one.
const proceed = -1

// Exit statuses of the tools the deploy runs.
const (
	schemaLossy   = 1   // schema-risk.sh: a saved value would be lost
	schemaOutage  = 2   // schema-risk.sh: the slot change breaks saved data
	notExecutable = 126 // what bash returns for a command it cannot execute
	notFound      = 127 // what bash returns for a command that does not exist
	signalBase    = 128 // bash: 128 + the signal that killed the command
)
