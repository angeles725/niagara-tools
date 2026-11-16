package readtools

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// What the read tools share: the station client they need, the state they keep and the table
// that serves each tool by name.

// RootOrd is ROOT_ORD.
const RootOrd = "station:|slot:/"

// Client is what the tools need of a station client (*box.Client has it all).
type Client interface {
	Open() (any, error)
	LoadTree(ord string, depth int, opt box.LoadOptions) (*pyjson.Object, error)
	GetOrd(ord string, timeout float64, maxBytes int) (string, bool, error)
	GetFile(urlPath string, timeout float64, maxBytes int) (*box.File, error)
	BaseURL() string
	Close()
}

// aboutReader is a client that can read the oBIX About document; one that cannot is tier C.
type aboutReader interface {
	About(timeout float64, maxBytes int) (string, error)
}

// Factory is Context.client_factory: a client for cfg that calls onAuthError with each AuthError
// it raises. A client may ignore the hook: the dispatcher latches on the returned error too.
type Factory func(cfg box.Config, onAuthError func(*box.Error)) (Client, error)

// BoxFactory makes the real BOX client.
func BoxFactory(cfg box.Config, onAuthError func(*box.Error)) (Client, error) {
	c, err := box.New(cfg)
	if err != nil {
		return nil, err
	}
	c.OnAuthError = onAuthError
	return c, nil
}

// Tools is the state the read tools share.
type Tools struct {
	Ctx           *session.Context
	NewClient     Factory
	ServerVersion string
	// NiagaraHomes maps a station NAME to the Niagara installation directory the operator gave
	// for it (--niagara-home): where n4_license_info reads the licence files, when reachable.
	NiagaraHomes map[string]string
	// Audit is the write machinery's audit log (ctx.write.audit), nil in read-only mode.
	Audit interface {
		Append(entry *pyjson.Object) error
	}
	// undescribed is the types the station left out of a registry reply this session.
	undescribed undescribed
}

// Register serves the ported read tools on s, with t.Ctx as its session state.
func Register(s *jsonrpc.Server, t *Tools) {
	s.Ctx = t.Ctx
	s.Handlers["n4_connect"] = t.Connect
	s.Handlers["n4_describe_session"] = t.DescribeSession
	s.Handlers["n4_navigate"] = t.Navigate
	s.Handlers["n4_read_slots"] = t.ReadSlots
	s.Handlers["n4_list_links"] = t.ListLinks
	s.Handlers["n4_find_dangling_outputs"] = t.FindDanglingOutputs
	s.Handlers["n4_bql_query"] = t.BqlQuery
	s.Handlers["n4_read_history"] = t.ReadHistory
	s.Handlers["n4_read_alarms"] = t.ReadAlarms
	s.Handlers["n4_list_alarm_classes"] = t.ListAlarmClasses
	s.Handlers["n4_inventory"] = t.Inventory
	s.Handlers["n4_read_file"] = t.ReadFile
	s.Handlers["n4_list_px_views"] = t.ListPxViews
	s.Handlers["n4_px_summary"] = t.PxSummary
	s.Handlers["n4_list_batches"] = t.ListBatches
	s.Handlers["n4_session_retro_draft"] = t.SessionRetroDraft
	s.Handlers["n4_spy_page"] = t.SpyPage
	s.Handlers["n4_station_health"] = t.StationHealth
	s.Handlers["n4_license_info"] = t.LicenseInfo
	s.Handlers["n4_station_runtime"] = t.StationRuntime
	s.Handlers["n4_commissioning_report"] = t.CommissioningReport
	registerSchedules(s, t)
	s.Handlers["n4_read_tags"] = t.ReadTags
	s.Handlers["n4_neql_query"] = t.NeqlQuery
	s.Handlers["n4_list_tag_dictionaries"] = t.ListTagDictionaries
}

func toolError(format string, a ...any) error {
	return &jsonrpc.ToolError{Msg: fmt.Sprintf(format, a...)}
}
