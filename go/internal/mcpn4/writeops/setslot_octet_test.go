package writeops_test

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// A BACnet octet string (a Spyder's MS/TP MAC, an IP device's address) is stored as lowercase hex
// octets separated by spaces: "0C" is stored as "0c", "C0A80101BAC0" as "c0 a8 01 01 ba c0".
// The same octets are verified, other octets stay a mismatch (retro 2026-11-17 R1, batch 176b9aba).
func TestAnOctetStringReadBackComparesTheOctetsNotTheSpelling(t *testing.T) {
	spyder := bacnetNet + "/$32013"
	nodes := func(mac string) *pyjson.Object {
		return obj("", obj("h", "77", "t", "honeywellBacnetSpyder:BacnetSpyder"),
			"address", obj("t", "bacnet:BacnetAddress"),
			"address/macAddress", obj("t", "bacnet:BacnetOctetString", "v", mac))
	}
	for _, tc := range []struct{ requested, stored, verdict string }{
		{"0C", "0c", "verified"},
		{"0c", "0c", "verified"},
		{"C0A80101BAC0", "c0 a8 01 01 ba c0", "verified"},
		{"c0 A8 01 01 BA c0", "c0 a8 01 01 ba c0", "verified"},
		{"0D", "0c", "mismatch"},
		{"0C0", "0c", "mismatch"},
	} {
		c := &loads{nodes: obj(spyder, nodes("00"))}
		args := obj("ord", spyder, "slot", "address/macAddress", "value", tc.requested, "value_type", "baja:Simple")
		p, err := writeops.SetSlot.Plan(c, args)
		if err != nil {
			t.Fatalf("%s: plan: %v", tc.requested, err)
		}
		c.nodes = obj(spyder, nodes(tc.stored))
		_, _, _, verdict, err := writeops.SetSlot.Readback(c, args, p, []any{nil}, p.Inverse)
		if err != nil || verdict != tc.verdict {
			t.Errorf("%q stored as %q: verdict = %q, %v; want %s", tc.requested, tc.stored, verdict, err, tc.verdict)
		}
	}
}
