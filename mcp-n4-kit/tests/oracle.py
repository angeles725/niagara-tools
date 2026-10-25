"""Frozen Python-server oracles: what the in-process Python server answered, recorded (S17).

A few black-box tests compared the server under test, byte for byte, with the in-process
Python server on the same input. S17 removes that Python, so those answers are frozen into
`fixtures/oracles/<name>.json`: per test (`Class.test`), the calls it made in order, each as
`[args, answer]`. The Go binary is compared with the recording; the args must match it, so a
test that changes its input fails instead of comparing against a stale answer.

Paths that change between runs (a temporary state dir) and today's date were recorded as
placeholders (`aliases`) and are replayed as this run's values. The recordings were made with
`MCP_N4_RECORD_ORACLE=1` while the Python server existed (S17, before its removal); they are
frozen and cannot be re-recorded.
"""
import datetime
import json
import os

FIXTURES = os.path.join(os.path.dirname(os.path.abspath(__file__)), "fixtures", "oracles")
TODAY = "<today-utc>"


def today_utc():
    return datetime.datetime.now(datetime.timezone.utc).date().isoformat()


class Frozen:
    """The recorded oracle of one test: `frozen(args)` is the answer of its next call."""

    def __init__(self, test, name, aliases=()):
        self.test, self.path = test, os.path.join(FIXTURES, name + ".json")
        self.key = ".".join(test.id().split(".")[-2:])
        self.aliases = list(aliases)
        self.next = 0
        with open(self.path, encoding="utf-8") as fh:
            self.recorded = json.load(fh)["calls"].get(self.key)
        if self.recorded is None:
            test.fail("%s has no recording for %s" % (self.path, self.key))
        test.addCleanup(self._all_used)

    def _swap(self, text, back):
        for real, placeholder in self.aliases:
            text = text.replace(placeholder, real) if back else text.replace(real, placeholder)
        return text

    def __call__(self, args):
        """The recorded answer to `args` (which must be the recorded args of this call)."""
        args = json.loads(self._swap(json.dumps(args), False))
        i, self.next = self.next, self.next + 1
        if i >= len(self.recorded):
            self.test.fail("%s: %s makes more calls than the %d recorded"
                           % (self.path, self.key, len(self.recorded)))
        want_args, answer = self.recorded[i]
        self.test.assertEqual(args, want_args, "%s: call %d of %s sends other args"
                              % (self.path, i + 1, self.key))
        return json.loads(self._swap(json.dumps(answer), True))

    def _all_used(self):
        if self.next != len(self.recorded):
            self.test.fail("%s: %s made %d of the %d recorded calls"
                           % (self.path, self.key, self.next, len(self.recorded)))
