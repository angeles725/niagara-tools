package newretro

import "testing"

func TestFlip(t *testing.T) {
	t.Parallel()
	const open, end = "<!-- build-state.v1 -->\n", "<!-- /build-state.v1 -->\n"
	for _, tc := range []struct {
		name, module, in, want string
		out                    Outcome
	}{
		{"module line, comment stripped", "kit", open + "module:  kit # c\nretro_pending: false\n" + end,
			open + "module:  kit # c\nretro_pending: true\n" + end, Flipped},
		{"only the first false of the section", "kit", open + "module: kit\nretro_pending:false\nretro_pending: false\n",
			open + "module: kit\nretro_pending:true\nretro_pending: false\n", Flipped},
		{"other module untouched", "kit", open + "module: ModuleA\nretro_pending: false\n" + end,
			open + "module: ModuleA\nretro_pending: false\n" + end, NoSection},
		{"heading before any module line", "kit", open + "## kit \nmodule: other\nretro_pending: falsey",
			open + "## kit \nmodule: other\nretro_pending: truey\n", Flipped},
		{"heading after a module line ignored", "kit", open + "module: other\n## kit\nretro_pending: false\n",
			open + "module: other\n## kit\nretro_pending: false\n", NoSection},
		{"outside a section and a CRLF marker", "kit", "## kit\nretro_pending: false\n<!-- build-state.v1 -->\r\nmodule: kit\nretro_pending: false\n",
			"## kit\nretro_pending: false\n<!-- build-state.v1 -->\r\nmodule: kit\nretro_pending: false\n", NoSection},
		{"already true", "kit", open + "module: kit\nretro_pending: true # set\n" + end,
			open + "module: kit\nretro_pending: true # set\n" + end, AlreadyTrue},
		{"no retro_pending line", "kit", open + "module: kit\nretro_required: true\n" + end,
			open + "module: kit\nretro_required: true\n" + end, NoPending},
		{"flipped beats already true elsewhere", "kit", open + "## kit\nretro_pending: true\n" + end + open + "## kit\nretro_pending: false\n",
			open + "## kit\nretro_pending: true\n" + end + open + "## kit\nretro_pending: true\n", Flipped},
		{"module matched literally", `a\b`, open + "module: a\\b\nretro_pending: false\n", open + "module: a\\b\nretro_pending: true\n", Flipped},
		{"empty", "kit", "", "", NoSection},
	} {
		if got, out := Flip(tc.in, tc.module); got != tc.want || out != tc.out {
			t.Errorf("%s: Flip = %q, %d; want %q, %d", tc.name, got, out, tc.want, tc.out)
		}
	}
}
