package niagaratest

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// --reuse-home <dir> boots a second time on a scratch dir an earlier --keep-home run kept, so a test can prove
// what survives a station restart. It is the one flag that points the runner at an existing dir, so the dir must
// lie under the scratch parent, must not be (or hold, or sit in) a real Niagara user home (D3), and must carry
// the marker the runner wrote in it, bound to the kept run's own log and to this run's install. Reused: the home
// copy (bin etc jre lib security defaults, the copied modules\*.jar, the registry the NRE built), the scratch
// niagara_user_home (station data) and the scratch user.home. Rebuilt: :<part>:jar and :moduleTestJar install
// into the kept home's modules\ again, and out\ (build log, test.exe streams, the -output: dir) starts empty.

// markerName is the marker file a run writes at the top of every scratch dir it creates or takes.
const markerName = "run-niagara-test.marker"

// markerHeader is the marker's first line, its format version.
const markerHeader = "run-niagara-test home marker 1"

// The marker's fields, in the order it writes them.
const (
	fieldRunID   = "run id"
	fieldInstall = "install"
	fieldSysSum  = "system.properties sha256"
	fieldLog     = "log"
)

// markerMax bounds the marker read; logHeadMax bounds the read of the kept run's log header.
const (
	markerMax  = 4 << 10
	logHeadMax = 64 << 10
)

// noSysProps is the marker's sha256 field for an install without defaults/system.properties.
const noSysProps = "absent"

