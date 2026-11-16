package mergewait

import "fmt"

// failure reports the failed checks with their log tails. When none was re-run yet and every one failed
// on a transient download error, it re-runs their failed jobs once, marks them in rerun and returns
// ExitMerged so the wait goes on; otherwise it returns ExitFailed (or ExitGh when the re-run fails).
func (c Config) failure(bad []checkRun, rerun map[int64]bool) int {
	transient := len(rerun) == 0
	runs := map[string]bool{}
	causes := map[int64]string{}
	for _, r := range bad {
		fmt.Fprintf(c.Err, msgFailed, r.Name, r.Conclusion, r.DetailsURL)
		lines := c.logTail(r)
		cause := transientCause(lines)
		run, _ := runAndJob(r)
		if cause == "" || run == "" {
			transient = false
		}
		causes[r.ID] = cause
		runs[run] = true
	}
	if !transient {
		return ExitFailed
	}
	for _, r := range bad {
		fmt.Fprintf(c.Err, msgRerun, r.Name, causes[r.ID])
		rerun[r.ID] = true
	}
	for run := range runs {
		if _, err := c.Gh("run", "rerun", run, "--failed"); err != nil {
			return c.ghFailed(err)
		}
	}
	return ExitMerged
}

// logTail prints and returns the last LogLines lines of the failed steps of an Actions job. Reading
// the log is itself a download: one failed attempt is retried once.
func (c Config) logTail(r checkRun) []string {
	_, job := runAndJob(r)
	if job == "" {
		fmt.Fprintf(c.Err, msgNoLog, "not a GitHub Actions job")
		return nil
	}
	out, err := c.Gh("run", "view", "--job", job, "--log-failed")
	if err != nil {
		out, err = c.Gh("run", "view", "--job", job, "--log-failed")
	}
	if err != nil {
		fmt.Fprintf(c.Err, msgNoLog, err)
		return nil
	}
	lines := tail(out, c.LogLines)
	fmt.Fprintf(c.Err, msgLogTail, len(lines))
	for _, l := range lines {
		fmt.Fprintln(c.Err, l)
	}
	return lines
}
