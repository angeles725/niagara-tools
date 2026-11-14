"""The closing retro step is documented (the write mode itself is tested in Go:
go/cmd/mcp-n4/retro-draft/write_test.go, ported from this file's former TestNewRetro)."""
import os
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
KIT = os.path.dirname(HERE)


class TestRetroStepIsDocumented(unittest.TestCase):
    """The closing retro step is enforced here: the docs must name the tool and the CLI."""

    def doc(self, *parts):
        with open(os.path.join(KIT, *parts), encoding="utf-8") as fh:
            return fh.read()

    def test_methodology_and_skill_name_the_closing_step(self):
        for text in (self.doc("METHODOLOGY.md"), self.doc("skill", "SKILL.md")):
            self.assertIn("n4_session_retro_draft", text)
            self.assertIn("retro-draft --write", text)
            self.assertIn("never apply", text)

    def test_methodology_no_longer_marks_the_retro_step_manual_until_tooling(self):
        self.assertNotIn("tooling arrives in T7", self.doc("METHODOLOGY.md"))

    def test_the_template_and_index_exist_with_the_agreed_grammar(self):
        self.assertIn("| file | Station | Date | pending\\|folded | deltas |",
                      self.doc("retros", "INDEX.md"))
        self.assertIn("## Proposed kit deltas", self.doc(
            "..", "go", "internal", "mcpn4", "retro", "retro.template.md"))


if __name__ == "__main__":
    unittest.main()
