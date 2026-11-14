package livesmoke

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/atomicfile"
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

const prog = "live-smoke"

// Main runs the command: plan only unless --apply. environ is the environment the server is
// started with and the one credentials and MCP_N4_GO_BIN are read from. Exit 0 plan printed or
// every required step verified; 1 a run that did not verify (or a report that cannot be
// written); 2 usage.
func Main(argv []string, s cli.Stdio, environ []string) int {
	a, err := ParseArgs(argv)
	if errors.Is(err, errHelp) {
		fmt.Fprint(s.Out, helpText)
		return 0
	}
	var ue *UsageError
	if errors.As(err, &ue) {
		fmt.Fprintf(s.Err, "%s%s: error: %s\n", usageText, prog, ue.Msg)
		return 2
	}
	if !a.Apply {
		fmt.Fprint(s.Out, PlanText())
		return 0
	}
	stations, homes := pairs(a.Stations), pairs(a.StationHomes)
	if !applyTargetOK(stations, homes) {
		fmt.Fprintf(s.Err, "%s: --apply needs exactly one --station NAME=URL and the matching "+
			"--station-home NAME=PATH\n", prog)
		return 2
	}
	getenv := lookupIn(environ)
	if problem := ResolveServer(a, getenv); problem != "" {
		fmt.Fprintf(s.Err, "%s: %s\n", prog, problem)
		return 2
	}
	name := stations.keys[0]
	secrets := []string{getenv(a.CredentialEnv + "_USER"), getenv(a.CredentialEnv + "_PASSWORD")}
	log := func(text string) { fmt.Fprintln(s.Out, Scrub(text, secrets)) }

	var runErr any
	var steps []*pyjson.Object
	var stderr []string
	client, err := StartClient(ServerArgv(a), kitDir(), environ, seconds(a.CallTimeout))
	if err != nil {
		runErr = "server did not start: " + err.Error()
	} else {
		scenario := NewScenario(client, name, a.Settle, log)
		if err := client.Start(); err != nil {
			runErr = errorDetail(err)
		} else {
			scenario.Run()
		}
		client.Close()
		steps, stderr = scenario.Steps, client.StderrLines()
	}
	report := BuildReport(name, steps, runErr, stderr)
	text, err := RenderReport(report, secrets)
	if err != nil {
		fmt.Fprintf(s.Err, "%s: %s\n", prog, err)
		return 1
	}
	if a.Report != "" {
		if err := atomicfile.Write(a.Report, []byte(text+"\n")); err != nil {
			fmt.Fprintf(s.Err, "%s: %s\n", prog, Scrub(err.Error(), secrets))
			return 1
		}
	}
	fmt.Fprintln(s.Out, text)
	if ok, _ := report.Get("ok"); ok == true {
		return 0
	}
	return 1
}

// seconds is a timeout in seconds as a duration: one past any real run is as long as Go can
// wait; nan is 0, as Python's max(0.01, nan) leaves only the minimum wait.
func seconds(f float64) time.Duration {
	if math.IsNaN(f) {
		return 0
	}
	if !(f < 1e9) {
		return time.Duration(1<<63 - 1)
	}
	return time.Duration(f * float64(time.Second))
}

// orderedPairs is dict(item.partition("=")[::2] for item in items): the first occurrence of a
// name keeps its position, the last value wins.
type orderedPairs struct {
	keys []string
	vals map[string]string
}

func pairs(items []string) orderedPairs {
	p := orderedPairs{vals: map[string]string{}}
	for _, it := range items {
		k, v, _ := strings.Cut(it, "=")
		if _, seen := p.vals[k]; !seen {
			p.keys = append(p.keys, k)
		}
		p.vals[k] = v
	}
	return p
}

// applyTargetOK is one station with a non-empty URL and homes naming exactly that station.
func applyTargetOK(stations, homes orderedPairs) bool {
	if len(stations.keys) != 1 || len(homes.keys) != 1 || homes.keys[0] != stations.keys[0] {
		return false
	}
	return stations.vals[stations.keys[0]] != ""
}

// lookupIn reads one variable of environ ("" when unset); the last assignment wins.
func lookupIn(environ []string) func(string) string {
	return func(name string) string {
		val := ""
		for _, kv := range environ {
			if k, v, ok := strings.Cut(kv, "="); ok && k == name {
				val = v
			}
		}
		return val
	}
}
