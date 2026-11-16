// Package spy reads the diagnostics pages of a station's spy tree as data: the rule of the
// paths a read may name, the HTML of a page as tables, links and text, and the Global Capacity
// table of the metrics page. It sends nothing: the caller fetches URLPath(path) with its session.
//
// The pages are the station's own HTML (measured on Niagara 4.15 at GET /ord?spy:/<path>): a
// navigation table, then tables, lists of links and POST forms for the actions. Their layout is
// not a contract and differs between versions, so a page is returned as found.
package spy

// Root is the URL of the spy index; a page is Root plus its escaped path.
const Root = "/ord?spy:/"

// linkRoots are the href prefixes of a link that stays under the spy root.
var linkRoots = []string{Root, "/ord/spy:/", "spy:/"}

// MaxPathLen is the longest path a read may name, in bytes.
const MaxPathLen = 512

// refusedChars are the characters no path segment may hold, each with the reason shown: a
// scheme or another ORD, a query string that could name an action, or an escape the station
// would decode again.
var refusedChars = []struct {
	chars, why string
}{
	{":", "a scheme or an ORD"},
	{"|", "an ORD chain"},
	{"?&=#", "a query string"},
	{"%", "a percent escape (give the plain name)"},
	{";", "a path parameter"},
	{"\\", "a backslash"},
}

// actionPaths are the pages that run an action instead of showing one (measured on the util
// page of Niagara 4.15, where they are POST forms): never fetched, compared without case.
var actionPaths = []string{"util/gc", "util/forceThreadDump"}

// The caps of one parsed page.
const (
	DefaultMaxRows = 500   // table rows, all tables together
	MaxLinks       = 500   // links to child pages
	MaxCellRunes   = 2000  // one table cell
	MaxTextRunes   = 20000 // the text outside tables
)

// notFoundHeading is the heading of the page a station answers (HTTP 200) for an unknown path.
const notFoundHeading = "Spy Not Found"

// The names of the caps a page reports in Cut.
const (
	CutRows  = "max_rows"
	CutLinks = "links"
	CutCell  = "cell"
	CutText  = "text"
)

// The tables and columns of the metrics page that Capacity reads, compared without case.
const (
	capacityTitle = "global capacity"
	recountTitle  = "recount"
	columnType    = "type"
	columnLimit   = "limit"
	columnUsed    = "used"
	noLimit       = "none"
)
