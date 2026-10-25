package niagaratest

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf16"
)

// Cmd is one process the run starts. Env entries are added to the process environment.
type Cmd struct {
	Name   string
	Args   []string
	Dir    string
	Env    []string
	Stdout io.Writer // nil discards
	Stderr io.Writer // nil discards
}

// Runner is the command seam: every process of a run goes through it, so the orchestration is tested with a
// fake. Run returns the exit code, or an error when the process could not start or ctx ended it.
type Runner interface {
	Run(ctx context.Context, c Cmd) (int, error)
}

// host is where a run happens: the runner, the WSL mount root of the Windows drives, and the WSL home.
type host struct {
	run  Runner
	mnt  string
	home string
}

// execRunner runs a real process in its own process group; ctx ending kills the whole group.
type execRunner struct{}

func (execRunner) Run(ctx context.Context, c Cmd) (int, error) {
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	cmd.Dir, cmd.Stdout, cmd.Stderr = c.Dir, c.Stdout, c.Stderr
	if len(c.Env) > 0 {
		cmd.Env = append(os.Environ(), c.Env...)
	}
	ownGroup(cmd)
	cmd.WaitDelay = 10 * time.Second
	err := cmd.Run()
	if ctx.Err() != nil {
		return -1, ctx.Err()
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), nil
	}
	if err != nil {
		return -1, err
	}
	return 0, nil
}

// capped keeps the first max bytes written to it and drops the rest: process output is never unbounded.
type capped struct {
	buf bytes.Buffer
	max int
}

func (c *capped) Write(p []byte) (int, error) {
	if room := c.max - c.buf.Len(); room > 0 {
		c.buf.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}

// maxCaptured bounds what is kept of any probe or run output.
const maxCaptured = 64 << 10

// powershell runs script through -EncodedCommand (UTF-16LE base64), so nothing in it is re-quoted by WSL
// interop or cmd.exe; it returns the output lines, or ok false when the script did not print its final DONE.
func (h host) powershell(ctx context.Context, script string) ([]string, bool) {
	u := utf16.Encode([]rune(script + "\n'DONE'"))
	b := make([]byte, 2*len(u))
	for i, r := range u {
		b[2*i], b[2*i+1] = byte(r), byte(r>>8)
	}
	out := &capped{max: maxCaptured}
	code, err := h.run.Run(ctx, Cmd{Name: "powershell.exe", Args: []string{"-NoProfile", "-NonInteractive", "-EncodedCommand",
		base64.StdEncoding.EncodeToString(b)}, Stdout: out})
	var rows []string
	for _, l := range strings.Split(strings.ReplaceAll(out.buf.String(), "\r", ""), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			rows = append(rows, l)
		}
	}
	if err != nil || code != 0 || len(rows) == 0 || rows[len(rows)-1] != "DONE" {
		return nil, false
	}
	return rows[:len(rows)-1], true
}

// inUseScript lists what makes an install unsafe to copy from: a station.exe, nre.exe, java.exe or wb.exe
// whose executable path or command line names the install (PROC <pid> <image>), and a stations\*\config.bog.lock
// that another process holds open (LOCK <path>): a running station keeps that file open, so an exclusive
// read-only open fails with a sharing violation. Opening it never writes. A probe that cannot tell prints UNKNOWN.
const inUseScript = `$h = '%s\'
$names = 'station.exe','nre.exe','java.exe','wb.exe'
Get-CimInstance Win32_Process -ErrorAction Stop | Where-Object { $names -contains ([string]$_.Name).ToLower() } |
  Where-Object { ([string]$_.CommandLine + ' ' + [string]$_.ExecutablePath).IndexOf($h, [StringComparison]::OrdinalIgnoreCase) -ge 0 } |
  ForEach-Object { 'PROC ' + $_.ProcessId + ' ' + $_.Name }
$st = Join-Path $h 'stations'
if (Test-Path -LiteralPath $st) {
  try { $dirs = Get-ChildItem -LiteralPath $st -Directory -ErrorAction Stop } catch { 'UNKNOWN ' + $st; $dirs = @() }
  foreach ($d in $dirs) {
    $l = Join-Path $d.FullName 'config.bog.lock'
    if (Test-Path -LiteralPath $l) {
      try { $f = [IO.File]::Open($l, 'Open', 'Read', 'None'); $f.Close() }
      catch [IO.IOException] { 'LOCK ' + $l }
      catch { 'UNKNOWN ' + $l }
    }
  }
}`

// procsScript lists every Windows process of this run: those whose executable path or command line names the
// scratch dir the run created (its name is random), the probe itself excepted (PID <pid>).
const procsScript = `$t = '%s'
Get-CimInstance Win32_Process -ErrorAction Stop | Where-Object { $_.ProcessId -ne $PID } |
  Where-Object { ([string]$_.CommandLine + ' ' + [string]$_.ExecutablePath).IndexOf($t, [StringComparison]::OrdinalIgnoreCase) -ge 0 } |
  ForEach-Object { 'PID ' + $_.ProcessId }`
