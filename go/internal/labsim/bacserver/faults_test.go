package bacserver

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
)

func readRequest(invoke byte, object bacnet.ObjectID) []byte {
	return bacnet.ConfirmedRequest(bacnet.MaxAPDU, invoke, bacnet.ServiceReadProperty, bacnet.ReadProperty{Object: object, Ref: ref(bacnet.PropPresentValue)}.Bytes())
}

func TestAnOfflineDeviceAnswersNothing(t *testing.T) {
	t.Parallel()
	s := start(t, bench(false, 0), nil)
	c := s.client()
	s.advance(10 * time.Second)
	s.unicast(c, readRequest(1, supply))
	s.unicast(c, whoIs())
	s.frame(c, bacnet.RegisterForeignDevice, []byte{0, 60})
	s.await(3)
	s.advance(10 * time.Second)
	s.settle(c) // back online: the three frames above got nothing
	entries := s.logged()
	for i, e := range entries[:3] {
		if e.Result != reqlog.ResultNoResponse || e.Reason != reqlog.ReasonOffline {
			t.Errorf("entry %d while offline: %+v", i, e)
		}
	}
	if entries[0].Service != "read-property" || entries[0].Object != "analog-input,1" || entries[1].Service != "who-is" || entries[2].Link != "register-foreign-device" {
		t.Errorf("what was asked while offline is not in the log: %+v", entries[:3])
	}
	if got := s.server.Stats().ForeignDevices; got != 0 {
		t.Errorf("an offline device registered %d foreign devices", got)
	}
}

func TestAResponseIsHeldForTheLongestDelayItTouches(t *testing.T) {
	t.Parallel()
	s := start(t, bench(false, 100*time.Millisecond), nil)
	c := s.client()
	s.unicast(c, readRequest(1, supply))
	if p := s.recv(c); p.apdu.Type != bacnet.PDUComplexACK {
		t.Fatalf("%+v", p.apdu)
	}
	if e := s.last(); e.DelayMS != 100 {
		t.Errorf("the log says the response was held %d ms, want 100", e.DelayMS)
	}
	s.unicast(c, readRequest(2, fragile))
	if p := s.recv(c); p.apdu.Type != bacnet.PDUComplexACK || p.apdu.InvokeID != 2 {
		t.Fatalf("%+v", p.apdu)
	}
	s.unicast(c, whoIs())
	s.recv(c)
	if got := s.slept(); len(got) != 3 || got[0] != 100*time.Millisecond || got[1] != 300*time.Millisecond || got[2] != 100*time.Millisecond {
		t.Errorf("the responses were held %v, want 100ms, 300ms (the slow point) and 100ms (the I-Am)", got)
	}
}

func TestAFailedPointOrDeviceAnswersAnOperationalProblem(t *testing.T) {
	t.Parallel()
	s := start(t, bench(false, 0), nil)
	c := s.client()
	if got := s.read(c, fragile, ref(bacnet.PropPresentValue)); got[0].Real != 9 {
		t.Fatalf("the first read: %+v", got)
	}
	for name, request := range map[string][]byte{
		"a read": readRequest(11, fragile),
		"a read-multiple that touches it": bacnet.ConfirmedRequest(bacnet.MaxAPDU, 11, bacnet.ServiceReadPropertyMultiple, bacnet.ReadPropertyMultiple{
			{Object: supply, Refs: []bacnet.PropertyRef{ref(bacnet.PropPresentValue)}}, {Object: fragile, Refs: []bacnet.PropertyRef{ref(bacnet.PropUnits)}}}.Bytes()),
	} {
		s.unicast(c, request)
		p := s.recv(c)
		if p.apdu.Type != bacnet.PDUError || p.apdu.Err != *bacnet.ErrOperationalProblem {
			t.Errorf("%s of the failed point: %+v, want error device / operational-problem", name, p.apdu)
		}
		if e := s.last(); e.ErrorClass != "device" || e.ErrorCode != "operational-problem" {
			t.Errorf("%s: log entry %+v", name, e)
		}
	}
	if got := s.read(c, supply, ref(bacnet.PropPresentValue)); got[0].Real != 18.5 {
		t.Errorf("a point that has not failed: %+v", got)
	}
}

