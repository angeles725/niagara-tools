import json
import os
import stat
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from mcp_n4 import safety  # noqa: E402


class Clock:
    def __init__(self, now=1000.0):
        self.now = now

    def __call__(self):
        return self.now


class TestConfirmationTokens(unittest.TestCase):
    def setUp(self):
        self.clock = Clock()
        self.tokens = safety.ConfirmationTokens(ttl=300, clock=self.clock)
        self.args = {"ord": "station:|slot:/A", "slot": "out"}

    def issue(self, tool="n4_set_slot", args=None, plan_hash="p1"):
        return self.tokens.issue(tool, self.args if args is None else args, plan_hash)

    def consume(self, token, tool="n4_set_slot", args=None, plan_hash="p1"):
        return self.tokens.consume(tool, self.args if args is None else args, plan_hash, token)

    def test_issue_returns_token_and_expiry_from_ttl(self):
        token, expires_at = self.issue()
        self.assertIsInstance(token, str)
        self.assertEqual(expires_at, 1300)

    def test_valid_token_is_accepted_once(self):
        token, _ = self.issue()
        self.consume(token)
        with self.assertRaisesRegex(safety.SafetyError, "already used"):
            self.consume(token)

    def test_missing_or_malformed_token_is_refused(self):
        for bad in (None, "", "garbage", "1.2", 5):
            with self.assertRaises(safety.SafetyError, msg=repr(bad)):
                self.consume(bad)

    def test_non_ascii_or_oversized_token_parts_raise_only_safety_error(self):
        token, _ = self.issue()
        expiry, nonce, mac = token.split(".")
        for bad in ("\u00b2.%s.%s" % (nonce, mac), "%s.%s.\u00e9abc" % (expiry, nonce),
                    "%s.\u00e9.%s" % (expiry, mac), "\u0663\u0661.a.b", "9" * 400 + ".a.b",
                    "-5.a.b", "%s.%s.%s" % (expiry, nonce, mac) + "\u2166"):
            with self.assertRaises(safety.SafetyError, msg=repr(bad)):
                self.consume(bad)
        self.consume(token)  # the genuine token is still spendable

    def test_expired_token_is_refused(self):
        token, _ = self.issue()
        self.clock.now = 1300
        with self.assertRaisesRegex(safety.SafetyError, "expired"):
            self.consume(token)

    def test_token_still_valid_just_before_expiry(self):
        token, _ = self.issue()
        self.clock.now = 1299
        self.consume(token)

    def test_tampered_token_is_refused(self):
        token, _ = self.issue()
        flipped = token[:-1] + ("0" if token[-1] != "0" else "1")
        with self.assertRaises(safety.SafetyError):
            self.consume(flipped)

    def test_tampered_expiry_is_refused(self):
        token, _ = self.issue()
        expiry, rest = token.split(".", 1)
        with self.assertRaises(safety.SafetyError):
            self.consume("%d.%s" % (int(expiry) + 9999, rest))

    def test_token_is_bound_to_tool_args_and_plan_hash(self):
        for kw in ({"tool": "n4_create_link"}, {"args": {"ord": "station:|slot:/B"}},
                   {"plan_hash": "p2"}):
            token, _ = self.issue()
            with self.assertRaises(safety.SafetyError, msg=kw):
                self.consume(token, **kw)

    def test_dry_run_and_token_args_are_excluded_from_the_binding(self):
        token, _ = self.issue()
        self.consume(token, args=dict(self.args, dry_run=False, confirmation_token=token))

    def test_arg_order_does_not_matter(self):
        token, _ = self.issue(args={"a": 1, "b": 2})
        self.consume(token, args={"b": 2, "a": 1})

    def test_token_from_another_process_key_is_refused(self):
        other = safety.ConfirmationTokens(ttl=300, clock=self.clock)
        token, _ = other.issue("n4_set_slot", self.args, "p1")
        with self.assertRaises(safety.SafetyError):
            self.consume(token)

    def test_expired_entries_are_pruned_on_issue(self):
        self.issue()
        self.clock.now = 5000
        self.issue()
        self.assertEqual(len(self.tokens._pending), 1)


