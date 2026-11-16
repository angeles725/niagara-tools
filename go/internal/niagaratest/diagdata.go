package niagaratest

// The run log's bounds: every read is a tail of lines, each line cut at diagLineCap bytes, and a file is read
// from at most its last diagWindow bytes; no buffer is ever sized from a file's size.
const (
	diagRowLines    = 500
	diagStderrLines = 200
	diagStdoutLines = 50
	diagLineCap     = 4 << 10
	diagWindow      = 8 << 20
	diagMaxEntries  = 200
)

// evidenceCap bounds each file a failed run keeps beside its log: the testng-results.xml copy, the traces and the
// stderr copy.
const evidenceCap = 4 << 20

// exitNames spell the run's exit codes in the log.
var exitNames = map[int]string{exitPass: "PASS", exitFail: "FAIL", exitUsage: "usage or refused", exitEnv: "environment", exitTimeout: "timeout"}
