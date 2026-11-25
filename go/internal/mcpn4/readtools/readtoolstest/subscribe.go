package readtoolstest

import (
	"sync"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
)

// sscLog keeps, per station, every component-space call a client sent ("sub <ord>", "unsub <ord>"),
// in order: station.go holds the other fakes and may not grow (filesize debt).
var sscLog = struct {
	sync.Mutex
	calls map[*Station][]string
}{calls: map[*Station][]string{}}

// SSCCalls is every sub and unsub the station received, as "<key> <ord>", in order.
func (st *Station) SSCCalls() []string {
	sscLog.Lock()
	defer sscLog.Unlock()
	return append([]string{}, sscLog.calls[st]...)
}

// Subscribe answers the component-space sub (#875) as box-rt does: every ORD must resolve (an
// ORD with no tree is the station's resolve failure).
func (c *client) Subscribe(ords []string) error {
	c.logSSC("sub", ords)
	for _, o := range ords {
		if _, ok := c.st.Trees[o]; !ok {
			return &box.Error{Kind: box.KindBox, Msg: "Unable to resolve " + o, Channel: box.Channel, Key: "callssc"}
		}
	}
	return nil
}

// Unsubscribe answers the component-space unsub: the station answers null.
func (c *client) Unsubscribe(ords []string) error {
	c.logSSC("unsub", ords)
	return nil
}

func (c *client) logSSC(key string, ords []string) {
	sscLog.Lock()
	defer sscLog.Unlock()
	for _, o := range ords {
		sscLog.calls[c.st] = append(sscLog.calls[c.st], key+" "+o)
	}
}

// DrainPendingEvents answers no event: the fake pushes no subscription events.
func (c *client) DrainPendingEvents() []any { return nil }