class TestWriteScope(unittest.TestCase):
    def test_no_prefix_refuses_everything(self):
        with self.assertRaisesRegex(safety.SafetyError, "no --write-scope"):
            safety.WriteScope([]).check("station:|slot:/A")

    def test_prefix_covers_itself_and_descendants(self):
        scope = safety.WriteScope(["station:|slot:/A"])
        for ord_ in ("station:|slot:/A", "station:|slot:/A/", "station:|slot:/A/B",
                     "station:|slot:/A/B/C"):
            scope.check(ord_)

    def test_prefix_match_respects_slot_boundaries(self):
        scope = safety.WriteScope(["station:|slot:/A"])
        for ord_ in ("station:|slot:/AB", "station:|slot:/", "station:|slot:/B/A"):
            with self.assertRaisesRegex(safety.SafetyError, "outside the write scope"):
                scope.check(ord_)

    def test_trailing_slash_in_prefix_is_equivalent(self):
        scope = safety.WriteScope(["station:|slot:/A/"])
        scope.check("station:|slot:/A/B")
        with self.assertRaises(safety.SafetyError):
            scope.check("station:|slot:/AB")

    def test_root_prefix_covers_the_whole_station(self):
        scope = safety.WriteScope(["station:|slot:/"])
        scope.check("station:|slot:/")
        scope.check("station:|slot:/Anything/Deep")

    def test_any_of_several_prefixes_matches(self):
        scope = safety.WriteScope(["station:|slot:/A", "station:|slot:/B"])
        scope.check("station:|slot:/B/x")
        with self.assertRaises(safety.SafetyError):
            scope.check("station:|slot:/C")

    def test_parent_traversal_and_foreign_ords_are_refused(self):
        scope = safety.WriteScope(["station:|slot:/A"])
        for ord_ in ("station:|slot:/A/../B", "/A/B", "local:|foo", ""):
            with self.assertRaises(safety.SafetyError, msg=ord_):
                scope.check(ord_)

    def test_non_string_ord_is_refused(self):
        with self.assertRaises(safety.SafetyError):
            safety.WriteScope(["station:|slot:/"]).check(None)


class StateFileCase(unittest.TestCase):
    def setUp(self):
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.state_dir = os.path.join(tmp.name, "state", "mcp-n4")

    @staticmethod
    def mode(path):
        return stat.S_IMODE(os.stat(path).st_mode)

    def lines(self, name):
        with open(os.path.join(self.state_dir, name)) as fh:
            return [json.loads(line) for line in fh]


