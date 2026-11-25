package writeops

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/spy"
)

// tplDepsInstalled compares the manifest <dependencies> of a template with the station's loaded
// modules (spy sysManagers/moduleManager, the page n4_station_runtime reads). A module the
// station does not have is a refusal: the station's own installer fails the same way
// (ApplicationTemplateInstaller.checkStationForCompatibleModules: missingModules make the check
// fail, mismatchedModules only warn; TmplUtil.checkModuleDependencies builds both maps). A lower
// installed version is a note. A page the station refuses, or one that cannot prove a module
// absent, leaves the dependencies unchecked, said in a note.
func tplDepsInstalled(c plan.Client, deps []string) ([]any, error) {
	if len(deps) == 0 {
		return nil, nil
	}
	reader, ok := c.(interface {
		GetSpy(urlPath string, timeout float64, maxBytes int) (*box.File, error)
	})
	if !ok {
		return []any{"Module dependencies not checked: this client cannot read spy pages"}, nil
	}
	file, err := reader.GetSpy(spy.URLPath(tplModulePage), 60, 0)
	var be *box.Error
	if errors.As(err, &be) && be.Kind == box.KindHTTPStatus {
		return []any{"Module dependencies not checked: the station refused the module manager page (HTTP " +
			strconv.Itoa(be.Code) + ")"}, nil
	}
	if err != nil {
		return []any{"Module dependencies not checked: the module manager page could not be read: " + err.Error()}, nil
	}
	page, err := spy.Parse(string(file.Body), spy.Limits{MaxRows: tplMaxModules})
	if err != nil {
		return []any{"Module dependencies not checked: " + err.Error()}, nil
	}
	parts, found := tplLoadedParts(page)
	if found && len(parts) == 0 {
		return []any{"Module dependencies not checked: the module manager table has no row in its own layout"}, nil
	}
	if !found {
		return []any{"Module dependencies not checked: the module manager page holds no table with Module, " +
			"Vendor and Version columns (the layout measured on Niagara 4.15)"}, nil
	}
	missing, older, met := tplCompareDeps(deps, parts)
	cut := file.Truncated || slices.ContainsFunc(page.Cut, func(s string) bool {
		return s == spy.CutRows || s == spy.CutTables || s == spy.CutCells || s == spy.CutCell
	})
	if len(missing) > 0 && !cut {
		return nil, plan.ToolError("the station does not have the module(s) the template's manifest <dependencies> " +
			"name: " + strings.Join(missing, ", ") + "; install them, then deploy again")
	}
	notes := []any{"Module dependencies checked against the station's module manager: " + strconv.Itoa(met) +
		" of " + strconv.Itoa(len(deps)) + " installed"}
	if len(missing) > 0 {
		notes = []any{"Module dependencies not checked: the module manager page was cut, so " +
			strings.Join(missing, ", ") + " may be listed beyond the cut"}
	}
	for _, o := range older {
		notes = append(notes, o)
	}
	return notes, nil
}

const (
	tplModulePage = "sysManagers/moduleManager"
	tplMaxModules = 5000
)

// tplLoadedParts is the loaded part name -> version of the module manager page; found is false
// when no table has the Module, Vendor and Version columns.
func tplLoadedParts(page *spy.Page) (map[string]string, bool) {
	parts, found := map[string]string{}, false
	for _, tb := range page.Tables {
		name, version := slices.Index(tb.Header, "Module"), slices.Index(tb.Header, "Version")
		if name < 0 || version < 0 || !slices.Contains(tb.Header, "Vendor") {
			continue
		}
		found = true
		for _, row := range tb.Rows {
			if len(row) == len(tb.Header) {
				parts[row[name]] = row[version]
			}
		}
	}
	return parts, found
}

// tplCompareDeps is the dependencies with no loaded part ("name (ver rel)"), the notes of those
// loaded at a lower version, and how many are loaded. A dependency names a part (control-rt) or a
// module (baja, met by any of its parts: baja-rt); the highest version of its parts counts.
func tplCompareDeps(deps []string, parts map[string]string) (missing []string, older []string, met int) {
	for _, dep := range deps {
		name, need, rel := tplDepFields(dep)
		have, ok := "", false
		for part, v := range parts {
			if part == name || strings.HasPrefix(part, name+"-") {
				if !ok || tplVersionLess(have, v) {
					have = v
				}
				ok = true
			}
		}
		spec := strings.TrimSpace(need + " " + rel)
		if !ok {
			if spec != "" {
				name += " (" + spec + ")"
			}
			missing = append(missing, name)
			continue
		}
		met++
		if need != "" && tplVersionLess(have, need) {
			older = append(older, name+" "+have+" is installed, the manifest needs "+spec+
				" (the station's installer only warns of this)")
		}
	}
	return missing, older, met
}

// tplDepFields splits a manifest dependency "name vendorVersion rel" (the version is absent when
// the manifest leaves it out).
func tplDepFields(dep string) (name, version, rel string) {
	f := strings.Fields(dep)
	if len(f) == 0 {
		return "", "", ""
	}
	name, f = f[0], f[1:]
	if len(f) > 0 && f[0] != "" && f[0][0] >= '0' && f[0][0] <= '9' {
		version, f = f[0], f[1:]
	}
	return name, version, strings.Join(f, " ")
}

// tplVersionLess is whether version a is lower than b, comparing the dot-separated numbers one
// by one (a missing one is 0); a version that is not numbers compares as not lower.
func tplVersionLess(a, b string) bool {
	x, y := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < max(len(x), len(y)); i++ {
		p, q := 0, 0
		var errP, errQ error
		if i < len(x) {
			p, errP = strconv.Atoi(x[i])
		}
		if i < len(y) {
			q, errQ = strconv.Atoi(y[i])
		}
		if errP != nil || errQ != nil {
			return false
		}
		if p != q {
			return p < q
		}
	}
	return false
}
