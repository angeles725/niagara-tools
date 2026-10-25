package box

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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
	c.Close() // the caller's own close after the failed open (connect does it): no second del
	if keys := st.keys(); len(keys) != 3 || keys[2] != "del" || c.SID() != nil {
		t.Fatalf("requests %v, sid %v: want make, makessc, del and no sid", keys, c.SID())
	}
}

// A station that answers del a byte at a time, each well within the socket timeout, is still cut
// at one client timeout: the shutdown's del is bounded as a whole.
func TestADrippingDelIsCutAtTheClientTimeout(t *testing.T) {
	stop := make(chan struct{})
	st := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		for {
			w.Write([]byte(" "))
			w.(http.Flusher).Flush()
			select {
			case <-stop:
				return
			case <-r.Context().Done():
				return
			case <-time.After(20 * time.Millisecond):
			}
		}
	}))
	t.Cleanup(st.Close)
	t.Cleanup(func() { close(stop) }) // first: the handler returns, so Close does not wait on it
	c := newClient(t, st.URL, func(cfg *Config) { cfg.Timeout = 0.2 })
	c.sid = "sess-1"
	closed := make(chan struct{})
	go func() { c.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(3 * time.Second):
		t.Fatal("Close still waits on the dripping del after 3 s (client timeout 0.2 s)")
	}
	if c.SID() != nil {
		t.Errorf("sid %v after Close", c.SID())
	}
}
