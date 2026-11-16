package niagaratest

import (
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
