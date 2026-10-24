package newretro

import "testing"

func TestFlip(t *testing.T) {
	t.Parallel()
	const open, end = "<!-- build-state.v1 -->\n", "<!-- /build-state.v1 -->\n"
	for _, tc := range []struct{ name, module, in, want string }{
		{"module line, comment stripped", "kit", open + "module:  kit # c\nretro_pending: false\n" + end,
			open + "module:  kit # c\nretro_pending: true\n" + end},
		{"only the first false of the section", "kit", open + "module: kit\nretro_pending:false\nretro_pending: false\n",
			open + "module: kit\nretro_pending:true\nretro_pending: false\n"},
		{"other module untouched", "kit", open + "module: ModuleA\nretro_pending: false\n" + end,
			open + "module: ModuleA\nretro_pending: false\n" + end},
		{"heading before any module line", "kit", open + "## kit \nmodule: other\nretro_pending: falsey",
			open + "## kit \nmodule: other\nretro_pending: truey\n"},
		{"heading after a module line ignored", "kit", open + "module: other\n## kit\nretro_pending: false\n",
			open + "module: other\n## kit\nretro_pending: false\n"},
		{"outside a section and a CRLF marker", "kit", "## kit\nretro_pending: false\n<!-- build-state.v1 -->\r\nmodule: kit\nretro_pending: false\n",
			"## kit\nretro_pending: false\n<!-- build-state.v1 -->\r\nmodule: kit\nretro_pending: false\n"},
		{"empty", "kit", "", ""},
	} {
		if got := Flip(tc.in, tc.module); got != tc.want {
			t.Errorf("%s: Flip = %q, want %q", tc.name, got, tc.want)
		}
	}
}
