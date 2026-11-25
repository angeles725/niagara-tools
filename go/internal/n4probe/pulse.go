package n4probe

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// pulseRun is the state of one pulse: the baseline, what each field shows now and which ones ever moved.
type pulseRun struct {
	w       io.Writer
	fields  []string
	base    map[string]string
	now     map[string]string
	changed map[string]bool
	reads   int
	slowest time.Duration
	lastErr string
}

func runPulse(args []string, s cli.Stdio) int {
	var c common
	var press, observe, body string
	var fields stringList
	fs := flag.NewFlagSet("pulse", flag.ContinueOnError)
	c.register(fs)
	fs.StringVar(&press, "press", "", "URL to POST")
	fs.StringVar(&body, "press-body", "{}", "JSON body of the press")
	fs.StringVar(&observe, "observe", "", "URL to GET and watch")
	fs.Var(&fields, "field", "dot path of a field to watch (repeatable)")
	if code, done := parse(fs, args, s); done {
		return code
	}
	for flagName, raw := range map[string]string{"--press": press, "--observe": observe} {
		if err := checkURL(flagName, raw); err != nil {
			return usageErr(s, "%v", err)
		}
	}
	if len(fields) == 0 {
		return usageErr(s, "at least one --field is required")
	}
	if !json.Valid([]byte(body)) {
		return usageErr(s, "--press-body is not valid JSON")
	}
	cl, err := c.client()
	if err != nil {
		return usageErr(s, "%v", err)
	}
	r := &pulseRun{w: s.Out, fields: fields, base: map[string]string{}, now: map[string]string{}, changed: map[string]bool{}}
	first := cl.do(http.MethodGet, observe, nil)
	if !first.ok() || !r.baseline(first.body) {
		fmt.Fprintln(s.Err, "n4-probe: baseline read of --observe failed (transport error, non-2xx or not JSON)")
		return ExitFailed
	}
	t0 := time.Now()
	pr := cl.do(http.MethodPost, press, []byte(body))
	if pr.err != nil {
		fmt.Fprintf(s.Err, "n4-probe: press failed: %v\n", pr.err)
		return ExitFailed
	}
	fmt.Fprintf(s.Out, "press status %d (%s) %s\n", pr.status, ms(pr.took), label(press))
	released := time.Duration(-1)
	every(t0, c.rate, c.duration, func(time.Duration) bool {
		rd := cl.do(http.MethodGet, observe, nil)
		at := time.Since(t0)
		if r.observe(rd, at) {
			released = at
		}
		return released >= 0
	})
	if released >= 0 {
		fmt.Fprintf(s.Out, "released after %d ms (%d reads, slowest %s)\n", released.Milliseconds(), r.reads, ms(r.slowest))
		return ExitOK
	}
	fmt.Fprintf(s.Out, "not released within %s (%d reads, slowest %s)\n", c.duration, r.reads, ms(r.slowest))
	if list := r.pending(func(f string) bool { return !r.changed[f] }); list != "" {
		fmt.Fprintf(s.Out, "  never changed: %s\n", list)
	}
	if list := r.pending(func(f string) bool { return r.changed[f] && r.now[f] != r.base[f] }); list != "" {
		fmt.Fprintf(s.Out, "  not back at baseline: %s\n", list)
	}
	return ExitNotReleased
}

// baseline records the first read; false when it is not JSON.
func (r *pulseRun) baseline(body []byte) bool {
	doc, err := decode(body)
	if err != nil {
		return false
	}
	for _, f := range r.fields {
		r.base[f] = render(doc, f)
		r.now[f] = r.base[f]
		fmt.Fprintf(r.w, "baseline %s=%s\n", f, r.base[f])
	}
	return true
}

// observe takes one read: it prints each field change (ms since the press, field, old -> new) and reports
// whether every field has changed and is back at its baseline.
func (r *pulseRun) observe(rd reply, at time.Duration) bool {
	r.reads++
	r.slowest = max(r.slowest, rd.took)
	var doc any
	err := rd.err
	if err == nil && !rd.ok() {
		err = fmt.Errorf("HTTP %d", rd.status)
	}
	if err == nil {
		doc, err = decode(rd.body)
	}
	if err != nil {
		if msg := err.Error(); msg != r.lastErr {
			r.lastErr = msg
			fmt.Fprintf(r.w, "%6d ms  read error: %s\n", at.Milliseconds(), msg)
		}
		return false
	}
	r.lastErr = ""
	back := true
	for _, f := range r.fields {
		if v := render(doc, f); v != r.now[f] {
			fmt.Fprintf(r.w, "%6d ms  %s  %s -> %s\n", at.Milliseconds(), f, r.now[f], v)
			r.now[f] = v
			r.changed[f] = r.changed[f] || v != r.base[f]
		}
		back = back && r.changed[f] && r.now[f] == r.base[f]
	}
	return back
}

// pending lists the fields for which keep is true, comma separated.
func (r *pulseRun) pending(keep func(string) bool) string {
	var out []string
	for _, f := range r.fields {
		if keep(f) {
			out = append(out, f)
		}
	}
	return strings.Join(out, ", ")
}
