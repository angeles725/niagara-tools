package verifymodule

import (
	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/driverscaffold"
)

// driverChecks are the rows driver mode prints, in order (types/driver-authoring.md is the doctrine).
var driverChecks = []string{"driver-scaffold"}

// Driver detection: a profile dir is a driver module when a class under its src extends a Baja network
// or device base (driverBases), directly or through another class of the same src. The source decides,
// not the build file: a *-rt that depends on driver/ndriver but declares no network or device (a helper
// library) gets no driver rows. A module with no such class prints no driver row at all, so every
// non-driver run is byte-identical to the gate without driver mode.
var (
	networkBases = []string{"BDeviceNetwork", "BLoadableNetwork", "BBasicNetwork", "BSerialNetwork", "BNNetwork"}
	deviceBases  = []string{"BDevice", "BLoadableDevice", "BBasicDevice", "BNDevice"}
)

// javaUnit is one parsed source file of the driver module: its javaView (raw, code, bare), the tokens
// of its code and its class declarations.
type javaUnit struct {
	path    string
	view    *javaView
	toks    []javasrc.Token
	classes []javasrc.Class
}

// driverModule is what driver mode knows of one profile dir: its parsed sources and the classes that are
// networks or devices.
type driverModule struct {
	pd       string
	units    []*javaUnit
	networks map[string]bool
	devices  map[string]bool
}

// loadDriver parses <pd>/src and reports whether it is a driver module; false too when a file cannot be
// read (the run then ends, exit 3).
func (r *run) loadDriver(pd string) (*driverModule, bool) {
	m := &driverModule{pd: pd}
	var all []javasrc.Class
	for _, f := range r.javaFiles(pd + "/src") {
		v, ok := r.java(f)
		if !ok {
			return nil, false
		}
		u := &javaUnit{path: f, view: v, toks: javasrc.Tokens(v.code)}
		u.classes = javasrc.Classes(u.toks)
		all = append(all, u.classes...)
		m.units = append(m.units, u)
	}
	m.networks, m.devices = javasrc.Descendants(all, networkBases...), javasrc.Descendants(all, deviceBases...)
	return m, len(m.networks)+len(m.devices) > 0
}

// driver is driver mode: the driver checks on a driver module's profile dir, and no row on any other.
func (r *run) driver() bool {
	if r.src == "" {
		return true
	}
	pd := r.src + "/" + basename(r.jar.path, ".jar")
	if !lint.IsDir(pd + "/src") {
		return true
	}
	m, ok := r.loadDriver(pd)
	if !ok {
		return true
	}
	pass := true
	for _, c := range []func(*driverModule) bool{r.driverScaffold} {
		if !c(m) {
			pass = false
		}
	}
	return pass
}

// driverScaffold is lint-driver-scaffold, always on for a driver module: a wizard stub left in a shipped
// driver (doPing that only calls pingOk() FAILs, the other placeholders WARN), at the lint's own severity.
// Doctrine: types/driver-authoring.md § 5 (doPing) and § 1.6 (job.submit(null)).
func (r *run) driverScaffold(m *driverModule) bool {
	rule := driverscaffold.Spec().Rule
	failed, rows := false, 0
	for _, u := range m.units {
		fds, _ := rule.Check(javasrc.NewFile(u.path, u.view.raw.lines))
		for _, fd := range fds {
			status := "WARN"
			if fd.Severity == lint.Fail {
				status, failed = "FAIL", true
			}
			r.row(status, "driver-scaffold", u.path+":"+itoa(fd.Line)+"  "+fd.Msg)
			rows++
		}
	}
	if rows == 0 {
		r.row("PASS", "driver-scaffold", "no NDriver wizard stub under "+m.pd+"/src")
	}
	return !failed
}
