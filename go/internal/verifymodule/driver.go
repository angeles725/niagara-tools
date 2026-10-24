package verifymodule

import (
	"os"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/agentonincludesync"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/driverscaffold"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// driverChecks are the rows driver mode prints, in order (types/driver-authoring.md is the doctrine).
var driverChecks = []string{"driver-scaffold", "driver-comm-config", "driver-discovery-alive", "driver-server-enabled", "driver-manager"}

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
	all      []javasrc.Class // every class of the src, for Descendants
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
	m.all = all
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
	for _, c := range []func(*driverModule) bool{r.driverScaffold, r.driverCommConfig, r.driverDiscoveryAlive, r.driverServerEnabled, r.driverManager} {
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

// serverOnRe is a setServerEnabled(true) call in the bare view.
var serverOnRe = regexp.MustCompile(`(^|[^A-Za-z0-9_$])setServerEnabled[[:space:]]*\([[:space:]]*true[[:space:]]*\)`)

// driverServerEnabled FAILs a BTcpCommConfig subclass that turns serverEnabled on by default: a property
// override with defaultValue "true", or a setServerEnabled(true) call in its file. A client driver that
// listens holds the device's port (Address already in use) and never connects. Doctrine:
// types/driver-authoring.md § 1.6 (Client drivers keep serverEnabled false).
func (r *run) driverServerEnabled(m *driverModule) bool {
	tcp := javasrc.Descendants(m.all, "BTcpCommConfig")
	failed, configs := false, 0
	for _, u := range m.units {
		mine := false
		for _, c := range u.classes {
			if tcp[c.Name] {
				mine = true
				configs++
			}
		}
		if !mine {
			continue
		}
		javasrc.NiagaraProperties(u.view.code, func(text string, line int) {
			n, d := propNameRe.FindStringSubmatch(text), propDefRe.FindStringSubmatch(text)
			if n != nil && n[1] == "serverEnabled" && d != nil && strings.TrimSpace(d[1]) == "true" {
				r.row("FAIL", "driver-server-enabled", u.path+":"+itoa(line)+
					"  serverEnabled defaults to true: a client driver keeps it false (types/driver-authoring.md § 1.6)")
				failed = true
			}
		})
		for i, ln := range u.view.bare {
			if serverOnRe.MatchString(ln) {
				r.row("FAIL", "driver-server-enabled", u.path+":"+itoa(i+1)+
					"  setServerEnabled(true): a client driver keeps serverEnabled false (types/driver-authoring.md § 1.6)")
				failed = true
			}
		}
	}
	switch {
	case configs == 0:
		r.row("SKIP", "driver-server-enabled", "no TCP comm config under "+m.pd+"/src")
	case !failed:
		r.row("PASS", "driver-server-enabled", itoa(configs)+" TCP comm config keeps serverEnabled false under "+m.pd+"/src")
	}
	return !failed
}

// groupUnit is one parsed source of the module group (every <module>-* part beside the profile dir).
type groupUnit struct {
	path    string
	classes []javasrc.Class
	agents  []agentonincludesync.AgentType
}

// group parses every part of the module the jar belongs to (<src>/<module>-*/src, the module being the jar
// basename up to its last "-") and maps each class's simple name to its type name from the parts'
// module-include.xml (<type class=... name=...>; a class not declared there maps to its name without the
// leading B). ok is false when an input cannot be read or parsed (the run then ends, exit 3).
func (r *run) group() (module string, units []groupUnit, typeName func(string) string, ok bool) {
	base := basename(r.jar.path, ".jar")
	module = base
	if i := strings.LastIndexByte(base, '-'); i > 0 {
		module = base[:i]
	}
	names := map[string]string{}
	entries, _ := os.ReadDir(r.src)
	for _, e := range entries {
		part := r.src + "/" + e.Name()
		if !e.IsDir() || (e.Name() != base && !strings.HasPrefix(e.Name(), module+"-")) {
			continue
		}
		if inc := part + "/module-include.xml"; lint.IsFile(inc) {
			data, ok := r.readFile(inc)
			if !ok {
				return "", nil, nil, false
			}
			doc, err := xmlsrc.Parse(inc, data)
			if err != nil {
				r.envError(err.Error())
				return "", nil, nil, false
			}
			for _, i := range doc.Find("type") {
				el := doc.Elements()[i]
				c, _ := el.Attr("class")
				n, _ := el.Attr("name")
				names[simpleName(c)] = n
			}
		}
		if !lint.IsDir(part + "/src") {
			continue
		}
		for _, f := range r.javaFiles(part + "/src") {
			v, ok := r.java(f)
			if !ok {
				return "", nil, nil, false
			}
			toks := javasrc.Tokens(v.code)
			units = append(units, groupUnit{path: f, classes: javasrc.Classes(toks), agents: agentonincludesync.AgentTypes(toks)})
		}
	}
	typeName = func(class string) string {
		if n, ok := names[class]; ok && n != "" {
			return n
		}
		return strings.TrimPrefix(class, "B")
	}
	return module, units, typeName, true
}

// simpleName is the last part of a binary class name (pkg.Outer$Inner -> Inner).
func simpleName(c string) string {
	return c[strings.LastIndexAny(c, ".$")+1:]
}

// driverManager checks the Workbench managers of a driver module group:
//   - every point manager (a BNPointManager or BPointManager subclass in any part) is @AgentOn one of this
//     driver's point device exts, else it never shows: types/driver-authoring.md § 1.6 (a point manager is
//     @AgentOn the <mod>:<X>PointDeviceExt and <mod>:<X>PointFolder);
//   - a BBasicNetwork driver ships a manager @AgentOn its network or device type: PD-01,
//     types/driver-authoring.md § 1.3. An NDriver network has the framework's device and point managers,
//     so PD-01 (which names BBasicNetwork) does not bind it.
func (r *run) driverManager(m *driverModule) bool {
	module, units, typeName, ok := r.group()
	if !ok {
		return true
	}
	var all []javasrc.Class
	for _, u := range units {
		all = append(all, u.classes...)
	}
	exts := javasrc.Descendants(m.all, "BNPointDeviceExt", "BPointDeviceExt")
	var extTargets []string
	for _, c := range m.all {
		if exts[c.Name] {
			extTargets = append(extTargets, module+":"+typeName(c.Name))
		}
	}
	extTargets = sortU(extTargets)
	managers := javasrc.Descendants(all, "BNPointManager", "BPointManager")
	failed := false
	var held []string
	for _, u := range units {
		for _, c := range u.classes {
			if !managers[c.Name] || len(extTargets) == 0 {
				continue
			}
			var on []string
			for _, a := range u.agents {
				if simpleName(a.Class) == c.Name {
					on = append(on, a.Types...)
				}
			}
			if anyIn(on, extTargets) {
				held = append(held, "point manager "+c.Name+" @AgentOn "+strings.Join(on, ","))
				continue
			}
			r.row("FAIL", "driver-manager", u.path+":"+itoa(c.Line+1)+"  point manager "+c.Name+" is not @AgentOn "+
				strings.Join(extTargets, " or ")+": it never shows (types/driver-authoring.md § 1.6)")
			failed = true
		}
	}
	basic := javasrc.Descendants(m.all, "BBasicNetwork", "BSerialNetwork")
	if len(basic) > 0 {
		var targets []string
		for _, c := range m.all {
			if m.networks[c.Name] || m.devices[c.Name] {
				targets = append(targets, module+":"+typeName(c.Name))
			}
		}
		targets = sortU(targets)
		by := ""
		for _, u := range units {
			for _, a := range u.agents {
				if by == "" && anyIn(a.Types, targets) {
					by = simpleName(a.Class)
				}
			}
		}
		if by == "" {
			r.row("FAIL", "driver-manager", m.pd+"/src  PD-01: a BBasicNetwork driver ships no manager @AgentOn "+
				strings.Join(targets, " or ")+" (types/driver-authoring.md § 1.3)")
			failed = true
		} else {
			held = append(held, "PD-01 "+by+" @AgentOn the network or device")
		}
	} else {
		held = append(held, "PD-01 n/a (no BBasicNetwork: framework device manager)")
	}
	if !failed {
		r.row("PASS", "driver-manager", strings.Join(held, "; "))
	}
	return !failed
}

// anyIn reports whether some element of xs is in set.
func anyIn(xs, set []string) bool {
	for _, x := range xs {
		if contains(set, x) {
			return true
		}
	}
	return false
}
