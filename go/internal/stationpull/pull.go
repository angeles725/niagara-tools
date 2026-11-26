package stationpull

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/platread"
)

type puller struct {
	s           cli.Stdio
	cr          platread.Cred
	exe         string
	o           options
	dest, local string
}

func (p puller) fget(shown bool) []string {
	args := append([]string{"fget", "-recurse", "-overwrite:none"}, p.cr.Conn(shown)...)
	return append(args, "~stations/"+p.o.station, p.local)
}

func (p puller) flist(shown bool) []string {
	return append(append([]string{"flist", "-recurse"}, p.cr.Conn(shown)...), "~stations/"+p.o.station)
}

// plat runs one plat call and returns its cleaned, masked output. A non-zero exit is not an error:
// plat's exit code does not say whether the copy worked.
func (p puller) plat(args []string) (string, error) {
	out, err := exec.Command(p.exe, args...).CombinedOutput()
	if _, ok := err.(*exec.ExitError); ok {
		err = nil
	}
	if err != nil {
		err = fmt.Errorf("%s", p.cr.Mask(err.Error()))
	}
	return p.cr.Mask(platread.Clean(string(out))), err
}

func (p puller) fail(format string, a ...any) int {
	fmt.Fprintln(p.s.Err, "station-pull: "+p.cr.Mask(fmt.Sprintf(format, a...)))
	return 1
}

func (p puller) pull() int {
	if err := os.MkdirAll(p.dest, 0o755); err != nil {
		return p.fail("%v", err)
	}
	root := filepath.Join(p.dest, "station")
	copyOut, err := p.plat(p.fget(false))
	if err != nil {
		return p.fail("cannot run plat: %v", err)
	}
	files, _ := copied(root)
	if len(files) == 0 {
		why := "plat copied no files into " + root
		if f := platread.Failure(copyOut); f != "" {
			why += " (" + f + ")"
		}
		return p.fail("%s", why)
	}
	listOut, err := p.plat(p.flist(false))
	if err != nil {
		return p.fail("cannot run plat: %v", err)
	}
	rows, bad := parseFlist(listOut)
	if len(bad) > 0 {
		return p.fail("plat flist gave %d rows with an unreadable CRC: %s", len(bad), strings.Join(bad, ", "))
	}
	if len(rows) == 0 {
		return p.fail("plat flist listed no files, so the %d copied files cannot be verified", len(files))
	}
	verified, problems := Verify(root, rows)
	if len(problems) > 0 {
		return p.fail("%d of %d listed files failed verification:\n  %s", len(problems), len(rows),
			strings.Join(problems, "\n  "))
	}
	if verified == 0 {
		return p.fail("plat flist gave no file a CRC, so no copied file was verified; no archive written")
	}
	archive := filepath.Join(p.dest, p.o.station+"-"+now().UTC().Format("20060102T150405Z")+".zip")
	zipped, excluded, err := Zip(root, archive, p.o.station, p.o.ex)
	if err != nil {
		return p.fail("zip: %v", err)
	}
	info, err := os.Stat(archive)
	if err != nil {
		return p.fail("%v", err)
	}
	fmt.Fprintf(p.s.Out, "station-pull: %s copied=%d verified=%d zipped=%d bytes=%d excluded=%d\n",
		archive, len(files), verified, zipped, info.Size(), excluded)
	return 0
}
