package mergewait

import "regexp"

// transientPatterns are log lines of a download that failed for a reason outside the change: Maven
// Central's 429 to the runners (niagara-tools #681), a server error, a reset or timed-out connection, a
// name that did not resolve. A failed check whose log tail matches one is re-run once; data only.
var transientPatterns = []*regexp.Regexp{
	regexp.MustCompile(`returned error: (429|5[0-9][0-9])\b`),                                                     // curl -f
	regexp.MustCompile(`\b(429 Too Many Requests|50[234] (Bad Gateway|Service Unavailable|Gateway Time-?out))\b`), // wget, npm, go
	regexp.MustCompile(`(?i)connection (reset|timed out|refused)`),
	regexp.MustCompile(`(?i)could not resolve host`),
	regexp.MustCompile(`(?i)TLS handshake timeout`),
	regexp.MustCompile(`(?i)unexpected EOF`),
}

// networkPatterns are gh errors of a request that never reached GitHub (this host loses its route now
// and then: `dial tcp ...:443: connect: network is unreachable`), safe to send again; data only.
var networkPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)dial tcp`),
	regexp.MustCompile(`(?i)network is unreachable`),
	regexp.MustCompile(`(?i)no such host`),
	regexp.MustCompile(`(?i)TLS handshake timeout`),
	regexp.MustCompile(`(?i)error connecting to`),
	regexp.MustCompile(`: timed out after `), // ExecGh's deadline: a stalled connection
}

func isNetworkError(err error) bool {
	for _, p := range networkPatterns {
		if p.MatchString(err.Error()) {
			return true
		}
	}
	return false
}

// transientCause returns the first log line that names a transient download failure ("" when none).
func transientCause(lines []string) string {
	for _, l := range lines {
		for _, p := range transientPatterns {
			if p.MatchString(l) {
				return p.FindString(l)
			}
		}
	}
	return ""
}
