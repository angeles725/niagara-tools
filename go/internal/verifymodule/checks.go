package verifymodule

// registered is one check of the gate: the name its toolbelt/verify-module.sh header entry carries (the row
// name it prints; "driver" prints the driverChecks rows) and its check_* function.
type registered struct {
	name string
	run  check
}

// checks is every check the gate runs on each jar, in the order it runs them (the bash loop's order). The
// header of toolbelt/verify-module.sh lists the same names in the same order: TestHeaderListsEveryCheck.
var checks = []registered{
	{"bytecode", (*run).bytecode}, {"signed", (*run).signed}, {"types", (*run).types}, {"baja", (*run).baja},
	{"stored", (*run).storedCheck}, {"typecount", (*run).typeCount}, {"facets", (*run).facets},
	{"facets-req", (*run).facetPresence}, {"ord-literal", (*run).ordLiteral}, {"rcbackup", (*run).rcBackup},
	{"palette", (*run).palette}, {"wb-scaffold", (*run).wbScaffold}, {"phantom-dep", (*run).phantomDep},
	{"moduletest", (*run).moduleTest}, {"cross-module-type", (*run).crossModuleType},
	{"transient-operator", (*run).transientOperator}, {"compact3-import", (*run).compact3Imports},
	{"subscription-leak", (*run).subscriptionLeak}, {"slot-wall", (*run).slotWall},
	{"station-save", (*run).stationSave}, {"driver", (*run).driver},
}