// keptLogName is a run log's name as createLog makes it.
var keptLogName = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}Z-[A-Za-z0-9][A-Za-z0-9_.-]*\.log$`)

// marker is the parsed marker of a scratch dir.
type marker struct{ runID, install, sysSum, log string }

// newRunID is <UTC stamp>-<module>-<scratch dir suffix>-<8 hex>: unique per run, and it names its scratch dir.
func newRunID(now time.Time, module, dir string) (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%s-%s", now.UTC().Format("20060102T150405Z"), module, dirSuffix(dir), hex.EncodeToString(b[:])), nil
}

// dirSuffix is the random part of a scratch dir's name.
func dirSuffix(dir string) string { return strings.TrimPrefix(filepath.Base(dir), "run-niagara-test-") }

// namesDir says whether a run id ends with the suffix of dir and a nonce.
func namesDir(id, dir string) bool {
	return regexp.MustCompile(`-` + regexp.QuoteMeta(dirSuffix(dir)) + `-[0-9a-f]{8}$`).MatchString(id)
}

// installSum is the sha256 of the install's defaults/system.properties, noSysProps when it has none.
func installSum(install string) (string, error) {
	sum, err := fileSum(install + "/" + sysPropsRel)
	if errors.Is(err, fs.ErrNotExist) {
		return noSysProps, nil
	}
	return sum, err
}

// writeMarker writes the dir's marker whole (a temp file renamed over it), mode 0600.
func writeMarker(dir string, m marker) error {
	for _, v := range []string{m.runID, m.install, m.sysSum, m.log} {
		if strings.ContainsAny(v, "\r\n") {
			return fmt.Errorf("marker value %q holds a line break", v)
		}
	}
	body := fmt.Sprintf("%s\n%s: %s\n%s: %s\n%s: %s\n%s: %s\n", markerHeader,
		fieldRunID, m.runID, fieldInstall, m.install, fieldSysSum, m.sysSum, fieldLog, m.log)
	tmp := filepath.Join(dir, markerName+".tmp")
	if err := os.WriteFile(tmp, []byte(body), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, markerName))
}

// readMarker reads and parses dir's marker: a regular file (no symlink) of at most markerMax bytes, the header
// line then each field exactly once.
func readMarker(dir string) (marker, error) {
	p := filepath.Join(dir, markerName)
	info, err := os.Lstat(p)
	if err != nil {
		return marker{}, fmt.Errorf("has no %s (only a dir a run created carries one)", markerName)
	}
	if !info.Mode().IsRegular() {
		return marker{}, fmt.Errorf("its %s is not a regular file", markerName)
	}
	f, err := os.Open(p)
	if err != nil {
		return marker{}, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, markerMax+1))
	if err != nil {
		return marker{}, err
	}
	if len(b) > markerMax {
		return marker{}, fmt.Errorf("its %s is over %d bytes", markerName, markerMax)
	}
	lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if lines[0] != markerHeader {
		return marker{}, fmt.Errorf("its %s does not start with %q", markerName, markerHeader)
	}
	var m marker
	dst := map[string]*string{fieldRunID: &m.runID, fieldInstall: &m.install, fieldSysSum: &m.sysSum, fieldLog: &m.log}
	seen := map[string]bool{}
	for _, l := range lines[1:] {
		k, v, ok := strings.Cut(l, ":")
		if !ok || dst[k] == nil || seen[k] {
			return marker{}, fmt.Errorf("its %s has a bad line %q", markerName, l)
		}
		seen[k], *dst[k] = true, strings.TrimPrefix(v, " ")
	}
	for _, k := range []string{fieldRunID, fieldInstall, fieldSysSum, fieldLog} {
		if !seen[k] {
			return marker{}, fmt.Errorf("its %s has no %s field", markerName, k)
		}
	}
	return m, nil
}

// realUserHome is D3 at the gate: the dir is, holds or sits in a %USERPROFILE%\Niagara4.*\<brand> dir, or has the
// registry\ or security\ a Niagara user home has at its top (a copy of one). A marker copied into it changes
// nothing. It names the reason, "" when the dir is none of those.
func realUserHome(dir, prof winDir) string {
	roots, _ := filepath.Glob(filepath.Join(prof.wsl, "Niagara4.*", "*"))
	for _, r := range roots {
		rel, err := filepath.Rel(prof.wsl, r)
		if err != nil || !isDir(r) {
			continue
		}
		win := prof.win + `\` + strings.ReplaceAll(rel, "/", `\`)
		if within(dir.win, win) || within(win, dir.win) {
			return fmt.Sprintf("is a real Niagara user home or overlaps one (%s)", win)
		}
	}
	for _, sub := range []string{"registry", "security"} {
		if isDir(dir.wsl + "/" + sub) {
			return fmt.Sprintf("is a real Niagara user home or a copy of one (it has %s\\ at its top)", sub)
		}
	}
	return ""
}

// checkReuse validates --reuse-home against the plan; it fills p.dir and returns the refusal, "" when the dir
// may be booted again. It runs after the scratch parent and guard 1 are settled.
func (h host) checkReuse(p *plan) string {
	dir, err := h.canonical("--reuse-home", p.reuseHome)
	if err != nil {
		return err.Error()
	}
	if !within(dir.win, p.parent.win) || within(p.parent.win, dir.win) {
		return fmt.Sprintf("--reuse-home %s is not under the scratch parent %s", dir.win, p.parent.win)
	}
	if why := realUserHome(dir, p.profile); why != "" {
		return fmt.Sprintf("--reuse-home %s %s", dir.win, why)
	}
	m, err := readMarker(dir.wsl)
	if err != nil {
		return fmt.Sprintf("--reuse-home %s %v", dir.win, err)
	}
	sum, err := installSum(p.install.wsl)
	if err != nil {
		return fmt.Sprintf("--reuse-home: cannot hash the install's %s: %v", sysPropsRel, err)
	}
	switch {
	case !strings.EqualFold(m.install, p.install.win):
		return fmt.Sprintf("--reuse-home %s: marker install %s is not this run's --install %s", dir.win, m.install, p.install.win)
	case m.sysSum != sum:
		return fmt.Sprintf("--reuse-home %s: marker system.properties sha256 %s is not the install's now (%s)", dir.win, m.sysSum, sum)
	case !namesDir(m.runID, dir.wsl):
		return fmt.Sprintf("--reuse-home %s: marker run id %q does not name this dir", dir.win, m.runID)
	case m.log == "":
		return fmt.Sprintf("--reuse-home %s: marker log is empty: the run that took this dir did not finish with --keep-home", dir.win)
	}
	if why := h.checkKeptLog(p, m); why != "" {
		return fmt.Sprintf("--reuse-home %s: %s", dir.win, why)
	}
	p.dir = dir
	return ""
}

// checkKeptLog binds the marker to the kept run's own record: its log must be a run log directly in this run's
// --log-dir, and the run id in its header must be the marker's.
func (h host) checkKeptLog(p *plan, m marker) string {
	if p.logDir == "" || filepath.Dir(m.log) != filepath.Clean(p.logDir) || !keptLogName.MatchString(filepath.Base(m.log)) {
		return fmt.Sprintf("marker log %s is not a run log in this run's --log-dir %s", m.log, p.logDir)
	}
	id, err := logRunID(m.log)
	if err != nil {
		return fmt.Sprintf("marker log %s: %v", m.log, err)
	}
	if id != m.runID {
		return fmt.Sprintf("marker run id %q is not the kept run's (its log %s records %q)", m.runID, m.log, id)
	}
	return ""
}

// logRunID is the run id a run log's header records, read from at most its first logHeadMax bytes.
func logRunID(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(io.LimitReader(f, logHeadMax))
	sc.Buffer(make([]byte, 0, 4<<10), diagLineCap)
	for sc.Scan() {
		l := sc.Text()
		if strings.HasPrefix(l, "== ") {
			break
		}
		if id, ok := strings.CutPrefix(l, fieldRunID+": "); ok {
			return id, nil
		}
	}
	return "", fmt.Errorf("no %q line in its header", fieldRunID)
}

// takeHome binds the scratch dir to this run before anything is built: a fresh run id in the marker and its log
// empty, so a run that dies before cleanup leaves a dir no later --reuse-home accepts. A reuse run also empties
// the kept out\ (the old -output: dir would otherwise be judged).
func (h host) takeHome(p *plan) error {
	id, err := newRunID(time.Now(), p.module, p.dir.wsl)
	if err != nil {
		return err
	}
	sum, err := installSum(p.install.wsl)
	if err != nil {
		return err
	}
	p.runID, p.marker = id, marker{runID: id, install: p.install.win, sysSum: sum}
	if p.reuseHome != "" {
		if err := os.RemoveAll(p.sub("out")); err != nil {
			return err
		}
	}
	return writeMarker(p.dir.wsl, p.marker)
}
