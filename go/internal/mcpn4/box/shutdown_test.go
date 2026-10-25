package box

import (
	"context"
	"testing"
)

// An interrupt that cancels a step after make still deletes the server session: Open's cleanup
// del is not cancelled with it (test_box.TestLockoutSafeOpen).
func TestInterruptAfterMakeStillClosesTheServerSession(t *testing.T) {
	reached, release := make(chan struct{}), make(chan struct{})
	st := newScripted(t, func(s seen) *reply {
		if s.key() == "makessc" {
			close(reached)
			<-release
		}
		return nil
	})
	t.Cleanup(func() { close(release) }) // before the station's Close, which waits for handlers
	interrupt, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newClient(t, st.URL, func(cfg *Config) { cfg.Interrupt = interrupt })
	go func() { <-reached; cancel() }()
	if _, err := c.Open(); err == nil {
		t.Fatal("Open succeeded through the interrupt")
	}
	if keys := st.keys(); len(keys) != 3 || keys[2] != "del" || c.SID() != nil {
		t.Fatalf("requests %v, sid %v: want make, makessc, del and no sid", keys, c.SID())
	}
}
