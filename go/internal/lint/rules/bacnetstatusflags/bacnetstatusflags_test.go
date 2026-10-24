package bacnetstatusflags

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the G36 contract (ledger.d/2026-10-23-r8-p1-drivers.md), one hand-written case per testdata/*.txt: a
// BACnet Present_Value proxy (propertyId absent, 85 or 85@...) whose point facets lack statusFlags=true WARNs, one
// row per device with its count; statusFlags=b:true, a non-Present_Value propertyId or a useStatusFlags slot are
// clean; the module alias of a type is resolved (x=bacnet counts, a BacnetNumericProxyExt of another module does
// not); a vendor device type above the point device ext; a proxy outside a device; --strict; a malformed bog, no bog and usage exit 3. TestZipped: a zipped bog.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 8) }

// TestZipped: a saved station config.bog (file.xml zipped) is read through its member like a plain one.
func TestZipped(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("file.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte(`<bajaObjectGraph version='4.0'><p m='b=baja' t='b:Station'><p n='D' m='bac=bacnet' t='bac:BacnetDevice'>` +
		`<p n='P' m='c=control' t='c:NumericPoint'><p n='proxyExt' t='bac:BacnetNumericProxyExt'/></p></p></p></bajaObjectGraph>`))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.bog"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e})
	want := "WARN  lint-bacnet-status-flags  " + dir + "/config.bog: /D: 1 Present_Value proxy point(s) without facets statusFlags=true (first: /D/P) -- a field alarm, fault or override reads {ok}\n"
	if code != 0 || o.String() != want || e.Len() != 0 {
		t.Errorf("exit %d out %q err %q, want 0 and %q", code, o.String(), e.String(), want)
	}
}
