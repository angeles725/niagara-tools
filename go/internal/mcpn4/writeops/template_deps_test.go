package writeops_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// withModules is a station that also answers the spy module manager page (the part, vendor and
// version of each loaded module); a nil page is a station that refuses spy pages (403), and err a
// read that fails another way (a timeout).
type withModules struct {
	*typed
	page  *string
	paths []string
	err   error
}

func (c *withModules) GetSpy(urlPath string, _ float64, _ int) (*box.File, error) {
	c.paths = append(c.paths, urlPath)
	if c.err != nil {
		return nil, c.err
	}
	if c.page == nil {
		return nil, &box.Error{Kind: box.KindHTTPStatus, Code: 403, Msg: "HTTP 403 from station"}
	}
	return &box.File{Body: []byte(*c.page)}, nil
}

// modulePage is the module manager table of a station holding the given parts (name, version).
func modulePage(parts ...string) *string {
	var b strings.Builder
	b.WriteString("<html><body><table><tr><th>Module</th><th>Vendor</th><th>Version</th></tr>")
	for i := 0; i+1 < len(parts); i += 2 {
		b.WriteString("<tr><td>" + parts[i] + "</td><td>Tridium</td><td>" + parts[i+1] + "</td></tr>")
	}
	b.WriteString("</table></body></html>")
	s := b.String()
	return &s
}

func depsPlan(t *testing.T, page *string) (*withModules, error, string) {
	t.Helper()
	c := &withModules{typed: tplClient(tplTypes()), page: page}
	p, err := writeops.DeployTemplate.Plan(c, deployArgs(tplFile(t, nil, nil)))
	if err != nil {
		return c, err, ""
	}
	return c, nil, js(t, p.Notes)
}

// A manifest dependency the station does not have is refused in the dry run, each missing module
// named with what the manifest asks (the station's own installer fails the same way:
// checkStationForCompatibleModules, missingModules).
func TestADeployRefusesAMissingManifestDependency(t *testing.T) {
	c, err, _ := depsPlan(t, modulePage("baja-rt", "4.15.6.20"))
	if err == nil {
		t.Fatal("a deploy needing control-rt and template-rt planned on a station with baja only")
	}
	want := "the station does not have the module(s) the template's manifest <dependencies> name: " +
		"control-rt (4.15 minimum), template-rt (4.15 minimum); install them, then deploy again"
	if err.Error() != want {
		t.Errorf("error\n got  %s\n want %s", err, want)
	}
	if len(c.paths) != 1 || c.paths[0] != "/ord?spy:/sysManagers/moduleManager" {
		t.Errorf("spy reads %v, want one read of the module manager page", c.paths)
	}
	if len(c.calls) != 0 {
		t.Errorf("the type registry was asked %v after the dependency refusal", c.calls)
	}
}

// Every dependency installed (a bare module name is met by any of its parts) plans; a lower
// installed version is a note, not a refusal.
func TestADeployNotesALowerInstalledDependencyVersion(t *testing.T) {
	_, err, notes := depsPlan(t, modulePage("baja-rt", "4.15.6.20", "control-rt", "4.14.2", "template-rt", "4.15.0"))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{
		"Module dependencies checked against the station's module manager: 3 of 3 installed",
		"control-rt 4.14.2 is installed, the manifest needs 4.15 minimum (the station's installer only warns of this)",
	} {
		if !strings.Contains(notes, s) {
			t.Errorf("notes lack %q:\n%s", s, notes)
		}
	}
	if strings.Contains(notes, "baja-rt 4.15.6.20 is installed") || strings.Contains(notes, "template-rt 4.15.0 is installed") {
		t.Errorf("a version at or above the manifest's is noted:\n%s", notes)
	}
}

// A station that refuses the module manager page leaves the dependencies unchecked, said in a
// note: the deploy still plans (the type registry check stays).
func TestADeployNotesUncheckedDependenciesWhenTheModulePageIsRefused(t *testing.T) {
	_, err, notes := depsPlan(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notes, "Module dependencies not checked: the station refused the module manager page (HTTP 403)") {
		t.Errorf("notes lack the unchecked dependencies note:\n%s", notes)
	}
}

// A page with no module table (another layout) proves nothing: no refusal, a note.
func TestADeployDoesNotRefuseOnAnUnrecognizedModulePage(t *testing.T) {
	page := "<html><body><p>nothing here</p></body></html>"
	_, err, notes := depsPlan(t, &page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notes, "Module dependencies not checked: the module manager page holds no table with Module, Vendor and Version columns") {
		t.Errorf("notes lack the unrecognized page note:\n%s", notes)
	}
}

// A module table with no usable row (header only, or rows of another width) and a read that fails
// other than by an HTTP status prove nothing either: the deploy plans, with the unchecked note.
func TestADeployDoesNotRefuseOnAnEmptyTableOrAFailedRead(t *testing.T) {
	odd := "<html><body><table><tr><th>Module</th><th>Vendor</th><th>Version</th></tr>" +
		"<tr><td colspan=3>baja-rt</td></tr></table></body></html>"
	for name, c := range map[string]*withModules{
		"header only": {typed: tplClient(tplTypes()), page: modulePage()},
		"odd rows":    {typed: tplClient(tplTypes()), page: &odd},
		"timeout":     {typed: tplClient(tplTypes()), err: errors.New("read timed out")},
	} {
		p, err := writeops.DeployTemplate.Plan(c, deployArgs(tplFile(t, nil, nil)))
		if err != nil {
			t.Errorf("%s: refused: %v", name, err)
			continue
		}
		if notes := js(t, p.Notes); !strings.Contains(notes, "Module dependencies not checked") {
			t.Errorf("%s: notes lack the unchecked note:\n%s", name, notes)
		}
	}
}
