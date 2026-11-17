package writeops

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The local persistence route: config.bog under --station-home, before and after the save.

// bogSettle (seconds) is how long the read-back waits before its one retry of a config.bog that
// is unreadable right after the save (overamps retro row M9): the station writes
// config.bog.working, moves config.bog to its backup, then renames the working file over it
// (baja Station.java:466-492), so the file can be briefly absent.
const bogSettle = 1.0

// bogPath is os.path.join(home, "config.bog") for the non-empty home a plan checks.
func bogPath(home string) string {
	if strings.HasSuffix(home, "/") {
		return home + "config.bog"
	}
	return home + "/config.bog"
}

// bogState is _bog_state: config.bog's mtime (st_mtime), size and sha256, nil when it cannot be
// read (an OSError). The file is stat'ed once open, not by path first: the same outcome bar a
// race.
func bogState(home string) *pyjson.Object {
	f, err := os.Open(bogPath(home))
	if err != nil {
		return nil
	}
	defer f.Close()
	st, err := f.Stat()
	digest := sha256.New()
	if err == nil {
		_, err = io.Copy(digest, f)
	}
	if err != nil {
		return nil
	}
	t := st.ModTime() // st_mtime is sec + nsec * 1e-9 as a double
	return obj("mtime", float64(t.Unix())+float64(t.Nanosecond())*1e-9, "size", st.Size(),
		"sha256", hex.EncodeToString(digest.Sum(nil)))
}

// bogPersisted polls config.bog until it is changed (sha256 or a later mtime), unreadable after
// its one retry, or the timeout passes: persisted, unknown or not. The first unreadable read
// waits bogSettle and reads again, once; the retry is in the evidence.
func bogPersisted(d saveData) (any, *pyjson.Object, string) {
	path := bogPath(d.home)
	deadline := d.w.Monotonic() + d.w.SaveTimeout
	var after *pyjson.Object
	retried := false
	for {
		after = bogState(d.home)
		if after == nil && !retried {
			retried = true
			d.w.Sleep(bogSettle)
			continue
		}
		if after == nil || changed(d.before, after) || d.w.Monotonic() >= deadline {
			break
		}
		d.w.Sleep(d.w.SaveInterval)
	}
	evidence := obj("path", path, "before", d.before, "after", nil)
	if retried {
		evidence.Set("retry", obj("after_s", bogSettle, "reason", "config.bog was unreadable right after the save "+
			"(a station renames config.bog.working over it); read again once"))
	}
	var persisted any = false
	verdict := "mismatch"
	switch {
	case after == nil:
		evidence.Set("hint", "config.bog became unreadable after the save and stayed so after one retry")
		persisted, verdict = "unknown", "unverified"
	case changed(d.before, after):
		persisted, verdict = true, "verified"
	}
	if after != nil { // a nil *Object would not be None
		evidence.Set("after", after)
	}
	return persisted, evidence, verdict
}

// changed is a new sha256 or a later mtime.
func changed(before, after *pyjson.Object) bool {
	return get(after, "sha256") != get(before, "sha256") || get(after, "mtime").(float64) > get(before, "mtime").(float64)
}
