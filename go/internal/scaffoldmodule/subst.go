package scaffoldmodule

import "strings"

// rule replaces every occurrence of from by to. Rules run in order, each on the result of the one before
// (one sed `-e s|from|to|g` each), so the more specific token comes first.
type rule struct{ from, to string }

func apply(s string, rules []rule) string {
	for _, r := range rules {
		s = strings.ReplaceAll(s, r.from, r.to)
	}
	return s
}

// value is one option value as sed replacement text; in expands it for the token it replaces.
type value struct{ option, raw string }

// names holds everything a run substitutes.
type names struct {
	module, moduleLower, moduleSym    string // DemoRoomPan, demoRoomPan, drp
	fixture, fixtureLower, fixtureSym string // MinimalPan, minimalPan, mp
	vendor, vendorLower               value
	targetVersion, pluginVersion      value
}

// newNames derives the names of a run and refuses an option value that cannot be substituted.
func newNames(o options, typ moduleType) (names, *failure) {
	// typed is each value as the user wrote it: the lowercased vendor is reported as --vendor's.
	typed := map[string]string{"--vendor": o.vendor, "--target-version": o.targetVersion, "--plugin-version": o.pluginVersion}
	n := names{
		module: o.module, moduleLower: lowerFirst(o.module), moduleSym: initials(o.module),
		fixture: typ.fixture, fixtureLower: lowerFirst(typ.fixture), fixtureSym: initials(typ.fixture),
		vendor:        value{"--vendor", o.vendor},
		vendorLower:   value{"--vendor", asciiLower(o.vendor)},
		targetVersion: value{"--target-version", o.targetVersion},
		pluginVersion: value{"--plugin-version", o.pluginVersion},
	}
	// Whether a value can be expanded does not depend on the token it replaces.
	for _, v := range []value{n.vendor, n.vendorLower, n.targetVersion, n.pluginVersion} {
		if _, ok := expandReplacement(v.raw, ""); !ok {
			return names{}, failf(ExitUsage, badValue, v.option, typed[v.option])
		}
	}
	return n, nil
}

// in is v expanded for token; newNames has already accepted every value.
func (v value) in(token string) string {
	text, _ := expandReplacement(v.raw, token)
	return text
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
		rule{fixtureVendorLower, n.vendorLower.in(fixtureVendorLower)},
	)
}

// fixtureTextRules rewrite a fixture text file: the path rules plus the lowerCamel name, the display
// vendor, the palette namespace prefix (before = and :) and the two versions.
func (n names) fixtureTextRules() []rule {
	return append(n.profileRules(),
		rule{n.fixtureLower, n.moduleLower},
		rule{n.fixture, n.module},
		rule{fixtureVendor, n.vendor.in(fixtureVendor)},
		rule{fixtureVendorLower, n.vendorLower.in(fixtureVendorLower)},
		rule{n.fixtureSym + "=", n.moduleSym + "="},
		rule{n.fixtureSym + ":", n.moduleSym + ":"},
		rule{fixturePluginVersion, n.pluginVersion.in(fixturePluginVersion)},
		rule{fixtureTargetVersion, n.targetVersion.in(fixtureTargetVersion)},
	)
}

// templatePathRules rewrite a templates/station-test relative path.
func (n names) templatePathRules() []rule {
	return []rule{
		{templatePackagePath, "com/" + n.vendorLower.in(templatePackagePath) + "/"},
		{templateModule, n.module},
	}
}

// templateTextRules rewrite a station-test template file for a type's link-in slots.
func (n names) templateTextRules(linkInSlots string) []rule {
	return []rule{
		{templateLinkInSlots, linkInSlots},
		{templatePackage, "com." + n.vendorLower.in(templatePackage) + "."},
		{templateModule, n.module},
		{templateVendor, n.vendor.in(templateVendor)},
	}
}
