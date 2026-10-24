package citeresolve

import "testing"

// TestKinds pins the I4 kinds and roots of kit issue #418.
func TestKinds(t *testing.T) {
	t.Parallel()
	check(t, []struct {
		name, text string
		want       []want
	}{
		{"course pdf page in its text twin", "[ev: course Labs/Lab Instructions.pdf p2]", []want{{Resolved, "p2: Lab two title"}}},
		{"course pdf page range", "[ev: course Labs/Lab Instructions.pdf p1-3]", []want{{Resolved, "p1: Lab one title"}}},
		{"course pdf page past the end", "[ev: course Labs/Lab Instructions.pdf p4]", []want{{LineOutOfRange, "has 3"}}},
		{"course directory", "[ev: course Labs/Lab10 Jobs]", []want{{Resolved, "directory"}}},
		{"course file lines", "[ev: course Slide Code/gradle.properties:14-25]", []want{{Resolved, "course.prop14=x"}}},
		{"course text by stem, prose after it", "[ev: course Review Assignment 2 build]", []want{{Resolved, "unchecked: build"}}},
		{"course text lines", "[ev: course Lab Instructions.txt:2]", []want{{Resolved, "body"}}},
		{"course file missing", "[ev: course Labs/Nope.pdf p1]", []want{{MissingFile, "course tree"}}},
		{"bitácora section, worktree copies one file", "[ev: bitácora shop-5cuartos §11]", []want{{Resolved, "## 11. Enfriar"}}},
		{"bitácora by short slug", "[ev: bitácora 5cuartos §6]", []want{{Resolved, "## 6. Deploy"}}},
		{"bitácora missing section", "[ev: bitácora 5cuartos §4C]", []want{{MissingSection, "§4C"}}},
		{"bitácora in two clients is ambiguous", "[ev: bitácora twice]", []want{{Ambiguous, "2 files match"}}},
		{"bitácora missing", "[ev: bitácora nope]", []want{{MissingFile, "bitacora/<date>-nope.md"}}},
		{"kit path", "[ev: kit types/platform.md § 2]", []want{{Resolved, "## 2. Software Manager"}}},
		{"research retro", "[ev: retro module-worktree-location-retro Δ1]", []want{{Resolved, "research delta one"}}},
		{"external retro", "[ev: external retro niagara-research/retros/2026-09-11-module-worktree-location-retro.md Δ1]", []want{{Resolved, "[research retros/"}}},
		{"module.xml of a part", "[ev: gamma-rt module.xml]", []want{{Resolved, "gamma-rt module line 1"}}},
		{"client module code", "[ev: Shop BShop.java:3]", []want{{Resolved, "shop line 3"}}},
		{"memory named before the word", "[ev: shared-checkout memory]", []want{{Resolved, "shared-checkout-git-discipline.md"}}},
		{"vineflower with no module", "[ev: vineflower kitControl/BFoo.java :7]", []want{{Resolved, "// BFoo line 7"}}},
	})
}
