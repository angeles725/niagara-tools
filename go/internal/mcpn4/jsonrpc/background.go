package jsonrpc

import "sync"

// Background is the work a reply went out before (#925: a write past its reply budget answers
// {state: running} while it finishes). The serve loop waits for it before it reads the next
// request line, and before it closes the session, so two calls never use the station session at
// once: only the reply comes early, never the next call.
type Background struct{ wg sync.WaitGroup }

// Go runs f as background work the loop waits for.
func (b *Background) Go(f func()) {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		f()
	}()
}

// Wait returns once every background work has ended; a nil Background has none.
func (b *Background) Wait() {
	if b != nil {
		b.wg.Wait()
	}
}
