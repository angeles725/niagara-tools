package scaffoldmodule

import "strings"

// rule replaces every occurrence of from by to. The rules of one list run in a single pass over the text:
// at each position the first rule whose token matches wins (so the more specific token comes first), and
// what a rule writes is never read again. A value is therefore inserted exactly as typed, whatever it holds.
type rule struct{ from, to string }

func apply(s string, rules []rule) string {
	pairs := make([]string, 0, 2*len(rules))
	for _, r := range rules {
		pairs = append(pairs, r.from, r.to)
	}
	return strings.NewReplacer(pairs...).Replace(s)
}

// names holds everything a run substitutes; the option values are literal text.
type names struct {
	module, moduleLower, moduleSym    string // DemoRoomPan, demoRoomPan, drp
	fixture, fixtureLower, fixtureSym string // MinimalPan, minimalPan, mp
	vendor, vendorLower               string // Acme, acme
	targetVersion, pluginVersion      string
}

// newNames derives the names of a run from options checkValues has accepted.
func newNames(o options, typ moduleType) names {
	return names{
		module: o.module, moduleLower: lowerFirst(o.module), moduleSym: initials(o.module),
		fixture: typ.fixture, fixtureLower: lowerFirst(typ.fixture), fixtureSym: initials(typ.fixture),
		vendor: o.vendor, vendorLower: asciiLower(o.vendor),
		targetVersion: o.targetVersion, pluginVersion: o.pluginVersion,
	}
}

// profileRules rename the fixture's profile directories and artifact names, -ux and -wb before -rt.
func (n names) profileRules() []rule {
	return []rule{
		{n.fixture + "-ux", n.module + "-ux"},
		{n.fixture + "-wb", n.module + "-wb"},
		{n.fixture + "-rt", n.module + "-rt"},
	}
}

// fixturePathRules rewrite a fixture-relative path: profile names, the module name, the vendor package.
func (n names) fixturePathRules() []rule {
	return append(n.profileRules(),
		rule{n.fixture, n.module},
		rule{fixtureVendorLower, n.vendorLower},
	)
}

// fixtureTextRules rewrite a fixture text file: the path rules plus the lowerCamel name, the display
// vendor, the palette namespace prefix (before = and :) and the two versions.
func (n names) fixtureTextRules() []rule {
	return append(n.profileRules(),
		rule{n.fixtureLower, n.moduleLower},
		rule{n.fixture, n.module},
		rule{asciiLower(n.fixture), asciiLower(n.module)}, // the dashboard servlet's mount, "minimaldash"
		rule{fixtureVendor, n.vendor},
		rule{fixtureVendorLower, n.vendorLower},
		rule{n.fixtureSym + "=", n.moduleSym + "="},
		rule{n.fixtureSym + ":", n.moduleSym + ":"},
		rule{fixturePluginVersion, n.pluginVersion},
		rule{fixtureTargetVersion, n.targetVersion},
	)
}

// templatePathRules rewrite a templates/station-test relative path.
func (n names) templatePathRules() []rule {
	return []rule{
		{templatePackagePath, "com/" + n.vendorLower + "/"},
		{templateModule, n.module},
	}
}

// templateTextRules rewrite a station-test template file for a type's link-in slots.
func (n names) templateTextRules(linkInSlots string) []rule {
	return []rule{
		{templateLinkInSlots, linkInSlots},
		{templatePackage, "com." + n.vendorLower + "."},
		{templateModule, n.module},
		{templateVendor, n.vendor},
	}
}

// loginPathRules rewrite a templates/dashboard-login relative path (station-test's, plus the lower-case module
// name of the asset directory).
func (n names) loginPathRules() []rule {
	return append(n.templatePathRules(), rule{templateModuleLower, asciiLower(n.module)})
}

// loginTextRules rewrite a templates/dashboard-login text file (the dashboard has no link-in token there); a
// fragment may name a source path (the module map row), so the package path is a rule too.
func (n names) loginTextRules() []rule {
	return append(n.templateTextRules(""), rule{templatePackagePath, "com/" + n.vendorLower + "/"},
		rule{templateModuleLower, asciiLower(n.module)}, rule{templateSym, n.moduleSym})
}