class TestJournal(StateFileCase):
    def test_append_writes_one_json_line_per_entry(self):
        journal = safety.Journal(self.state_dir)
        journal.append({"batch_id": "b1", "inverse": [{"nm": "v", "h": "3", "n": "x"}]})
        journal.append({"batch_id": "b2", "inverse": []})
        entries = self.lines("journal.jsonl")
        self.assertEqual([e["batch_id"] for e in entries], ["b1", "b2"])
        self.assertEqual(entries[0]["inverse"], [{"nm": "v", "h": "3", "n": "x"}])

    def test_dir_is_0700_and_file_is_0600(self):
        safety.Journal(self.state_dir).append({"batch_id": "b1"})
        self.assertEqual(self.mode(self.state_dir), 0o700)
        self.assertEqual(self.mode(os.path.join(self.state_dir, "journal.jsonl")), 0o600)

    def test_existing_dir_is_never_chmodded_by_append(self):
        os.makedirs(self.state_dir, mode=0o750)
        os.chmod(self.state_dir, 0o750)
        safety.Journal(self.state_dir).append({"batch_id": "b1"})
        self.assertEqual(self.mode(self.state_dir), 0o750)
        self.assertEqual(self.mode(os.path.join(self.state_dir, "journal.jsonl")), 0o600)

    def test_created_dir_is_0700_whatever_the_umask(self):
        old = os.umask(0)
        self.addCleanup(os.umask, old)
        safety.Journal(self.state_dir).append({"batch_id": "b1"})
        self.assertEqual(self.mode(self.state_dir), 0o700)

    def test_read_merges_intent_and_result_and_flags_in_doubt(self):
        journal = safety.Journal(self.state_dir)
        self.assertIsNone(journal.read("b1"))  # no file yet
        journal.append({"batch_id": "b1", "phase": "intent", "tool": "t", "ops": [1],
                        "inverse_plan": [2], "station_name": "S"})
        view = journal.read("b1")
        self.assertEqual((view["state"], view["inverse_plan"], view["tool"]),
                         ("in-doubt", [2], "t"))
        self.assertIsNone(view["inverse"])
        journal.append({"batch_id": "b1", "phase": "result", "accepted": "ok",
                        "inverse": [3], "verdict": "verified"})
        view = journal.read("b1")
        self.assertEqual((view["state"], view["inverse"], view["verdict"], view["accepted"]),
                         ("completed", [3], "verified", "ok"))
        self.assertIsNone(journal.read("nope"))

    def test_read_ignores_torn_lines_and_flags_in_doubt_results(self):
        journal = safety.Journal(self.state_dir)
        journal.append({"batch_id": "b1", "phase": "intent", "tool": "t"})
        with open(journal.path, "a") as fh:
            fh.write("{torn\n")
        journal.append({"batch_id": "b1", "phase": "result", "inverse": [],
                        "verdict": "unverified", "in_doubt": True})
        self.assertEqual(journal.read("b1")["state"], "in-doubt")

    def test_rollbacks_of_lists_batches_that_roll_back_another(self):
        journal = safety.Journal(self.state_dir)
        journal.append({"batch_id": "r1", "phase": "intent", "rollback_of": "b1"})
        journal.append({"batch_id": "r1", "phase": "result", "verdict": "failed", "inverse": []})
        journal.append({"batch_id": "r2", "phase": "intent", "rollback_of": "b1"})
        self.assertEqual([(v["batch_id"], v["state"]) for v in journal.rollbacks_of("b1")],
                         [("r1", "completed"), ("r2", "in-doubt")])


class TestJournalMergeRules(StateFileCase):
    def view(self, *entries):
        journal = safety.Journal(self.state_dir)
        journal.append({"batch_id": "b1", "phase": "intent", "tool": "t"})
        for entry in entries:
            journal.append(dict(entry, batch_id="b1"))
        return journal.read("b1")

    def test_relink_and_component_ops_are_present_exactly_when_a_record_exists(self):
        bare = self.view()
        self.assertNotIn("relink_ops", bare)
        self.assertNotIn("component_ops", bare)

    def test_an_empty_record_is_present_for_both_kinds(self):
        view = self.view({"phase": "relink-intent"}, {"phase": "component-intent", "ops": []})
        self.assertEqual(view["relink_ops"], [])
        self.assertEqual(view["component_ops"], [])

    def test_recorded_ops_are_kept_for_both_kinds(self):
        view = self.view({"phase": "relink-intent", "ops": [1]},
                         {"phase": "component-intent", "ops": [2]})
        self.assertEqual((view["relink_ops"], view["component_ops"]), ([1], [2]))


