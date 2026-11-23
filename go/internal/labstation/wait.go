package labstation

import (
	"io"
	"os"
	"strings"
	"time"
)

// readFrom returns the log from offset on, carriage returns dropped.
func (l *lab) readFrom(off int64) string {
	f, err := os.Open(l.path("out.log"))
	if err != nil {
		return ""
	}
	defer f.Close()
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return ""
	}
	b, _ := io.ReadAll(f)
	return strings.ReplaceAll(string(b), "\r", "")
}

// waitLine waits for a log line after off that match accepts. It gives up when done closes (the process exited:
// exited is true) or the timeout passes; then line is the one that says why the station could not boot, or the last.
func (l *lab) waitLine(off int64, match func(string) bool, done <-chan struct{}) (line string, ok, exited bool) {
	deadline := time.Now().Add(l.timeout)
	for {
		lines := strings.Split(l.readFrom(off), "\n")
		for _, s := range lines {
			if match(s) {
				return clean(s), true, false
			}
		}
		select {
		case <-done:
			time.Sleep(50 * time.Millisecond) // the last lines may land just after the exit
			for _, s := range strings.Split(l.readFrom(off), "\n") {
				if match(s) {
					return clean(s), true, true
				}
			}
			return reason(strings.Split(l.readFrom(off), "\n")), false, true
		default:
		}
		if time.Now().After(deadline) {
			return "timed out after " + l.timeout.String() + ": " + reason(lines), false, false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func reason(lines []string) string {
	last := ""
	for _, s := range lines {
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		if strings.Contains(s, "Cannot boot") {
			return s
		}
		last = s
	}
	return last
}

// waitText waits until the log after off holds mark and returns that text.
func (l *lab) waitText(off int64, mark string) (string, bool) {
	deadline := time.Now().Add(l.timeout)
	for time.Now().Before(deadline) {
		if t := l.readFrom(off); strings.Contains(t, mark) {
			return t, true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return "", false
}

// clean drops the console prompt a log line can start with: the station writes niagara> without a newline, so the
// next log line follows it on the same line.
func clean(s string) string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "niagara>"))
}
