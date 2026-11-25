package n4probe

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestStallStatsPercentilesAndHistogram(t *testing.T) {
	st := stallStats{status: map[int]int{}}
	for i := 1; i <= 100; i++ {
		st.add(reply{status: 200, took: time.Duration(i) * time.Millisecond})
	}
	st.add(reply{status: 503, took: 300 * time.Millisecond})
	st.add(reply{status: 503, took: 6 * time.Second})
	st.add(reply{err: fmt.Errorf("refused"), took: time.Millisecond})
	var b bytes.Buffer
	st.print(&b)
	for _, want := range []string{
		"requests 103\n", "errors 3 (transport 1, HTTP 503 2)\n",
		"latency min 1.0 ms p50 51.0 ms p90 92.0 ms p99 300.0 ms max 6000.0 ms\n",
		"  < 100ms  100\n", "  < 250ms  1\n", "  < 500ms  1\n", "  < 1s     0\n", "  < 2s     0\n", "  < 5s     0\n", "  >= 5s    1\n",
	} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("report lacks %q:\n%s", want, b.String())
		}
	}
}

func TestStallMeasuresControlledDelays(t *testing.T) {
	var n atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1)%3 == 0 {
			time.Sleep(130 * time.Millisecond)
		}
	}))
	defer srv.Close()
	code, out, errs := run("stall", "--url", srv.URL, "--env-file", writeEnv(t, creds), "--rate", "100", "--allow-rate-above-4", "--duration", "800ms")
	if code != ExitOK || errs != "" {
		t.Fatalf("code %d err %q", code, errs)
	}
	count := bucket(t, out, "requests ")
	slow, fast := bucket(t, out, "  < 250ms "), bucket(t, out, "  < 100ms ")
	if count < 6 || slow < 2 || fast < 4 || slow+fast != count {
		t.Errorf("requests %d, fast %d, slow %d: want a fast and a slow group that sum to the count\n%s", count, fast, slow, out)
	}
	if !strings.Contains(out, "errors 0 (transport 0)") || !strings.Contains(out, "latency min ") {
		t.Errorf("report lacks the error and latency lines:\n%s", out)
	}
}

func TestStallCountsErrorsByStatus(t *testing.T) {
	var n atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1)%2 == 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer srv.Close()
	code, out, _ := run("stall", "--url", srv.URL, "--env-file", writeEnv(t, creds), "--rate", "100", "--allow-rate-above-4", "--duration", "300ms")
	if code != ExitOK || !regexp.MustCompile(`errors [1-9]\d* \(transport 0, HTTP 503 [1-9]\d*\)`).MatchString(out) {
		t.Errorf("code %d, want 0 and a 503 count:\n%s", code, out)
	}
}

func TestStallExitsOneWhenEveryRequestFails(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := down.URL
	down.Close()
	code, out, _ := run("stall", "--url", url, "--env-file", writeEnv(t, creds), "--rate", "100", "--allow-rate-above-4", "--duration", "100ms")
	if code != ExitFailed || !strings.Contains(out, "transport ") || strings.Contains(out, "transport 0") {
		t.Errorf("transport failure: code %d, want 1 with a transport count:\n%s", code, out)
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	defer bad.Close()
	if code, _, _ := run("stall", "--url", bad.URL, "--env-file", writeEnv(t, creds), "--rate", "100", "--allow-rate-above-4", "--duration", "100ms"); code != ExitFailed {
		t.Errorf("every request 500: code %d, want 1", code)
	}
}

// bucket is the integer after prefix on its output line.
func bucket(t *testing.T, out, prefix string) int {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if rest, ok := strings.CutPrefix(line, prefix); ok {
			n, err := strconv.Atoi(strings.Fields(rest)[0])
			if err != nil {
				t.Fatalf("line %q: %v", line, err)
			}
			return n
		}
	}
	t.Fatalf("no line starts with %q:\n%s", prefix, out)
	return 0
}