class TestExistingStateFiles(StateFileCase):
    def seed(self, mode):
        os.makedirs(self.state_dir, mode=0o700)
        path = os.path.join(self.state_dir, "journal.jsonl")
        with open(path, "w"):
            pass
        os.chmod(path, mode)
        return path

    def test_append_refuses_a_loose_existing_file_and_leaves_it_alone(self):
        path = self.seed(0o644)
        with self.assertRaises(PermissionError) as cm:
            safety.Journal(self.state_dir).append({"batch_id": "b1"})
        self.assertIn("chmod 600", str(cm.exception))
        self.assertEqual(self.mode(path), 0o644)
        self.assertEqual(os.path.getsize(path), 0)

    def test_append_still_accepts_a_private_existing_file(self):
        path = self.seed(0o600)
        safety.Journal(self.state_dir).append({"batch_id": "b1"})
        self.assertGreater(os.path.getsize(path), 0)

    def test_check_state_dir_names_a_loose_state_file_at_startup(self):
        self.seed(0o664)
        with self.assertRaises(safety.SafetyError) as cm:
            safety.check_state_dir(self.state_dir)
        self.assertIn("journal.jsonl", str(cm.exception))

    def test_check_state_dir_accepts_private_state_files(self):
        self.seed(0o600)
        safety.check_state_dir(self.state_dir)


class TestStateDirCheck(StateFileCase):
    def test_missing_dir_is_fine(self):
        safety.check_state_dir(self.state_dir)

    def test_owned_private_dir_is_fine(self):
        os.makedirs(self.state_dir, mode=0o700)
        safety.check_state_dir(self.state_dir)

    def test_group_or_world_writable_dir_is_refused_and_not_chmodded(self):
        os.makedirs(self.state_dir)
        for mode in (0o775, 0o777, 0o722):
            os.chmod(self.state_dir, mode)
            with self.assertRaises(safety.SafetyError) as cm:
                safety.check_state_dir(self.state_dir)
            self.assertIn("--state-dir", str(cm.exception))
            self.assertEqual(self.mode(self.state_dir), mode)

    def test_dir_owned_by_another_user_is_refused(self):
        os.makedirs(self.state_dir, mode=0o700)
        real = os.getuid
        safety.os.getuid = lambda: real() + 1
        self.addCleanup(setattr, safety.os, "getuid", real)
        with self.assertRaises(safety.SafetyError):
            safety.check_state_dir(self.state_dir)

    def test_a_file_in_place_of_the_dir_is_refused(self):
        os.makedirs(os.path.dirname(self.state_dir))
        with open(self.state_dir, "w"):
            pass
        with self.assertRaises(safety.SafetyError):
            safety.check_state_dir(self.state_dir)


    def test_construction_has_no_filesystem_side_effect(self):
        safety.Journal(self.state_dir)
        self.assertFalse(os.path.exists(self.state_dir))


class TestAuditLog(StateFileCase):
    def test_append_and_modes(self):
        audit = safety.AuditLog(self.state_dir)
        audit.append({"tool": "n4_set_slot", "outcome": "planned"})
        self.assertEqual(self.lines("audit.jsonl")[0]["tool"], "n4_set_slot")
        self.assertEqual(self.mode(self.state_dir), 0o700)
        self.assertEqual(self.mode(os.path.join(self.state_dir, "audit.jsonl")), 0o600)

    def test_redaction_masks_secret_looking_keys_at_any_depth(self):
        args = {"ord": "station:|slot:/A", "Password": "p1", "api_secret": "s1",
                "confirmation_token": "t1", "my_Credential": "c1",
                "nested": {"passphrase": "p2", "ok": 1}, "items": [{"token": "t2", "n": 2}]}
        red = safety.redact_args(args)
        self.assertEqual(red["ord"], "station:|slot:/A")
        self.assertEqual(red["nested"], {"passphrase": safety.REDACTED, "ok": 1})
        self.assertEqual(red["items"], [{"token": safety.REDACTED, "n": 2}])
        text = json.dumps(red)
        for secret in ("p1", "s1", "t1", "c1", "p2", "t2"):
            self.assertNotIn('"%s"' % secret, text)

    def test_redaction_does_not_mutate_the_input(self):
        args = {"password": "p1"}
        safety.redact_args(args)
        self.assertEqual(args, {"password": "p1"})


if __name__ == "__main__":
    unittest.main()
