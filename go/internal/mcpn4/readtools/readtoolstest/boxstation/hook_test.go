package boxstation

import (
	"net/http"
	"strings"
	"testing"
)

// TestInstallingAHookWhileServingIsRaceFree: a test may install a hook while the server goroutines
// answer frames (go test -race reports a direct field write here).
func TestInstallingAHookWhileServingIsRaceFree(t *testing.T) {
	st := New("u", "p", "S").Start(t)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 50; i++ {
			req, _ := http.NewRequest(http.MethodPost, st.URL+"/box/", strings.NewReader(`{"m":[]}`))
			req.SetBasicAuth("u", "p")
			if res, err := http.DefaultClient.Do(req); err == nil {
				res.Body.Close()
			}
		}
	}()
	for {
		select {
		case <-done:
			return
		default:
			st.SetHook(func(map[string]any) (int, []byte) { return 0, nil })
		}
	}
}

// TestTheHookRunsWithTheStationUnlocked: a hook may take the station lock itself.
func TestTheHookRunsWithTheStationUnlocked(t *testing.T) {
	st := New("u", "p", "S").Start(t)
	ran := false // written and read under the station lock
	st.SetHook(func(map[string]any) (int, []byte) {
		st.Do(func(*Station) { ran = true }) // deadlocks if the hook ran locked
		return http.StatusTeapot, []byte("hooked")
	})
	req, _ := http.NewRequest(http.MethodPost, st.URL+"/box/", strings.NewReader(`{"m":[]}`))
	req.SetBasicAuth("u", "p")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	st.Do(func(*Station) {})
	if res.StatusCode != http.StatusTeapot || !ran {
		t.Errorf("status %d, hook ran %v", res.StatusCode, ran)
	}
}
