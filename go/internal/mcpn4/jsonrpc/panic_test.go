package jsonrpc

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pytarget"
)

// raisingDriver answers callEcho on the Python server whose tool raises an exception the server
// does not expect (its catch-all), then a ping: the process must live on.
const raisingDriver = `
import sys
sys.path.insert(0, sys.argv[1])
from mcp_n4 import server
srv = server.Server()
def boom(ctx, args):
    raise Exception("topsecret-value")
srv.tools["echo"] = srv.tools["n4_navigate"]._replace(name="echo", handler=boom, needs_session=False)
for line in sys.stdin:
    print(srv.handle_line(line))
`

const pingLine = `{"jsonrpc": "2.0", "id": 2, "method": "ping"}`

func panicking(*pyjson.Object) (*pyjson.Object, error) {
	var o *pyjson.Object
	o.Set("topsecret-value", 1) // a nil dereference
	return o, nil
}

func TestAPanickingToolAnswersAsPythonsCatchAllAndTheServerLivesOn(t *testing.T) {
	py := pytarget.Python(t)
	kit, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "mcp-n4-kit"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(py, "-I", "-c", raisingDriver, kit)
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	cmd.Stdin = strings.NewReader(callEcho + "\n" + pingLine + "\n")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("python3 driver: %v", err)
	}
	s := echoServer(panicking)
	got := handle(t, s, callEcho) + "\n" + handle(t, s, pingLine) + "\n"
	if got != string(out) {
		t.Errorf("go:\n%s\npython:\n%s", got, out)
	}
	if strings.Contains(got, "topsecret") || !strings.Contains(got, "internal error (Exception)") {
		t.Errorf("reply %s", got)
	}
}

func TestRecoverPassesResultsAndErrorsThrough(t *testing.T) {
	t.Parallel()
	v, err := Recover(func() (int, error) { return 7, nil })
	if v != 7 || err != nil {
		t.Errorf("%v %v", v, err)
	}
	want := &ToolError{"no"}
	if _, err := Recover(func() (int, error) { return 0, want }); err != want {
		t.Errorf("%v", err)
	}
	v, err = Recover(func() (int, error) { panic("x") })
	if p, ok := err.(*Panic); !ok || v != 0 || p.PyType() != "Exception" || p.Error() != "x" {
		t.Errorf("%v %v", v, err)
	}
}
