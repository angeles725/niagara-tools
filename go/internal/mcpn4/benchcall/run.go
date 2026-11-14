package benchcall

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/livesmoke"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// options is the parsed command line.
type options struct {
	calls      string
	dryRunOnly bool
	timeout    float64
	server     []string
}

// Main runs the command; environ is the environment the server is started with and the one the
// credential values to mask are read from.
func Main(argv []string, s cli.Stdio, environ []string) int {
	o, err := parse(argv)
	if err != nil {
		fmt.Fprintf(s.Err, "%s%s: error: %s\n", usageText, prog, err)
		return 2
	}
	calls, err := loadCalls(o.calls)
	if err != nil {
		fmt.Fprintf(s.Err, "%s: %s\n", prog, err)
		return 2
	}
	secrets := secretsOf(o.server, environ)
	client, err := livesmoke.StartClient(o.server, "", environ, time.Duration(o.timeout*float64(time.Second)))
	if err != nil {
		fmt.Fprintf(s.Err, "%s: server did not start: %s\n", prog, livesmoke.Scrub(err.Error(), secrets))
		return 1
	}
	defer client.Close()
	if err := client.Start(); err != nil {
		fmt.Fprintf(s.Err, "%s: %s\n", prog, livesmoke.Scrub(err.Error(), secrets))
		return 1
	}
	r := &runner{client: client, out: func(line string) { fmt.Fprintln(s.Out, livesmoke.Scrub(line, secrets)) }, dryRunOnly: o.dryRunOnly}
	for _, c := range calls {
		if err := r.run(c); err != nil {
			fmt.Fprintf(s.Err, "%s: %s: %s\n", prog, c.tool, livesmoke.Scrub(err.Error(), secrets))
			return 1
		}
		if r.failed {
			return 1
		}
	}
	return 0
}

// parse reads the flags before "--" and the server command line after it.
func parse(argv []string) (*options, error) {
	o := &options{timeout: defaultCallTimeout}
	for i := 0; i < len(argv); i++ {
		switch a := argv[i]; a {
		case "--":
			o.server = argv[i+1:]
			if o.calls == "" || len(o.server) == 0 {
				return nil, fmt.Errorf("--calls FILE and a server command after -- are required")
			}
			return o, nil
		case "--dry-run-only":
			o.dryRunOnly = true
		case "--calls", "--call-timeout":
			if i+1 >= len(argv) {
				return nil, fmt.Errorf("%s needs a value", a)
			}
			i++
			if a == "--calls" {
				o.calls = argv[i]
			} else if f, err := strconv.ParseFloat(argv[i], 64); err != nil || !(f > 0) {
				return nil, fmt.Errorf("--call-timeout: not a positive number: %s", argv[i])
			} else {
				o.timeout = f
			}
		default:
			return nil, fmt.Errorf("unrecognized argument: %s", a)
		}
	}
	return nil, fmt.Errorf("--calls FILE and a server command after -- are required")
}

// secretsOf is the credential values of the prefix the server command line names.
func secretsOf(server, environ []string) []string {
	prefix := "MCP_N4"
	for i, a := range server {
		if a == "--credential-env" && i+1 < len(server) {
			prefix = server[i+1]
		} else if v, ok := strings.CutPrefix(a, "--credential-env="); ok {
			prefix = v
		}
	}
	var out []string
	for _, kv := range environ {
		for _, name := range []string{prefix + "_USER=", prefix + "_PASSWORD="} {
			if v, ok := strings.CutPrefix(kv, name); ok && v != "" {
				out = append(out, v)
			}
		}
	}
	return out
}

// runner sends calls and prints one line per request.
type runner struct {
	client     *livesmoke.Client
	out        func(string)
	dryRunOnly bool
	failed     bool // a tool answered with an error
}

func (r *runner) run(c call) error {
	if !c.write {
		_, err := r.request(c.tool, phaseRead, c.args)
		return err
	}
	c.args.Set("dry_run", true)
	plan, err := r.request(c.tool, phaseDryRun, c.args)
	if err != nil || r.failed || r.dryRunOnly {
		return err
	}
	obj, _ := plan.(*pyjson.Object)
	var token any
	if obj != nil {
		token, _ = obj.Get("confirmation_token")
	}
	if text, ok := token.(string); !ok || text == "" {
		return fmt.Errorf("the dry run returned no confirmation_token")
	}
	c.args.Set("dry_run", false)
	c.args.Set("confirmation_token", token)
	_, err = r.request(c.tool, phaseExecute, c.args)
	return err
}

// request is one tools/call and its output line.
func (r *runner) request(tool, phase string, args *pyjson.Object) (any, error) {
	isErr, res, err := r.client.Call(tool, args)
	if err != nil {
		return nil, err
	}
	line := pyjson.NewObject()
	line.Set("tool", tool)
	line.Set("phase", phase)
	line.Set("error", isErr)
	line.Set("result", res)
	text, err := pyjson.Marshal(line, pyjson.Options{Compact: true})
	if err != nil {
		return nil, err
	}
	r.out(string(text))
	r.failed = r.failed || isErr
	return res, nil
}
