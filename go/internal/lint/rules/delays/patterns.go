package delays

// maxCont is the continuation-line window of one Clock.schedule*( call.
const maxCont = 12

var (
	relLiteral = re(`BRelTime\.make(Seconds|Minutes|Hours|Days)?\([0-9]+\)`)
	facetMin   = re(`BFacets\.MIN,[[:space:]]*BRelTime\.make(Seconds|Minutes|Hours|Days)?\([0-9]+\)`)
	constDecl  = re(`static[[:space:]]+final[[:space:]]+BRelTime[[:space:]]+`)
	propDecl   = re(`static[[:space:]]+final[[:space:]]+Property[[:space:]]+`)
	slotName   = re(`^[a-z][A-Za-z0-9_]*`)
	propName   = re(`name[[:space:]]*=[[:space:]]*"[^"]*"`)
	getterDecl = re(`public[[:space:]]+BRelTime[[:space:]]+get[A-Za-z0-9_]+[[:space:]]*\(`)
	getterMods = re(`public[[:space:]]+BRelTime[[:space:]]+`)
	getterName = re(`^get[A-Za-z0-9_]+`)
	getQuoted  = re(`get\("[^"]+"\)`)
	getField   = re(`get\([a-z][A-Za-z0-9_]*\)`)
	callSite   = re(`Clock\.schedule(Periodically)?\(`)
	maxInMake  = re(`BRelTime\.make\(Math\.max\(`)
	makeOpen   = re(`^BRelTime\.make\(`)
	qualCall   = re(`^[A-Za-z_][A-Za-z0-9_]*\.[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(`)
	bareCall   = re(`^[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(`)
	callTail   = re(`[[:space:]]*\($`)
	makeVar    = re(`BRelTime\.make\([^)]+\)`)
	maxBinding = re(`Math\.max\([^,]+,[[:space:]]*`)
	digits     = re(`^[0-9]+`)
	bareLong   = re(`^[0-9]+[lL]?$`)
	assign     = re(`^[[:space:]]*=[^=]`)
	getterCall = re(`^get[A-Za-z0-9_]+[[:space:]]*\([[:space:]]*\)`)
	getterArgs = re(`[[:space:]]*\(.*`)
)
