// Package previewshot takes a headless screenshot of a dashboard preview (issue #1126). Two modes: a URL that
// is already being served, or --serve <rc-dir>, which starts dashboard-preview on a port the OS hands out,
// records the server's PID in a state file, waits until it answers, shoots, and stops exactly that process.
// A WebGL canvas is black without software GL (see defaultGL in browser.go) and the shot is taken before the
// page's polls fill it without --virtual-time-budget; both are always passed.
package previewshot

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const (
	exitFail  = 1
	exitUsage = 3
)

// Main is the preview-shot command.
func Main(args []string, s cli.Stdio) int {
	o, err := parse(args)
	if err != nil {
		return failf(s, exitUsage, "%v\n%s", err, usage)
	}
	shell, err := findBrowser(s)
	if err != nil {
		return fail(s, err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	target := o.url
	if o.serve != "" {
		bin, err := findServer(s)
		if err != nil {
			return fail(s, err)
		}
		sv, err := startServer(ctx, s, bin, o)
		if sv != nil {
			defer sv.stop() // always: a failed wait, a failed shot and an interrupt all end here
		}
		if err != nil {
			return fail(s, err)
		}
		target = sv.url
	}
	size, err := takeShot(ctx, s, shell, target, o)
	if err != nil {
		return fail(s, err)
	}
	fmt.Fprintf(s.Out, "preview-shot: wrote %s (%d bytes)\n", o.out, size)
	return 0
}

// exitError is an error that carries its exit code.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

func errf(code int, format string, a ...any) error {
	return &exitError{code, fmt.Sprintf(format, a...)}
}

func fail(s cli.Stdio, err error) int {
	if ee, ok := err.(*exitError); ok {
		return failf(s, ee.code, "%s", ee.msg)
	}
	return failf(s, exitFail, "%v", err)
}

func failf(s cli.Stdio, code int, format string, a ...any) int {
	fmt.Fprintf(s.Err, "preview-shot: "+format+"\n", a...)
	return code
}
