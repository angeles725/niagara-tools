package mergewait

import (
	"regexp"
	"strings"
)

// verdict is where the checks of one commit stand.
type verdict int

const (
	pending  verdict = iota // no check yet, or one not completed
	green                   // all completed, none failing, at least one success
	failed                  // all completed, at least one failing
	notGreen                // all completed, none failing, none succeeded (all skipped or neutral)
)

// passing are the conclusions that do not block a merge; every other conclusion of a completed run
// (failure, cancelled, timed_out, action_required, startup_failure, stale, or an unknown one) blocks.
var passing = map[string]bool{"success": true, "neutral": true, "skipped": true}

// judge reads the latest runs of a commit. It waits until every run completed, so a failed job's
// workflow is finished before it is reported or re-run (gh run rerun refuses a run in progress).
func judge(runs []checkRun) (v verdict, done int, bad []checkRun) {
	success := false
	for _, r := range runs {
		if r.Status != "completed" {
			continue
		}
		done++
		switch {
		case !passing[r.Conclusion]:
			bad = append(bad, r)
		case r.Conclusion == "success":
			success = true
		}
	}
	switch {
	case len(runs) == 0 || done < len(runs):
		return pending, done, bad
	case len(bad) > 0:
		return failed, done, bad
	case !success:
		return notGreen, done, nil
	}
	return green, done, nil
}

// jobURL is a GitHub Actions job page: .../actions/runs/<run id>/job/<job id>.
var jobURL = regexp.MustCompile(`/actions/runs/([0-9]+)/job/([0-9]+)`)

// runAndJob returns the workflow run and job ids of an Actions check run ("" when it is not one).
func runAndJob(r checkRun) (run, job string) {
	if m := jobURL.FindStringSubmatch(r.DetailsURL); m != nil {
		return m[1], m[2]
	}
	return "", ""
}

// tail is the last n lines of s, without a trailing empty line.
func tail(s string, n int) []string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}
