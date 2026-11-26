package reportmodule

import (
	"reflect"
	"testing"
)

func TestSplitModuleOnlyKeepsOtherRowsAndListsModuleFiles(t *testing.T) {
	out := "FAIL  eslint  js/app.js:1  " + moduleOnlyParse + "\n" +
		"FAIL  eslint  js/floor.js:2  parse: Parsing error: Unexpected token ?\n" +
		"WARN  eslint  js/x.js:9  no-unused-vars: 'y' is assigned a value but never used.\n" +
		"FAIL  eslint  dir/lib.js:3  " + moduleOnlyParse + "\n"
	kept, files, removed := splitModuleOnly(out)
	want := "FAIL  eslint  js/floor.js:2  parse: Parsing error: Unexpected token ?\n" +
		"WARN  eslint  js/x.js:9  no-unused-vars: 'y' is assigned a value but never used.\n"
	if kept != want {
		t.Fatalf("kept = %q, want %q", kept, want)
	}
	if !reflect.DeepEqual(files, []string{"js/app.js", "dir/lib.js"}) || !removed {
		t.Fatalf("files = %v removed = %v", files, removed)
	}
	if !hasFailRow(kept) {
		t.Fatal("the other parse error must stay a FAIL row")
	}
}

func TestSplitModuleOnlyLeavesOutputWithoutModuleRowsUnchanged(t *testing.T) {
	out := "FAIL  eslint  js/app.js:2  eqeqeq: Expected '===' and instead saw '=='.\n"
	kept, files, removed := splitModuleOnly(out)
	if kept != out || files != nil || removed {
		t.Fatalf("kept=%q files=%v removed=%v", kept, files, removed)
	}
}
