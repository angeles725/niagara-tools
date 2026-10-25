package cli

import (
	"context"
	"os"
	"syscall"
)

// watchSignals turns the first signal on sigs into the cancellation of the returned context: the
// station requests in flight stop, the serve loop ends, the station session is closed (its del
// bounded by the client timeout) and the process exits 0, as Python does after KeyboardInterrupt.
// A second signal while that shutdown runs exits at once with 128+signo; no file is left
// half-written, as the journal and the audit append each line in one write.
func watchSignals(sigs <-chan os.Signal, exit func(int)) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sigs
		cancel()
		code := 128 + int(syscall.SIGINT)
		if n, ok := (<-sigs).(syscall.Signal); ok {
			code = 128 + int(n)
		}
		exit(code)
	}()
	return ctx
}
