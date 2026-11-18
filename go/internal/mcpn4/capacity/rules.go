// Package capacity measures a station against its global capacity licence: per rule (devices,
// proxy points) the count the licence limits beside the limit, and whether a plan of additions
// fits. It counts as the station's own recount does (baja com.tridium.sys.metrics.Metrics: a
// component whose type is driver:Device is a device, driver:ProxyExt a point, unless its type's
// module is listed in the feature's excludedDevices / excludedPoints), with one BQL select per
// rule instead of a tree walk. Sub-group features (a per-driver capacity with modules=) are not
// read.
package capacity

// FeatureName is the licence feature that holds the global limits.
const FeatureName = "tridium:globalCapacity"

// Rule is one limited resource: its name in a reply, the key of a plan that adds to it, the
// feature attributes of its limit and of its excluded modules, and the select whose rows are
// the counted components' types.
type Rule struct{ Name, PlanKey, LimitKey, ExcludeKey, Query string }

// The rules read, in reply order.
var (
	Devices = Rule{"devices", "add_devices", "device.limit", "excludedDevices", "select type from driver:Device"}
	Points  = Rule{"points", "add_points", "point.limit", "excludedPoints", "select type from driver:ProxyExt"}
	Rules   = []Rule{Devices, Points}
)

// The base the selects run from, and the word a limit uses for none.
const (
	base    = "station:|slot:/"
	noLimit = "none"
)

// What a usage or a plan says when something is unknown (data).
const (
	noteLimitUnread   = "the licence states %s=%q, which is neither a whole number nor none: the limit is unknown"
	noteLimitCut      = "the %s feature lists more attributes than were read and %s is not among them: the limit is unknown"
	noteExcludedCut   = "the %s attribute was cut on the licence page: which modules it excludes is unknown, so the count is not taken"
	noteCountFailed   = "the count (%s) failed: %s"
	noteCountCut      = "the count was cut at the read cap: it is unknown, not a lower bound"
	noteCountShape    = "the count answer has no Type column"
	notePlanUnknown   = "%s: the count or the limit is unknown, so whether the plan fits is unknown"
	noteNoClient      = "this client cannot run BQL"
	noteLicenceUnread = "the licence manager page was not read: %s"
)
