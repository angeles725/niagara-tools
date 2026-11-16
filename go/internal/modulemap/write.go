package modulemap

import (
	"os"
	"path/filepath"
)

// mkdirs creates dir and its missing parents like os.MkdirAll, appending each directory it created to created,
// shallowest first.
func mkdirs(dir string, created *[]string) error {
	var missing []string
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Lstat(d); err == nil {
			break
		}
		missing = append(missing, d)
		if filepath.Dir(d) == d {
			break
		}
	}
	for i := len(missing) - 1; i >= 0; i-- {
		if err := os.Mkdir(missing[i], 0o755); err != nil {
			return err
		}
		*created = append(*created, missing[i])
	}
	return nil
}

// writeTemp writes data to the temp file tmp; when the write or close fails after tmp was opened, it removes tmp,
// so a failed run leaves no MODULE-MAP.md.tmp. A tmp it could not open is left as it was (not this run's).
func writeTemp(tmp string, data []byte) error {
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		// Mutation: MMA-tmp -- a failed write leaves MODULE-MAP.md.tmp behind
		_ = os.Remove(tmp)
	}
	return werr
}
