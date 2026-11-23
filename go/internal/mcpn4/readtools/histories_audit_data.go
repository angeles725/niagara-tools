package readtools

// The defaults, caps and messages of n4_histories_audit, as data.

// Defaults of n4_histories_audit (the registry schema bounds each argument).
const (
	haDefaultStaleFactor  = 3.0
	haDefaultCovMaxAge    = "24h"
	haDefaultStaleMin     = "10m"
	haDefaultCapacityWarn = 0.9
	haDefaultMaxHistories = 500
	haDefaultTimeout      = 60
	// haMaxPoints caps the control points read for the no-history check.
	haMaxPoints = 20000
	// haWorkers is how many history reads run at once: enough to cut the audit of a large station
	// from minutes to under one, few enough not to load the station.
	haWorkers = 4
	// haListCap caps each finding list; counts holds the full numbers.
	haListCap = 100
)

// haExtFields and haPointFields name the columns of the two reads by position (a localized
// station translates headers, and every encodeToString column is headed alike).
var (
	haExtFields   = []string{"path", "type", "enabled", "id", "capacity", "policy", "interval", "last"}
	haPointFields = []string{"path"}
)

// haLists are the finding lists of the reply, in order.
var haLists = []string{"no_history", "disabled", "stale", "near_capacity", "colliding_ids"}

// Messages of n4_histories_audit.
const (
	haBadBase      = "base must be a plain slot path such as station:|slot:/Drivers (default: the whole station), got %s"
	haBadSpan      = "%s must be a span such as 30m, 24h or 7d, got %s"
	haExtFailed    = "the history extensions could not be read (%s): %s"
	haPointsFailed = "the control points could not be read (%s): %s; no_history is not checked"
	haCutAbsence   = "the %s read stopped at %d rows (max_histories or the response cap): a point's missing " +
		"history cannot be asserted; narrow base or raise max_histories"
	haCutRows   = "the history extensions read stopped at %d rows: raise max_histories or narrow base to see the rest"
	haCountNote = "%d record count read(s) failed, those histories are not in near_capacity; first: %s"
	haNoAnswer  = "cells the station answered in no known form, per column; those extensions are left out of the " +
		"checks that need them"
	haListsCut = "lists capped at 100 items (counts holds the totals): %s"
)
