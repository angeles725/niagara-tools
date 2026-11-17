package scaffoldmodule

// moduleType is one --type: the bundled fixture it copies and the link-in slots of the fixture's
// component, which the station-test overlay tests ("" = the type gets no overlay).
type moduleType struct {
	name        string
	fixture     string
	linkInSlots string
}

// moduleTypes is the --type table. The driver fixture ships its own loopback station test, so it has no
// overlay (types/driver-authoring.md § 12.1).
var moduleTypes = []moduleType{
	{name: "logic", fixture: "MinimalPan", linkInSlots: `"setpoint", "interval"`},
	{name: "dashboard", fixture: "MinimalDash", linkInSlots: `"setpoint", "status"`},
	{name: "driver", fixture: "MinimalDriver"},
}

// Option names. --vendor, --target-version and --plugin-version are written into the emitted tree
// (checkValues gates them); --type only selects a fixture.
const (
	optVendor        = "--vendor"
	optTargetVersion = "--target-version"
	optPluginVersion = "--plugin-version"
	optType          = "--type"
)

// Option defaults. --vendor has none: the scaffold never guesses the module's identity.
const (
	defaultType          = "logic"
	defaultTargetVersion = "4.14"
	defaultPluginVersion = "7.6.17"
)

// pluginForTarget is the Niagara gradle plugin a release ships in <niagara_home>/etc/m2, the default
// --plugin-version for that --target-version (read from 4.13.2, 4.14 and 4.15 installations); a release not
// listed keeps defaultPluginVersion.
var pluginForTarget = map[string]string{"4.13": "7.3.40", "4.14": "7.6.17", "4.15": "7.6.22"}

// What the fixtures are written with; a run replaces each by the module's value.
const (
	fixtureVendor        = "Angeles"
	fixtureVendorLower   = "angeles"
	fixtureTargetVersion = "4.14"
	fixturePluginVersion = "7.6.17"
)

// Tokens of templates/station-test (types/moduleTest.md).
const (
	templateModule      = "ModuleA"
	templateVendor      = "Example"
	templatePackage     = "com.example."
	templatePackagePath = "com/example/"
	templateLinkInSlots = "@LINK_IN_SLOTS@"
)

// Kit paths below <toolbelt>/.. and emitted paths below the destination.
const (
	stationTestDir     = "templates/station-test"
	stationTestProbe   = "moduleTest-include.xml" // the file whose presence says the template is bundled
	ledgerTemplatePath = "templates/VERSION-LEDGER.md"
	ledgerDest         = "odd/VERSION-LEDGER.md"
	ledgerSampleRow    = "| <Module> | <x.y.z> |" // a line starting with this is the template's sample row
	rootWrapperScript  = "gradlew"                // emitted executable
)

func lookupType(name string) (moduleType, bool) {
	for _, t := range moduleTypes {
		if t.name == name {
			return t, true
		}
	}
	return moduleType{}, false
}
