package mbserver

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
)

type exchange struct {
	name string
	req  []byte
	want []byte
}

func run(t *testing.T, steps []exchange) {
	t.Helper()
	c := start(t, nil).dial()
	for _, x := range steps {
		if got := ask(t, c, 1, x.req...); !bytes.Equal(got, x.want) {
			t.Errorf("%s: % X, want % X", x.name, got, x.want)
		}
	}
}

func TestEachReadFunctionAnswersTheScenarioValues(t *testing.T) {
	t.Parallel()
	run(t, []exchange{
		{"01 read coils", []byte{0x01, 0, 0, 0, 3}, []byte{0x01, 0x01, 0x05}},
		{"02 read discrete inputs", []byte{0x02, 0, 0, 0, 2}, []byte{0x02, 0x01, 0x01}},
		{"03 read holding registers", []byte{0x03, 0, 0, 0, 3}, []byte{0x03, 0x06, 0x00, 0xD7, 0x3F, 0xC0, 0x00, 0x00}},
		{"04 read input registers", []byte{0x04, 0, 0, 0, 1}, []byte{0x04, 0x02, 0xFF, 0xFE}},
	})
}

func TestEachWriteFunctionChangesWhatIsReadBack(t *testing.T) {
	t.Parallel()
	run(t, []exchange{
		{"05 write single coil", []byte{0x05, 0, 1, 0xFF, 0x00}, []byte{0x05, 0, 1, 0xFF, 0x00}},
		{"coils after 05", []byte{0x01, 0, 0, 0, 3}, []byte{0x01, 0x01, 0x07}},
		{"06 write single register", []byte{0x06, 0, 0, 0x01, 0x2C}, []byte{0x06, 0, 0, 0x01, 0x2C}},
		{"register after 06", []byte{0x03, 0, 0, 0, 1}, []byte{0x03, 0x02, 0x01, 0x2C}},
		{"15 write multiple coils", []byte{0x0F, 0, 0, 0, 2, 1, 0x02}, []byte{0x0F, 0, 0, 0, 2}},
		{"coils after 15", []byte{0x01, 0, 0, 0, 3}, []byte{0x01, 0x01, 0x06}},
		{"16 write multiple registers", []byte{0x10, 0, 1, 0, 2, 4, 0x40, 0x40, 0, 0}, []byte{0x10, 0, 1, 0, 2}},
		{"registers after 16", []byte{0x03, 0, 1, 0, 2}, []byte{0x03, 0x04, 0x40, 0x40, 0, 0}},
	})
}

func TestEachFaultAnswersItsException(t *testing.T) {
	t.Parallel()
	run(t, []exchange{
		{"01 illegal function", []byte{0x2B, 0x0E, 0x01, 0x00}, []byte{0xAB, 0x01}},
		{"02 an address with no point", []byte{0x03, 0, 3, 0, 1}, []byte{0x83, 0x02}},
		{"02 a range with a gap", []byte{0x03, 0, 0, 0, 4}, []byte{0x83, 0x02}},
		{"02 an input register asked as a holding one", []byte{0x03, 0, 9, 0, 1}, []byte{0x83, 0x02}},
		{"02 a write to a read-only register", []byte{0x06, 0, 5, 0, 1}, []byte{0x86, 0x02}},
		{"02 a write to a read-only coil", []byte{0x05, 0, 2, 0, 0}, []byte{0x85, 0x02}},
		{"02 a multiple write that reaches a read-only coil", []byte{0x0F, 0, 1, 0, 2, 1, 0x03}, []byte{0x8F, 0x02}},
		{"the refused write changed nothing", []byte{0x01, 0, 0, 0, 3}, []byte{0x01, 0x01, 0x05}},
		{"03 quantity 0", []byte{0x03, 0, 0, 0, 0}, []byte{0x83, 0x03}},
		{"03 quantity over the limit", []byte{0x04, 0, 0, 0, 126}, []byte{0x84, 0x03}},
		{"03 a coil value that is neither FF00 nor 0000", []byte{0x05, 0, 0, 0x12, 0x34}, []byte{0x85, 0x03}},
		{"03 a byte count that disagrees", []byte{0x10, 0, 1, 0, 2, 2, 0x40, 0x40}, []byte{0x90, 0x03}},
		{"the last read the fragile point allows", []byte{0x04, 0, 2, 0, 1}, []byte{0x04, 0x02, 0x00, 0x09}},
		{"04 device failure after its reads", []byte{0x04, 0, 2, 0, 1}, []byte{0x84, 0x04}},
		{"the other points still answer", []byte{0x04, 0, 0, 0, 1}, []byte{0x04, 0x02, 0xFF, 0xFE}},
	})
}

