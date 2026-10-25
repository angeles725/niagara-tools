package kitticket

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// errText is the C-library message of err ("Permission denied"), as bash and coreutils print it.
func errText(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		err = pe.Err
	}
	return strerror.Text(err)
}

// create opens p as a bash `> p` redirection does.
func create(p string) (*os.File, error) {
	return os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
}

// writeFile is `{ ...; } > p`: the open error, or the content written.
func writeFile(p, content string) error {
	f, err := create(p)
	if err != nil {
		return err
	}
	_, err = f.WriteString(content)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// mkdirP is GNU `mkdir -p dir`: each missing component made in turn; on failure it prints mkdir's message
// naming the component that failed ("File exists" when dir itself is a non-directory) and returns false.
func mkdirP(dir string, errOut io.Writer) bool {
	parts := strings.Split(dir, "/")
	for i := range parts {
		if parts[i] == "" || parts[i] == "." {
			continue
		}
		p := strings.Join(parts[:i+1], "/")
		err := os.Mkdir(p, 0o777)
		if err == nil {
			continue
		}
		st, serr := os.Stat(p)
		if serr == nil && st.IsDir() {
			continue
		}
		if errors.Is(err, fs.ErrExist) && i < lastPart(parts) {
			err = syscall.ENOTDIR
		}
		fmt.Fprintf(errOut, "mkdir: cannot create directory '%s': %s\n", p, errText(err))
		return false
	}
	return true
}

// lastPart is the index of the last non-empty component.
func lastPart(parts []string) int {
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return i
		}
	}
	return 0
}

// mv is GNU `mv src dst` for one file: into dst when dst is a directory; on failure it prints mv's message and
// returns false.
func mv(src, dst string, errOut io.Writer) bool {
	if _, err := os.Lstat(src); err != nil {
		fmt.Fprintf(errOut, "mv: cannot stat '%s': %s\n", src, errText(err))
		return false
	}
	if st, err := os.Stat(dst); err == nil && st.IsDir() {
		dst += "/" + src[strings.LastIndex(src, "/")+1:]
	}
	if err := os.Rename(src, dst); err != nil {
		fmt.Fprintf(errOut, "mv: cannot move '%s' to '%s': %s\n", src, dst, errText(err))
		return false
	}
	return true
}
