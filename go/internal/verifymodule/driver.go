package verifymodule

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/driverscaffold"
)

// driverChecks are the rows driver mode prints, in order (types/driver-authoring.md is the doctrine).
var driverChecks = []string{"driver-scaffold", "driver-comm-config", "driver-discovery-alive"}

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
	for _, c := range []func(*driverModule) bool{r.driverScaffold, r.driverCommConfig, r.driverDiscoveryAlive} {
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

// isDriverClassFile reports whether u declares a network or device class.
func (m *driverModule) isDriverClassFile(u *javaUnit) bool {
	for _, c := range u.classes {
		if m.networks[c.Name] || m.devices[c.Name] {
			return true
		}
	}
	return false
}

var (
	propNameRe = regexp.MustCompile(`\bname[[:space:]]*=[[:space:]]*"([^"]*)"`)
	propTypeRe = regexp.MustCompile(`\btype[[:space:]]*=[[:space:]]*"([^"]*)"`)
	propDefRe  = regexp.MustCompile(`\bdefaultValue[[:space:]]*=[[:space:]]*"([^"]*)"`)
	// bareCommRe is a framework comm config named by its simple or qualified name, never a subclass.
	bareCommRe = regexp.MustCompile(`^(new[[:space:]]+)?([A-Za-z_][A-Za-z0-9_]*\.)*(BUdpCommConfig|BTcpCommConfig)[[:space:]]*(\(|$)`)
)

// driverCommConfig FAILs a @NiagaraProperty of a network or device class whose value is a bare framework
// comm config: its defaultValue constructs BUdpCommConfig or BTcpCommConfig, or its type is one of them
// with no defaultValue. A bare config keeps the base message factory, whose NMessage throws on every
// received frame. Doctrine: types/driver-authoring.md § 1.9 (Subclass the config; never ship a bare
// BUdpCommConfig).
func (r *run) driverCommConfig(m *driverModule) bool {
	failed := false
	for _, u := range m.units {
		if !m.isDriverClassFile(u) {
			continue
		}
		javasrc.NiagaraProperties(u.view.code, func(text string, line int) {
			bare := ""
			if d := propDefRe.FindStringSubmatch(text); d != nil {
				if bareCommRe.MatchString(strings.TrimSpace(d[1])) {
					bare = strings.TrimSpace(d[1])
				}
			} else if ty := propTypeRe.FindStringSubmatch(text); ty != nil && bareCommRe.MatchString(ty[1]) {
				bare = ty[1]
			}
			if bare == "" {
				return
			}
			name := ""
			if n := propNameRe.FindStringSubmatch(text); n != nil {
				name = n[1]
			}
			r.row("FAIL", "driver-comm-config", u.path+":"+itoa(line)+"  slot "+name+" holds a bare "+bare+
				": subclass it with your message factories (types/driver-authoring.md § 1.9)")
			failed = true
		})
	}
	if !failed {
		r.row("PASS", "driver-comm-config", "no network or device slot holds a bare comm config under "+m.pd+"/src")
	}
	return !failed
}

// driverDiscoveryAlive FAILs a getDiscoveryObjects body (a BINDiscoveryHost: the network for devices, the
// point device ext for points) that never calls isAlive(): a canceled discovery job then keeps scanning.
// Read in the bare view, so an isAlive() in a comment or a string is no call. Doctrine:
// types/driver-authoring.md § 1.11 step 1 (return null at once when !prefs.getJob().isAlive()).
func (r *run) driverDiscoveryAlive(m *driverModule) bool {
	failed, hosts := false, 0
	for _, u := range m.units {
		for _, meth := range javasrc.Methods(u.view.code) {
			if meth.Name != "getDiscoveryObjects" {
				continue
			}
			hosts++
			if !hasCall(u.view.bare, meth.Start, meth.End, "isAlive") {
				r.row("FAIL", "driver-discovery-alive", u.path+":"+itoa(meth.Start)+
					"  getDiscoveryObjects never checks job.isAlive(): a canceled job keeps scanning (types/driver-authoring.md § 1.11)")
				failed = true
			}
		}
	}
	switch {
	case hosts == 0:
		r.row("SKIP", "driver-discovery-alive", "no getDiscoveryObjects under "+m.pd+"/src")
	case !failed:
		r.row("PASS", "driver-discovery-alive", itoa(hosts)+" getDiscoveryObjects check job.isAlive() under "+m.pd+"/src")
	}
	return !failed
}

// hasCall reports whether name( is called in the lines [from, to] (1-based, inclusive) of bare.
func hasCall(bare []string, from, to int, name string) bool {
	for i := from - 1; i < to && i < len(bare); i++ {
		for ln, off := bare[i], 0; ; {
			j := strings.Index(ln[off:], name+"(")
			if j < 0 {
				break
			}
			if j += off; j == 0 || !identChar(ln[j-1]) {
				return true
			}
			off = j + 1
		}
	}
	return false
}

func identChar(c byte) bool {
	return c == '_' || c == '$' || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9')
}
