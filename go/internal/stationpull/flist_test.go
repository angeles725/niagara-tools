package stationpull

import (
	"reflect"
	"testing"
)

// flistFixture mixes the NRE boot line, a header, names with spaces, a CRC that was not computed (0),
// a directory row and the harmless trailing exception plat prints after a successful call.
const flistFixture = "INFO [nre] Booting\r\n" +
	"Name                        Size         Date                 CRC\r\n" +
	"config.bog                  1,234,567    2026-11-25 10:00:00  3,456,789,012\r\n" +
	"my station file.bog         10           2026-11-25 10:00:01  42\r\n" +
	"sub/                        0            2026-11-25 10:00:02  0\r\n" +
	"notes v2 final.txt         7            2026-11-25 10:00:03  0\r\n" +
	"javax.baja.sys.BajaRuntimeException: niagarad version ?\r\n" +
	"\tat javax.baja.sys.Sys.main(Sys.java:1)\r\n"

func TestParseFlistReadsNamesWithSpacesAndTheLastFieldCRC(t *testing.T) {
	want := []Row{
		{Name: "config.bog", CRC: 3456789012},
		{Name: "my station file.bog", CRC: 42},
		{Name: "notes v2 final.txt", CRC: 0},
	}
	got := ParseFlist(flistFixture)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rows\n got %#v\nwant %#v", got, want)
	}
}

func TestParseFlistSingleSpacedColumns(t *testing.T) {
	got := ParseFlist("my file.bog 1,024 2026-11-25 10:00:01 AM 77\nplain.txt 5 2026-11-25 10:00:01 0\n")
	want := []Row{{Name: "my file.bog", CRC: 77}, {Name: "plain.txt", CRC: 0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rows\n got %#v\nwant %#v", got, want)
	}
}

func TestParseFlistIgnoresLinesWithoutACRC(t *testing.T) {
	if got := ParseFlist("Platform summary\nDaemon Version:  4.14.0.162\n\nName  Size  CRC\n"); len(got) != 0 {
		t.Fatalf("rows %#v, want none", got)
	}
}
