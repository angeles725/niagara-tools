package niagaratest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// cleanup kills the Windows processes of this run, compares the install with its snapshot, deletes the
// scratch dir the run created (only that dir, unless --keep-home), compares the real user homes with theirs,
// then the gradle root's git status with its own (gradleroot.go), and last writes the run log (diagnostics.go)
// from the evidence read before the delete. It can only make code worse.
func (h host) cleanup(p *plan, snap, users snapshot, s cli.Stdio, code int) int {
	worse := func(c int) {
		if code == exitPass {
			code = c
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()
	if p.windows {
		if c := h.killRun(ctx, p, s); c != exitPass {
			worse(c)
		}
	}
	if now, err := takeSnapshot(p.install.wsl); err != nil {
		row(s, "FAIL", "install unchanged: cannot re-read the install: "+err.Error())
		worse(exitFail)
	} else if d := snap.diff(now); d != "" {
		row(s, "FAIL", "install changed during the run (the run already happened): "+d)
		worse(exitFail)
	} else {
		row(s, "PASS", fmt.Sprintf("install unchanged (%d entries checked)", len(snap)))
	}
	if p.verifyLow && !checkInstallSysProps(p, s) {
		worse(exitFail)
	}
	ev := gatherEvidence(p) // before the scratch dir goes: the run log is written last, with the final code
	if p.keepHome {
		row(s, "PASS", "cleanup: kept "+p.dir.win)
	} else if err := os.RemoveAll(p.dir.wsl); err != nil {
		row(s, "FAIL", "cleanup: cannot delete "+p.dir.win+": "+err.Error())
		worse(exitEnv)
	} else if p.reuseHome != "" {
		row(s, "PASS", "cleanup: deleted "+p.dir.win)
	}
	if now, err := userHomeSnapshot(p.profile); err != nil {
		row(s, "FAIL", "real user homes unchanged: cannot re-read them: "+err.Error())
		worse(exitFail)
	} else if d := users.diff(now); d != "" {
		row(s, "FAIL", "real user home changed during the run: "+d)
		worse(exitFail)
	} else {
		row(s, "PASS", fmt.Sprintf("real user homes unchanged (%d entries checked)", len(users)))
	}
	if !h.checkGradleRoot(ctx, p, s) {
		worse(exitFail)
	}
	if !h.writeRunLog(p, ev, code, s) {
		worse(exitEnv)
	}
	// A kept dir names its run log in its marker: the record a later --reuse-home binds the marker to.
	if p.keepHome && p.logPath != "" {
		p.marker.log = p.logPath
		if err := writeMarker(p.dir.wsl, p.marker); err != nil {
			row(s, "FAIL", "cleanup: cannot name the run log in the kept marker: "+err.Error())
			worse(exitEnv)
		}
	}
	return code
}

// killRun finds the processes of this run by the scratch dir name, kills each tree by its PID
// (taskkill /PID <pid> /T /F, never by image name) and lists again to confirm none is left.
func (h host) killRun(ctx context.Context, p *plan, s cli.Stdio) int {
	script := fmt.Sprintf(procsScript, filepath.Base(p.dir.wsl))
	rows, ok := h.powershell(ctx, script)
	if !ok {
		row(s, "FAIL", "cleanup: cannot list the processes of this run")
		return exitEnv
	}
	pids := pidsOf(rows)
	if len(pids) == 0 {
		return exitPass
	}
	for _, pid := range pids {
		_, _ = h.run.Run(ctx, Cmd{Name: "cmd.exe", Args: []string{"/c", "taskkill", "/PID", pid, "/T", "/F"}})
	}
	rows, ok = h.powershell(ctx, script)
	if left := pidsOf(rows); !ok || len(left) > 0 {
		row(s, "FAIL", fmt.Sprintf("cleanup: %s of this run still running: %s", plural(len(left), "process", "processes"), strings.Join(left, " ")))
		return exitEnv
	}
	row(s, "PASS", fmt.Sprintf("cleanup: %s of this run killed", plural(len(pids), "process", "processes")))
	return exitPass
}

func pidsOf(rows []string) []string {
	var pids []string
	for _, r := range rows {
		if pid, ok := strings.CutPrefix(r, "PID "); ok {
			if _, err := strconv.Atoi(pid); err == nil {
				pids = append(pids, pid)
			}
		}
	}
	return pids
}
