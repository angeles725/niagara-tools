package niagaratest

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// --module-verification low (TEST ONLY): 4.14 ships niagara.moduleVerificationMode=low in
// defaults/system.properties, 4.15 leaves it commented out so the code default medium rejects a test jar whose
// cert chain cannot be validated ("failed signature validation", then "invalid test target(s)" and exit 0).
// The flag appends the 4.14 line to the runner's own copy, never to --install. low still rejects a module that
// requests a signature-required permission group (ModuleManager, 4.14.0.162 decompile).
const (
	sysPropsRel   = "defaults/system.properties"
	verifyLowLine = "niagara.moduleVerificationMode=low"
)

// verificationFlag is --module-verification: off when unset, and `low` is its only value.
type verificationFlag struct{ low *bool }

func (f verificationFlag) String() string { return "" }

func (f verificationFlag) Set(v string) error {
	if v != "low" {
		return fmt.Errorf("--module-verification %q: the only value is low", v)
	}
	*f.low = true
	return nil
}

// fileSum is the sha256 of a file in hex, "" with an error when it cannot be read.
func fileSum(p string) (string, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:]), nil
}

// lowerVerification appends verifyLowLine to the copy's defaults/system.properties, after robocopy and before
// the build. The file must resolve, symlinks followed, inside the scratch dir this run created and outside
// --install; anything else is refused (exit 2) with nothing written.
func (h host) lowerVerification(_ context.Context, p *plan, s cli.Stdio) (int, error) {
	if !p.verifyLow {
		return -1, nil
	}
	refuse := func(format string, a ...any) (int, error) {
		row(s, "FAIL", "module verification: refused: "+fmt.Sprintf(format, a...))
		return exitUsage, nil
	}
	r, err := filepath.EvalSymlinks(p.home().wsl + "/" + sysPropsRel)
	if err != nil {
		return refuse("%v", err)
	}
	win, ok := h.toWin(r)
	switch {
	case !ok:
		return refuse("the copy's %s resolves off the Windows drives (%s)", sysPropsRel, r)
	case within(win, p.install.win):
		return refuse("the copy's %s resolves inside --install (%s)", sysPropsRel, win)
	case !within(win, p.dir.win):
		return refuse("the copy's %s resolves outside the run's scratch dir %s (%s)", sysPropsRel, p.dir.win, win)
	}
	b, err := os.ReadFile(r)
	if err != nil {
		return -1, err
	}
	eol := "\n"
	if bytes.Contains(b, []byte("\r\n")) {
		eol = "\r\n"
	}
	add := verifyLowLine + eol
	if len(b) > 0 && b[len(b)-1] != '\n' {
		add = eol + add
	}
	f, err := os.OpenFile(r, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return -1, err
	}
	if _, err := f.WriteString(add); err != nil {
		f.Close()
		return -1, err
	}
	if err := f.Close(); err != nil {
		return -1, err
	}
	row(s, "PASS", "module verification: low (copy only)")
	return -1, nil
}

// checkInstallSysProps is guard 4 at cleanup: the install's system.properties hashes as it did before the copy.
func checkInstallSysProps(p *plan, s cli.Stdio) bool {
	now, err := fileSum(p.install.wsl + "/" + sysPropsRel)
	switch {
	case err != nil:
		row(s, "FAIL", "install "+sysPropsRel+": cannot re-read it: "+err.Error())
	case now != p.sysPropsSum:
		row(s, "FAIL", fmt.Sprintf("install %s changed during the run (sha256 %s, was %s)", sysPropsRel, now[:8], p.sysPropsSum[:8]))
	default:
		row(s, "PASS", fmt.Sprintf("install %s unchanged (sha256 %s)", sysPropsRel, now[:8]))
		return true
	}
	return false
}

// signatureHint names the flag when a run without it shows the NRE rejecting the module's signature.
func signatureHint(p *plan, s cli.Stdio, logs ...string) {
	if p.verifyLow {
		return
	}
	for _, l := range logs {
		if logHas(l, badSignature) {
			row(s, "FAIL", "hint: the test NRE rejected the test module's signature; on a disposable copy pass "+
				"--module-verification low (test only; 4.15 defaults to medium, 4.14 ships low)")
			return
		}
	}
}

// logHas says whether a line of the file holds marker, in any case.
func logHas(path, marker string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), MaxLine)
	for sc.Scan() {
		if strings.Contains(strings.ToLower(sc.Text()), marker) {
			return true
		}
	}
	return false
}
