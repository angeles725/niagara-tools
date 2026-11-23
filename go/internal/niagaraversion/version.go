// Package niagaraversion answers one question before a module jar goes into a Niagara installation: is that
// installation the Niagara version the module was compiled against? A module compiled against 4.14 and copied
// into a 4.15 installation (or the other way round) loads wrongly or not at all, and nothing at the copy says so
// (retro deploy-target-version Δ2).
//
// What each side records:
//   - an installation: modules/baja.jar's META-INF/module.xml carries vendorVersion="4.14.0.162", the full
//     version of that installation (its directory name is only a convention of whoever installed it);
//   - a built module jar: the niagara-module plugin writes, for every Niagara module the code depends on, a
//     <dependency name=".." vendor="Tridium" vendorVersion="4.14"/> floor of major.minor in META-INF/module.xml.
//     The highest such floor is the version the module was compiled against.
//
// Consumers: check-target-version (toolbelt/check-target-version.sh), ng-deploy (before the copy) and build.sh
// (the line that names the installation a mirror was made from). Stdlib plus vendorversion only.
package niagaraversion

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/vendorversion"
)

// BajaJar is the module whose version is the version of the installation holding it.
const BajaJar = "baja.jar"

// Tridium is the vendor of the Niagara modules; only their dependency floors name a Niagara version.
const Tridium = "Tridium"

// maxDescriptor bounds a module.xml read: a larger one is not a descriptor this package understands.
const maxDescriptor = 8 << 20

// ErrNotInstallation is returned by InstallVersion for a directory with no modules/baja.jar (or baja.jar of
// its own): there is no Niagara version to compare, which is not the same as one that cannot be read.
var ErrNotInstallation = errors.New("no baja.jar: not a Niagara installation")

// InstallVersion is the full Niagara version of the installation dir names, which is its root or its modules
// directory, and the root itself (cleaned). The version is the vendorVersion of baja.jar's module.xml.
func InstallVersion(dir string) (version, root string, err error) {
	dir = filepath.Clean(dir)
	jar, root := dir+"/modules/"+BajaJar, dir
	if st, statErr := os.Stat(jar); statErr != nil || !st.Mode().IsRegular() {
		jar, root = dir+"/"+BajaJar, filepath.Dir(dir)
		if st, statErr := os.Stat(jar); statErr != nil || !st.Mode().IsRegular() {
			return "", "", fmt.Errorf("%s: %w", dir, ErrNotInstallation)
		}
	}
	text, err := moduleXML(jar)
	if err != nil {
		return "", "", err
	}
	if version = vendorversion.ModuleXMLVersion(text); version == "" {
		return "", "", fmt.Errorf("%s: its module.xml has no vendorVersion", jar)
	}
	return version, root, nil
}

// CompiledFor is the Niagara version a built module jar was compiled against, as major.minor: the highest
// floor among its Tridium dependencies. A jar that cannot be read, that holds no module.xml or names no
// Tridium dependency with a numeric floor records no version, which is an error: never a guess.
func CompiledFor(jar string) (string, error) {
	text, err := moduleXML(jar)
	if err != nil {
		return "", err
	}
	dec := xml.NewDecoder(strings.NewReader(text))
	dec.Strict = false
	highest := ""
	for {
		tok, err := dec.Token()
		if err != nil {
			break // io.EOF, or a descriptor that stops making sense: what was read counts
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "dependency" {
			continue
		}
		var vendor, floor string
		for _, a := range start.Attr {
			switch a.Name.Local {
			case "vendor":
				vendor = a.Value
			case "vendorVersion":
				floor = a.Value
			}
		}
		line, ok := majorMinor(floor)
		if vendor != Tridium || !ok {
			continue
		}
		if c, comparable := vendorversion.Cmp(line, highest); highest == "" || (comparable && c > 0) {
			highest = line
		}
	}
	if highest == "" {
		return "", fmt.Errorf("%s: its module.xml names no Tridium dependency with a numeric vendorVersion", jar)
	}
	return highest, nil
}

// Match reports whether a module compiled against one version may go into an installation of the other: the
// major and the minor must be equal (4.14 and 4.14.0.162 are; 4.14 and 4.15.6.20 are not). A version that has
// no major.minor is an error.
func Match(compiled, installed string) (bool, error) {
	a, okA := majorMinor(compiled)
	b, okB := majorMinor(installed)
	switch {
	case !okA:
		return false, fmt.Errorf("compile version %q has no major.minor", compiled)
	case !okB:
		return false, fmt.Errorf("installation version %q has no major.minor", installed)
	}
	return a == b, nil
}

// MajorMinor is the major.minor of a version ("4.14.0.162" -> "4.14"), "" when it has none.
func MajorMinor(version string) string {
	line, _ := majorMinor(version)
	return line
}

// majorMinor cuts a dotted version to its first two numeric components ("4.014" -> "4.14").
func majorMinor(version string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(version), ".")
	if len(parts) < 2 {
		return "", false
	}
	var n [2]int
	for i := range n {
		digits := parts[i]
		if i == 1 { // "14-rc" and the like: the number ends where the digits do
			digits = strings.TrimRight(digits, "-+_abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
		}
		v, err := strconv.Atoi(digits)
		if err != nil || v < 0 {
			return "", false
		}
		n[i] = v
	}
	return fmt.Sprintf("%d.%d", n[0], n[1]), true
}

// moduleXML is the text of the jar's META-INF/module.xml, read in process (no unzip on the host is needed).
func moduleXML(jar string) (string, error) {
	zr, err := zip.OpenReader(jar)
	if err != nil {
		return "", fmt.Errorf("%s: not a readable jar (%v)", jar, err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != "META-INF/module.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("%s: META-INF/module.xml cannot be read (%v)", jar, err)
		}
		defer rc.Close()
		data, err := io.ReadAll(io.LimitReader(rc, maxDescriptor+1))
		switch {
		case err != nil:
			return "", fmt.Errorf("%s: META-INF/module.xml cannot be read (%v)", jar, err)
		case len(data) > maxDescriptor:
			return "", fmt.Errorf("%s: META-INF/module.xml is over %d bytes", jar, maxDescriptor)
		}
		return string(data), nil
	}
	return "", fmt.Errorf("%s: no META-INF/module.xml", jar)
}
