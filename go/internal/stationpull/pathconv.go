package stationpull

import (
	"fmt"
	"regexp"
	"strings"
)

var drivePath = regexp.MustCompile(`^/mnt/([A-Za-z])(/.*)?$`)

// NativePath is p as the local path the plat executable understands. plat.exe (run under WSL) is a
// Windows program: /mnt/c/a/b becomes C:\a\b, and a path outside a /mnt/<drive> mount is an error because
// plat.exe cannot write there. Any other plat gets p unchanged.
func NativePath(p, exe string) (string, error) {
	if !strings.HasSuffix(strings.ToLower(exe), ".exe") {
		return p, nil
	}
	m := drivePath.FindStringSubmatch(p)
	if m == nil {
		return "", fmt.Errorf("%s is not under /mnt/<drive>: plat.exe can only write to a Windows drive", p)
	}
	return strings.ToUpper(m[1]) + `:` + strings.ReplaceAll(m[2], "/", `\`) + tail(m[2]), nil
}

func tail(rest string) string {
	if rest == "" {
		return `\`
	}
	return ""
}
