package reqlog

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

func TestWriterPrintsOneJSONLinePerRequest(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := NewWriter(&buf)
	w.Log(Entry{Time: "2026-01-02T03:04:05Z", ElapsedMS: 1500, Remote: "127.0.0.1:40000", Unit: 1, Device: "unit-a",
		Function: 16, Address: 1, Quantity: 2, Written: []int{16448, 0}, Result: ResultOK})
	w.Log(Entry{Time: "2026-01-02T03:04:06Z", Unit: 9, Function: 3, Address: 0, Quantity: 1, Result: ResultNoResponse, Reason: ReasonUnknownUnit})
	want := `{"time":"2026-01-02T03:04:05Z","elapsed_ms":1500,"remote":"127.0.0.1:40000","unit":1,"device":"unit-a","function":16,"address":1,"quantity":2,"written":[16448,0],"result":"ok"}
{"time":"2026-01-02T03:04:06Z","elapsed_ms":0,"remote":"","unit":9,"function":3,"address":0,"quantity":1,"result":"no_response","reason":"unknown_unit"}
`
	if got := buf.String(); got != want {
		t.Errorf("log =\n%s\nwant\n%s", got, want)
	}
}

func TestWriterKeepsConcurrentLinesWhole(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := NewWriter(&buf)
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() { w.Log(Entry{Unit: i, Result: ResultOK, Read: []int{1, 2, 3}}) })
	}
	wg.Wait()
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 50 {
		t.Fatalf("%d lines, want 50", len(lines))
	}
	for _, line := range lines {
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("line %q: %v", line, err)
		}
	}
}

func TestANilWriterDropsEntries(t *testing.T) {
	t.Parallel()
	var w *Writer
	w.Log(Entry{Unit: 1})
}

func TestABACnetEntryIsItsOwnLineAndLeavesModbusLinesAlone(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := NewWriter(&buf)
	invoke, priority := 7, 8
	w.Log(Entry{Time: "2026-01-02T03:04:05Z", Unit: 1, Function: 3, Address: 0, Quantity: 1, Result: ResultOK, Read: []int{5}})
	w.LogBACnet(BACnet{Time: "2026-01-02T03:04:06Z", ElapsedMS: 20, Protocol: ProtocolBACnetIP, Remote: "127.0.0.1:47808", Device: "unit-a",
		Link: "original-unicast-npdu", PDU: "confirmed-request", Service: "write-property", InvokeID: &invoke,
		Object: "analog-output,1", Property: "present-value", Priority: &priority, Written: "55", Result: ResultOK})
	w.LogBACnet(BACnet{Time: "2026-01-02T03:04:07Z", Protocol: ProtocolBACnetIP, Device: "unit-a", Link: "original-broadcast-npdu",
		PDU: "unconfirmed-request", Service: "who-is", Result: ResultOK, To: []string{"127.0.0.1:47808"}})
	want := `{"time":"2026-01-02T03:04:05Z","elapsed_ms":0,"remote":"","unit":1,"function":3,"address":0,"quantity":1,"result":"ok","read":[5]}
{"time":"2026-01-02T03:04:06Z","elapsed_ms":20,"protocol":"bacnet-ip","remote":"127.0.0.1:47808","device":"unit-a","link":"original-unicast-npdu","pdu":"confirmed-request","service":"write-property","invoke_id":7,"object":"analog-output,1","property":"present-value","priority":8,"written":"55","result":"ok"}
{"time":"2026-01-02T03:04:07Z","elapsed_ms":0,"protocol":"bacnet-ip","remote":"","device":"unit-a","link":"original-broadcast-npdu","pdu":"unconfirmed-request","service":"who-is","result":"ok","to":["127.0.0.1:47808"]}
`
	if got := buf.String(); got != want {
		t.Errorf("log =\n%s\nwant\n%s", got, want)
	}
	var none *Writer
	none.LogBACnet(BACnet{})
}
