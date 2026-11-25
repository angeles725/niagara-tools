package n4probe

import (
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

var bucketBounds = []time.Duration{100 * time.Millisecond, 250 * time.Millisecond, 500 * time.Millisecond,
	time.Second, 2 * time.Second, 5 * time.Second}

// stallStats is what a stall run saw.
type stallStats struct {
	lat       []time.Duration
	transport int
	status    map[int]int // non-2xx answers by status
}

func runStall(args []string, s cli.Stdio) int {
	var c common
	var target string
	fs := flag.NewFlagSet("stall", flag.ContinueOnError)
	c.register(fs)
	fs.StringVar(&target, "url", "", "URL to GET")
	if code, done := parse(fs, args, s); done {
		return code
	}
	if err := checkURL("--url", target); err != nil {
		return usageErr(s, "%v", err)
	}
	cl, err := c.client()
	if err != nil {
		return usageErr(s, "%v", err)
	}
	st := stallStats{status: map[int]int{}}
	every(time.Now(), c.rate, c.duration, func(time.Duration) bool {
		st.add(cl.do(http.MethodGet, target, nil))
		return false
	})
	fmt.Fprintf(s.Out, "stall GET %s at %g Hz for %s\n", label(target), c.rate, c.duration)
	st.print(s.Out)
	if st.errors() == len(st.lat) {
		return ExitFailed
	}
	return ExitOK
}

func (st *stallStats) add(r reply) {
	st.lat = append(st.lat, r.took)
	switch {
	case r.err != nil:
		st.transport++
	case !r.ok():
		st.status[r.status]++
	}
}

func (st *stallStats) errors() int {
	n := st.transport
	for _, c := range st.status {
		n += c
	}
	return n
}

// percentile is the nearest-rank percentile of a sorted sample.
func percentile(sorted []time.Duration, p float64) time.Duration {
	i := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	return sorted[max(i, 0)]
}

func ms(d time.Duration) string { return fmt.Sprintf("%.1f ms", float64(d)/float64(time.Millisecond)) }

func (st *stallStats) print(w io.Writer) {
	fmt.Fprintf(w, "requests %d\nerrors %d (transport %d", len(st.lat), st.errors(), st.transport)
	codes := make([]int, 0, len(st.status))
	for code := range st.status {
		codes = append(codes, code)
	}
	sort.Ints(codes)
	for _, code := range codes {
		fmt.Fprintf(w, ", HTTP %d %d", code, st.status[code])
	}
	fmt.Fprintln(w, ")")
	sorted := append([]time.Duration(nil), st.lat...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	fmt.Fprintf(w, "latency min %s p50 %s p90 %s p99 %s max %s\n", ms(sorted[0]), ms(percentile(sorted, 50)),
		ms(percentile(sorted, 90)), ms(percentile(sorted, 99)), ms(sorted[len(sorted)-1]))
	counts := make([]int, len(bucketBounds)+1)
	for _, d := range sorted {
		counts[sort.Search(len(bucketBounds), func(i int) bool { return d < bucketBounds[i] })]++
	}
	fmt.Fprintln(w, "histogram")
	for i, n := range counts {
		name := ">= " + bucketBounds[len(bucketBounds)-1].String()
		if i < len(bucketBounds) {
			name = "< " + bucketBounds[i].String()
		}
		fmt.Fprintf(w, "  %-8s %d\n", name, n)
	}
}
