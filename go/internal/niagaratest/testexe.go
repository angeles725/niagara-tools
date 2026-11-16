package niagaratest

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// testExe runs the copy's bin\test.exe straight from WSL, with the copy as cwd and its streams in files, then
// judges testng-results.xml with stderr as the console cross-check; the exit code is a third check only (WSL
// reports the Windows exit -1 as 255). The scratch niagara_user_home reaches the NRE through the interop
// environment (WSLENV niagara_user_home/w): a cmd.exe `set "…"` does not, as WSL re-quotes its inner quotes.
// D3 still gates it: the NRE must have created etc\ and security\ in the scratch home and written a file under
// it during this boot (nuhproof.go), and cleanup compares the real user homes.
func (h host) testExe(ctx context.Context, p *plan, s cli.Stdio) (int, error) {
	files := map[string]*os.File{}
	for _, n := range []string{"stdout", "stderr"} {
		f, err := os.Create(p.sub("out/" + n + ".txt"))
		if err != nil {
			return -1, err
		}
		defer f.Close()
		files[n] = f
	}
	wslenv := "niagara_user_home/w"
	if v := os.Getenv("WSLENV"); v != "" {
		wslenv = v + ":" + wslenv
	}
	before, err := walkNUH(p.sub("nuh"))
	if err != nil {
		return -1, fmt.Errorf("cannot read the scratch niagara_user_home before the boot: %v", err)
	}
	start := time.Now()
	exit, err := h.run.Run(ctx, Cmd{Name: p.home().wsl + "/bin/test.exe", Dir: p.home().wsl,
		Args:   []string{p.module, "-v:2", "-output:" + p.subWin("out/testng")},
		Env:    []string{"WSLENV=" + wslenv, "niagara_user_home=" + p.subWin("nuh")},
		Stdout: files["stdout"], Stderr: files["stderr"]})
	end := time.Now()
	if err != nil {
		return -1, err
	}
	code := testngVerdict(p.sub("out/testng/testng-results.xml"), []string{p.sub("out/stderr.txt")}, p.allow, s, nil)
	if code != exitPass {
		signatureHint(p, s, p.sub("out/stderr.txt"), p.sub("out/stdout.txt"))
	}
	written, unwritten := nuhWritten(p.sub("nuh"), before, start, end)
	switch {
	case !isDir(p.sub("nuh/etc")) || !isDir(p.sub("nuh/security")):
		row(s, "FAIL", "scratch niagara_user_home was not used (no etc/ and security/ in it)")
	case unwritten != "":
		row(s, "FAIL", unwritten)
	case exit != 0 && code == exitPass:
		row(s, "FAIL", fmt.Sprintf("run: exit %d but results green", exit))
	default:
		row(s, "PASS", written)
		return code, nil
	}
	return exitFail, nil
}