func TestAFloodIsDroppedWithoutGrowingAnything(t *testing.T) {
	t.Parallel()
	s := start(t, bench(false, 0), func(o *Options) { o.RateBurst, o.RatePerSecond = 5, 5 })
	c := s.client()
	for i := range 20 {
		s.unicast(c, readRequest(byte(i), supply))
	}
	for i := range 5 {
		if p := s.recv(c); p.apdu.InvokeID != byte(i) {
			t.Fatalf("answer %d has invoke id %d", i, p.apdu.InvokeID)
		}
	}
	s.await(20)
	s.advance(time.Second) // the clock is the test's: only now are 5 more allowed
	s.unicast(c, readRequest(100, supply))
	if p := s.recv(c); p.apdu.InvokeID != 100 {
		t.Errorf("after the flood the next answer has invoke id %d, want 100: the 15 over the rate got none", p.apdu.InvokeID)
	}
	if got := s.server.Stats(); got.RateLimited != 15 || got.Packets != 21 {
		t.Errorf("Stats() = %+v, want 15 of 21 packets dropped for the rate", got)
	}
	if got := len(s.logged()); got != 6 {
		t.Errorf("%d log entries, want 6: a dropped packet is counted, not logged", got)
	}
}

func TestAPacketOverTheLargestFrameIsDropped(t *testing.T) {
	t.Parallel()
	// The limit is lowered here: a datagram over the real one (1507 octets) needs
	// IP fragments, which not every loopback delivers.
	s := start(t, bench(false, 0), func(o *Options) { o.maxPacket = 200 })
	c := s.client()
	huge := bacnet.BVLL{Function: bacnet.OriginalUnicast, Data: make([]byte, 200)}.Bytes()
	s.raw(c, huge)
	s.settle(c)
	if got := s.server.Stats(); got.Oversized != 1 {
		t.Errorf("Stats() = %+v, want 1 oversized packet", got)
	}
}

func TestTheTablesOfSourcesStayBounded(t *testing.T) {
	t.Parallel()
	limit := newLimiter(5, 5, 3)
	for i := range 10 {
		limit.allow(netip.AddrFrom4([4]byte{192, 0, 2, byte(i + 1)}), 0)
	}
	if got := len(limit.sources); got > 3 {
		t.Errorf("the limiter tracks %d sources, want 3 at most", got)
	}
	seen := newDuplicates(4)
	for i := range 50 {
		seen.store(dupKey{invoke: byte(i)}, []byte{1}, []byte{2}, time.Duration(i)*time.Millisecond)
	}
	if got := len(seen.entries); got > 4 {
		t.Errorf("the duplicate table holds %d answers, want 4 at most", got)
	}
	if _, ok := seen.find(dupKey{invoke: 49}, []byte{1}, 49*time.Millisecond); !ok {
		t.Error("the newest answer was evicted")
	}
	if _, ok := seen.find(dupKey{invoke: 49}, []byte{9}, 49*time.Millisecond); ok {
		t.Error("another request under the same invoke id was taken for a retry")
	}
	if _, ok := seen.find(dupKey{invoke: 49}, []byte{1}, 49*time.Millisecond+duplicateWindow+1); ok {
		t.Error("an answer was replayed after the duplicate window")
	}
}

func TestHeldResponsesAreBounded(t *testing.T) {
	t.Parallel()
	release := make(chan struct{})
	s := start(t, bench(false, 50*time.Millisecond), func(o *Options) {
		o.Sleep = func(ctx context.Context, _ time.Duration) bool {
			select {
			case <-release:
				return true
			case <-ctx.Done():
				return false
			}
		}
	})
	c := s.client()
	for i := range maxDelayed + 1 {
		s.unicast(c, readRequest(byte(i), supply))
	}
	s.await(maxDelayed + 1)
	if e := s.last(); e.Result != reqlog.ResultNoResponse || e.Reason != reasonBusy || *e.InvokeID != maxDelayed {
		t.Errorf("the response past the %d held ones: %+v, want none", maxDelayed, e)
	}
	close(release)
	answered := map[byte]bool{}
	for range maxDelayed {
		answered[s.recv(c).apdu.InvokeID] = true
	}
	if len(answered) != maxDelayed || answered[maxDelayed] {
		t.Errorf("%d requests were answered once released (the one over the bound among them: %v)", len(answered), answered[maxDelayed])
	}
}