func TestAnUnknownUnitIsSilentOrAGatewayException(t *testing.T) {
	t.Parallel()
	silent := start(t, nil)
	c := silent.dial()
	send(t, c, 1, 9, 0x03, 0, 0, 0, 1)
	send(t, c, 2, 1, 0x04, 0, 0, 0, 1)
	if f := receive(t, c); f.Transaction != 2 {
		t.Errorf("first response is to transaction %d, want 2: unit 9 must get none", f.Transaction)
	}
	if e := silent.logged()[0]; e.Unit != 9 || e.Result != reqlog.ResultNoResponse || e.Reason != reqlog.ReasonUnknownUnit {
		t.Errorf("log of the unanswered request = %+v", e)
	}
	gateway := start(t, func(o *Options) { o.GatewayException = true })
	if got := ask(t, gateway.dial(), 9, 0x03, 0, 0, 0, 1); !bytes.Equal(got, []byte{0x83, 0x0B}) {
		t.Errorf("gateway mode: % X, want 83 0B", got)
	}
}

func TestSeveralSlavesAnswerBehindOneListener(t *testing.T) {
	t.Parallel()
	c := start(t, nil).dial()
	if got := ask(t, c, 1, 0x04, 0, 0, 0, 1); !bytes.Equal(got, []byte{0x04, 0x02, 0xFF, 0xFE}) {
		t.Errorf("unit 1: % X", got)
	}
	if got := ask(t, c, 2, 0x04, 0, 0, 0, 1); !bytes.Equal(got, []byte{0x04, 0x02, 0x00, 0x05}) {
		t.Errorf("unit 2: % X", got)
	}
}

func TestPipelinedAndSplitFramesAreEachAnswered(t *testing.T) {
	t.Parallel()
	c := start(t, nil).dial()
	one := []byte{0, 1, 0, 0, 0, 6, 1, 0x04, 0, 0, 0, 1}
	two := []byte{0, 2, 0, 0, 0, 6, 2, 0x04, 0, 0, 0, 1}
	foreign := []byte{0, 3, 0, 7, 0, 6, 1, 0x04, 0, 0, 0, 1}
	if _, err := c.Write(append(append(append([]byte{}, one...), foreign...), two[:5]...)); err != nil {
		t.Fatal(err)
	}
	if f := receive(t, c); f.Transaction != 1 || f.Unit != 1 {
		t.Errorf("first response %+v", f)
	}
	if _, err := c.Write(two[5:]); err != nil {
		t.Fatal(err)
	}
	if f := receive(t, c); f.Transaction != 2 || !bytes.Equal(f.PDU, []byte{0x04, 0x02, 0x00, 0x05}) {
		t.Errorf("second response %+v: the frame of protocol 7 must be discarded and the split one joined", f)
	}
}

func TestAFrameWithAnImpossibleLengthClosesTheConnection(t *testing.T) {
	t.Parallel()
	c := start(t, nil).dial()
	if _, err := c.Write([]byte{0, 1, 0, 0, 0xFF, 0xFF, 1, 3}); err != nil {
		t.Fatal(err)
	}
	if !closedByServer(c) {
		t.Errorf("read after the bad frame: the server did not close the connection")
	}
}

func TestBehavioursFollowTheInjectedClock(t *testing.T) {
	t.Parallel()
	s := start(t, nil)
	c := s.dial()
	for _, step := range []struct {
		advance time.Duration
		want    byte
	}{{0, 100}, {time.Second, 150}, {2 * time.Second, 50}} {
		s.advance(step.advance)
		if got := ask(t, c, 1, 0x04, 0, 1, 0, 1); !bytes.Equal(got, []byte{0x04, 0x02, 0x00, step.want}) {
			t.Errorf("after +%v: % X, want the sine at %d", step.advance, got, step.want)
		}
	}
}

func TestAResponseIsHeldForTheScenarioDelay(t *testing.T) {
	t.Parallel()
	s := start(t, nil)
	c := s.dial()
	ask(t, c, 1, 0x04, 0, 0, 0, 1)
	ask(t, c, 1, 0x04, 0, 2, 0, 1)
	ask(t, c, 2, 0x04, 0, 0, 0, 1)
	ask(t, c, 2, 0x04, 0, 7, 0, 1)
	want := []time.Duration{300 * time.Millisecond, 150 * time.Millisecond, 150 * time.Millisecond}
	if got := s.slept(); !reflect.DeepEqual(got, want) {
		t.Errorf("delays = %v, want %v (none for the first request, the device's for an exception too)", got, want)
	}
}

func TestADeviceInsideItsOfflineWindowDoesNotAnswer(t *testing.T) {
	t.Parallel()
	s := start(t, nil)
	c := s.dial()
	s.advance(10 * time.Second)
	send(t, c, 1, 2, 0x04, 0, 0, 0, 1)
	send(t, c, 2, 1, 0x04, 0, 0, 0, 1)
	if f := receive(t, c); f.Transaction != 2 {
		t.Errorf("first response is to transaction %d, want 2: unit 2 is offline", f.Transaction)
	}
	if e := s.logged()[0]; e.Device != "unit-b" || e.Result != reqlog.ResultNoResponse || e.Reason != reqlog.ReasonOffline {
		t.Errorf("log of the offline request = %+v", e)
	}
	s.advance(10 * time.Second)
	if got := ask(t, c, 2, 0x04, 0, 0, 0, 1); !bytes.Equal(got, []byte{0x04, 0x02, 0x00, 0x05}) {
		t.Errorf("after the window: % X", got)
	}
}
