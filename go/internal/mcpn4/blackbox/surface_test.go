package blackbox

// What the server under test reports and enforces (data only). Each literal is checked against
// the binary's own output by a test, so it cannot drift silently: the version and the protocol
// set against initialize, the tool lists against tools/list.

const serverVersion = "0.9.5"

// supportedProtocols are newest first.
var supportedProtocols = []string{"2025-06-18", "2025-03-26", "2024-11-05"}

// readTools are the tools a read-only server lists, in tools/list order.
var readTools = []string{
	"n4_connect",
	"n4_describe_session",
	"n4_navigate",
	"n4_read_slots",
	"n4_list_links",
	"n4_find_dangling_outputs",
	"n4_bql_query",
	"n4_inventory",
	"n4_read_file",
	"n4_list_px_views",
	"n4_px_summary",
	"n4_list_batches",
	"n4_session_retro_draft",
	"n4_spy_page",
	"n4_station_health",
	"n4_read_history",
	"n4_license_info",
	"n4_station_runtime",
	"n4_read_alarms",
	"n4_list_alarm_classes",
	"n4_event_report",
	"n4_commissioning_report",
	"n4_list_schedule_kinds",
	"n4_read_schedule",
	"n4_read_tags",
	"n4_neql_query",
	"n4_list_tag_dictionaries",
}
