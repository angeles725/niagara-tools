package readtools

import (
	"errors"
	"io/fs"
	"os"
	"regexp"
	"strings"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// bqlOutputFile is _OUTPUT_FILE: the output_file of n4_bql_query, a plain .csv or .json name.
var bqlOutputFile = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}\.(csv|json)$`)

// stateDir is ctx.state_dir, or the default when the context has none.
func (t *Tools) stateDir() string {
	if t.Ctx.StateDir != "" {
		return t.Ctx.StateDir
	}
	return safety.ExpandUser(safety.DefaultStateDir)
}

// outputPath is _output_path: <state-dir>/<subdir>/<name> for a plain new file name, else a
// ToolError before anything is read. The state dir must pass safety.CheckStateDir.
func (t *Tools) outputPath(name, subdir string, pattern *regexp.Regexp, kinds string) (string, error) {
	if !pattern.MatchString(name) || strings.Contains(name, "..") {
		return "", toolError("output_file must be a plain file name %s(letters, digits, '.', "+
			"'_', '-'), written under <state-dir>/%s/; got %s", kinds, subdir, pyjson.StrRepr(name))
	}
	dir := t.stateDir()
	if err := safety.CheckStateDir(dir); err != nil {
		return "", toolError("%s", err.Error())
	}
	path := safety.Join(dir, subdir, name)
	if _, err := os.Lstat(path); err == nil {
		return "", toolError("output_file %s already exists: choose another name (the server "+
			"never overwrites a result file)", path)
	}
	return path, nil
}

// writeFailed is the ToolError of an output file the OS refused: its strerror.
func writeFailed(path string, err error) error {
	why := "OSError"
	var errno syscall.Errno
	if errors.As(err, &errno) {
		why = strerror.Text(errno)
	}
	return toolError("output_file %s could not be written (%s)", path, why)
}

// isDir is os.path.isdir: false on any stat error.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// makedirs is os.makedirs(name, mode, exist_ok=True): missing parents get 0777 (less the
// umask), name gets mode.
func makedirs(name string, mode fs.FileMode) error {
	head := safety.Dirname(name)
	if tail := name[len(head):]; strings.Trim(tail, "/") != "" && head != "" && head != name {
		if _, err := os.Lstat(head); err != nil {
			if err := makedirs(head, 0o777); err != nil && !errors.Is(err, fs.ErrExist) {
				return err
			}
		}
	}
	if err := os.Mkdir(name, mode); err != nil && !isDir(name) {
		return err
	}
	return nil
}

// privateDirs is _private_dirs: the state dir and the result subdir of path, created 0700 when
// missing (an existing one is never chmodded).
func privateDirs(path string) error {
	for _, dir := range []string{safety.Dirname(safety.Dirname(path)), safety.Dirname(path)} {
		if isDir(dir) {
			continue
		}
		if err := makedirs(dir, 0o700); err != nil {
			return err
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}

// writePrivate creates path 0600 with O_EXCL (never over an existing file) and writes each
// chunk of write; on any failure the file is removed, so no truncated result is left behind.
func writePrivate(path string, write func(f *os.File) error) error {
	if err := privateDirs(path); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_CLOEXEC, 0o600)
	if err != nil {
		return err
	}
	err = write(f)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(path)
	}
	return err
}

// csvLine is one csv.writer row of the excel dialect: a cell holding , " \r or \n is quoted with
// its quotes doubled, a row of one empty cell is "", and the line ends in \r\n.
func csvLine(cells []string) string {
	if len(cells) == 1 && cells[0] == "" {
		return "\"\"\r\n"
	}
	out := make([]string, len(cells))
	for i, c := range cells {
		out[i] = c
		if strings.ContainsAny(c, ",\"\r\n") {
			out[i] = `"` + strings.ReplaceAll(c, `"`, `""`) + `"`
		}
	}
	return strings.Join(out, ",") + "\r\n"
}

// writeRows is _write_output: the rows as CSV (header first) or as the JSON of base, query,
// columns and rows, by the file's extension.
func writeRows(path, base, query string, columns []string, rows []*pyjson.Object) error {
	return writePrivate(path, func(f *os.File) error {
		if !strings.HasSuffix(path, ".csv") {
			data, err := pyjson.Marshal(obj("base", base, "query", query, "columns", columns, "rows", list(rows)), pyjson.Options{})
			if err == nil {
				_, err = f.Write(data)
			}
			return err
		}
		lines := [][]string{columns}
		for _, row := range rows {
			cells := make([]string, len(columns))
			for i, c := range columns {
				v, _ := row.Get(c)
				cells[i], _ = v.(string)
			}
			lines = append(lines, cells)
		}
		for _, cells := range lines {
			line := csvLine(cells)
			if err := pyjson.EncodeUTF8(line); err != nil {
				return err
			}
			if _, err := f.WriteString(line); err != nil {
				return err
			}
		}
		return nil
	})
}
