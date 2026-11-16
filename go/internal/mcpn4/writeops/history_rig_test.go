package writeops_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// histClient is typed plus the station's ORD servlet: GetOrd answers ids, the CSV of the
// station-wide history id query, and records each ORD it was asked.
type histClient struct {
	*typed
	ids    string
	idsErr error
	ords   []string
}

func (c *histClient) GetOrd(ord string, _ float64, _ int) (string, bool, error) {
	c.ords = append(c.ords, ord)
	return c.ids, false, c.idsErr
}

func (c *histClient) Close() {}

// noIds is the station's answer when no history extension exists yet.
const noIds = "Slot Path,Id\n"

// pointTypes are the registry entries of the point types the tests use, beside the recorded
// history extension contracts.
var pointTypes = [][2]string{{"bacnet:BacnetNumericPoint", "control:NumericPoint"},
	{"control:BooleanWritable", "control:BooleanPoint"}, {"control:BooleanPoint", "baja:Component"},
	{"control:EnumWritable", "control:EnumPoint"}, {"control:EnumPoint", "baja:Component"},
	{"control:StringPoint", "baja:Component"}, {"baja:Folder", "baja:Component"}}

// histPoints serves the given ORD -> nodes map, the point types and an empty history id table.
func histPoints(t *testing.T, kv ...any) *histClient {
	t.Helper()
	c := pointsClient(t, kv...)
	types := c.types.(*pyjson.Object)
	for _, p := range pointTypes {
		types.Set(p[0], obj("p", p[1]))
	}
	return &histClient{typed: c, ids: noIds}
}

// typedPoint is a point of type typ with handle h.
func typedPoint(typ, h string) *pyjson.Object { return obj("", obj("h", h, "t", typ)) }

var histSession = &session.Session{StationName: "FakeStation", BaseURL: "http://fake.invalid",
	SessionID: strings.Repeat("0", 32)}

// planHistory is the dry-run plan of n4_add_history.
func planHistory(c *histClient, args *pyjson.Object) (*plan.Planned, error) {
	return writeops.AddHistory.PlanWith(c, args, nil, histSession)
}

func wantRefusal(t *testing.T, name string, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("%s: err = %v, want %q", name, err, want)
	}
}

var errNoServlet = errors.New("ord servlet answered 404")

// without is a copy of o without key.
func without(o *pyjson.Object, key string) *pyjson.Object {
	out := pyjson.NewObject()
	for _, k := range o.Keys() {
		if k != key {
			v, _ := o.Get(k)
			out.Set(k, v)
		}
	}
	return out
}
