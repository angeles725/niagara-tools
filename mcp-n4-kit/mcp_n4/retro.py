"""Session retro drafts: evidence-backed CANDIDATE kit deltas from audit + journal.

Pure logic over the two append-only JSON-lines files the write tools produce. A
candidate is a mechanical observation (a refusal class, a bad read-back verdict, an
in-doubt batch, ...) with the batch_ids / audit timestamps that prove it; deciding
whether it is a real kit gap is the human reviewer's job. Candidates are built only
from timestamps, tool names, batch_ids, op names and fixed text: free-form audit
reasons and `args_redacted` are never copied, so nothing secret can leak.
Importing this module has no side effects.
"""
import datetime
import os
import re

from . import safety

#: Package data: it travels with `mcp_n4`, so it resolves whether the kit is run from the
#: repository or installed.
TEMPLATE = os.path.join(os.path.dirname(os.path.abspath(__file__)), "templates",
                        "retro.template.md")
HONESTY = "no new deltas; the kit already covers this session."
DELTA_HEADER = ("| # | Proposed change | Target (file · section) | "
                "Evidence (batch_id / tool call / audit line) | Type | Priority |")
_MAX_EVIDENCE = 5

#: (key, substrings of the refusal reason, proposed change, target, priority). The
#: substrings are the shared `safety.REASON_*` constants, so a reworded refusal cannot
#: silently stop being classified.
_REFUSALS = [
    ("token_missing", (safety.REASON_TOKEN_MISSING,),
     "Make the dry-run -> token step impossible to skip in the skill/checklist wording",
     "METHODOLOGY.md · section 4", "MEDIUM"),
    ("token_mismatch", (safety.REASON_TOKEN_MISMATCH,),
     "Warn that any argument change after the dry run invalidates the token",
     "METHODOLOGY.md · section 2 (L1)", "MEDIUM"),
    ("token_expired", (safety.REASON_TOKEN_EXPIRED,),
     "Document the token lifetime and when to raise --token-ttl for slow human approval",
     "README.md · running the server", "LOW"),
    ("token_reused", (safety.REASON_TOKEN_REUSED,),
     "State that a token is single use and a retry needs a fresh dry run",
     "METHODOLOGY.md · section 4 step 5", "LOW"),
    ("identity", (safety.REASON_IDENTITY,),
     "Put expected_station in the first checklist step so the connect is not repeated",
     "METHODOLOGY.md · section 4 step 1", "MEDIUM"),
    ("scope", (safety.REASON_SCOPE_NONE, safety.REASON_SCOPE_PLAIN,
               safety.REASON_SCOPE_OUTSIDE),
     "Explain how to choose --write-scope prefixes before the session starts",
     "README.md · write scope", "MEDIUM"),
    ("budget", (safety.REASON_BUDGET, safety.REASON_BUDGET_SMALL),
     "Document sizing --max-writes for rollbacks and multi-op batches",
     "README.md · write budget", "LOW"),
    ("not_connected", (safety.REASON_NOT_CONNECTED,),
     "Remind the agent to call n4_connect before any station tool",
     "skill/SKILL.md · session checklist", "LOW"),
    ("partial_status", (safety.REASON_PARTIAL_STATUS,),
     "Teach that Status slots take value_type + status together in the write tool docs",
     "METHODOLOGY.md · section 3", "MEDIUM"),
    ("unsupported", (safety.REASON_ACTION,),
     "Decide whether the refused action is worth supporting or documenting as unsupported",
     "mcp_n4/tools_write.py · allowed actions", "MEDIUM"),
    ("unsupported", (safety.REASON_VALUE_TYPE,),
     "Decide whether the refused value type is worth supporting or documenting",
     "mcp_n4/tools_write.py · VALUE_TYPES", "MEDIUM"),
    ("tier", (safety.REASON_TIER,),
     "Record the PoC/probe outcome for this build and decide on --allow-tier-b/--allow-tier-c",
     "METHODOLOGY.md · section 5", "MEDIUM"),
    ("unsupported", (safety.REASON_TYPE_SPEC,),
     "Document the module:Type spec expected for component creation",
     "README.md · write tools", "LOW"),
]
_VERDICTS = {
    "mismatch": ("Investigate why the read-back differs from the request and document the cause",
                 "METHODOLOGY.md · section 3", "HIGH"),
    "failed": ("Investigate why the read-back failed after the station accepted the write",
               "mcp_n4/tools_write.py · read-back", "HIGH"),
    "partial": ("Restore by hand the configuration a rollback reported as not restored "
                "(frozen_config_not_restored, link_inputs_not_restored)",
                "METHODOLOGY.md · section 3", "MEDIUM"),
    "unverified": ("Add a way to verify this write class, or document that it stays unverified",
                   "METHODOLOGY.md · section 3", "MEDIUM"),
}


def now():
    """Current UTC time as the ISO text the audit log uses."""
    return datetime.datetime.now(datetime.timezone.utc).isoformat(timespec="seconds")


def _parse_ts(text):
    """Aware UTC datetime from an ISO-8601 string; ValueError when it is not one."""
    if not isinstance(text, str):
        raise ValueError("timestamp must be an ISO-8601 string, got %r" % (text,))
    try:
        ts = datetime.datetime.fromisoformat(text.strip().replace("Z", "+00:00"))
    except ValueError:
        raise ValueError("not an ISO-8601 timestamp: %r" % (text,)) from None
    return ts if ts.tzinfo else ts.replace(tzinfo=datetime.timezone.utc)


def _since_filter(since):
    if since is None:
        return lambda ts: True
    floor = _parse_ts(since)

    def keep(ts):
        try:
            return _parse_ts(ts) >= floor
        except ValueError:  # an entry with no usable timestamp is kept, never hidden
            return True
    return keep


def load(state_dir):
    """`(audit entries, merged journal batch views)`; a missing dir yields two empty lists."""
    state_dir = os.path.expanduser(state_dir)
    return safety.AuditLog(state_dir).entries(), safety.Journal(state_dir).views()


def _safe(value):
    """A tool name, op name or batch id; anything odd is dropped rather than echoed."""
    return value if isinstance(value, str) and re.fullmatch(r"[A-Za-z0-9_:.+\-]{1,64}", value) \
        else "?"


def _candidate(key, change, target, evidence, priority, type_="new"):
    return {"key": key, "change": change, "target": target, "evidence": list(evidence),
            "count": len(evidence), "type": type_, "priority": priority}


def _window(audit, journal, since, observations):
    """`(audit, journal views, observations, by_batch)` for the entries at or after `since`.

    The one place it filters; `by_batch` maps every batch id of the unfiltered journal.
    """
    keep = _since_filter(since)
    return ([e for e in audit if keep(e.get("ts"))],
            [v for v in journal if keep(v.get("ts"))],
            [o for o in observations if isinstance(o, dict) and keep(o.get("ts"))],
            {v.get("batch_id"): v for v in journal})


def derive(audit, journal, since=None, observations=()):
    """Candidate deltas (list of dicts) for the entries at or after `since`."""
    return _derive(*_window(audit, journal, since, observations))


def _derive(audit, views, observations, by_batch):
    """Candidates for entries already restricted to the window; `by_batch` is unfiltered."""
    out = []

    groups = {}
    for entry in audit:
        reason = entry.get("reason") if isinstance(entry.get("reason"), str) else ""
        if entry.get("outcome") == "refused":
            for key, needles, change, target, prio in _REFUSALS:
                if any(needle in reason for needle in needles):
                    # the reasons of one class (and its first row's advice) share a candidate
                    group = groups.setdefault("refusal:" + key, (change, target, prio, []))
                    group[3].append("audit %s %s refused" % (_safe(entry.get("ts")),
                                                              _safe(entry.get("tool"))))
                    break
    for key, (change, target, prio, evidence) in groups.items():
        out.append(_candidate(key, change, target, evidence, prio))

    for verdict, (change, target, prio) in _VERDICTS.items():
        evidence = ["batch %s (%s, %s)" % (_safe(v.get("batch_id")), _safe(v.get("tool")),
                                           _safe(v.get("ts")))
                    for v in views if v.get("verdict") == verdict]
        if evidence:
            out.append(_candidate("verdict:" + verdict, change, target, evidence, prio))

    doubt = ["batch %s (%s, %s)" % (_safe(v.get("batch_id")), _safe(v.get("tool")),
                                    _safe(v.get("ts")))
             for v in views if v.get("state") == "in-doubt"]
    if doubt:
        out.append(_candidate("in_doubt", "Add a guided in-doubt recovery step: inspect the "
                              "station, then decide on retry or rollback",
                              "METHODOLOGY.md · section 4 step 6", doubt, "HIGH"))

    box = {}
    for entry in audit:
        if entry.get("outcome") != "error":
            continue
        reason = entry.get("reason") if isinstance(entry.get("reason"), str) else ""
        batch = by_batch.get(entry.get("batch_id"))
        if batch and "station call failed" in reason:
            ops = sorted({_safe(op.get("nm")) for op in batch.get("ops") or []
                          if isinstance(op, dict)}) or [_safe(entry.get("tool"))]
        elif reason.startswith("BoxError"):
            ops = [_safe(entry.get("tool"))]
        else:
            continue
        for op in ops:
            box.setdefault(op, []).append("audit %s %s error%s" % (
                _safe(entry.get("ts")), _safe(entry.get("tool")),
                " batch %s" % _safe(entry["batch_id"]) if entry.get("batch_id") else ""))
    for op, evidence in sorted(box.items()):
        out.append(_candidate("box_error:" + op, "Document or handle the BOX error seen on "
                              "op %s (cause and safe retry rule)" % op,
                              "METHODOLOGY.md · section 3", evidence, "MEDIUM"))

    seen = [o for o in observations if o.get("count")]
    if seen:
        evidence = ["audit-less read %s n4_find_dangling_outputs found %s" % (
            _safe(o.get("ts")), int(o["count"])) for o in seen]
        cand = _candidate("dangling", "Make the dangling-output check part of the plan "
                          "(link or remove the unused out slots it reports)",
                          "METHODOLOGY.md · section 4 step 7", evidence, "MEDIUM")
        out.append(cand)
    return out


def cell(text):
    """One markdown table cell: no pipes, no line breaks."""
    return str(text).replace("|", "\\|").replace("\n", " ").replace("\r", " ")


def _evidence_cell(evidence):
    shown = evidence[:_MAX_EVIDENCE]
    extra = len(evidence) - len(shown)
    return "; ".join(shown) + ("; +%d more" % extra if extra > 0 else "")


def _delta_table(candidates):
    rows = [DELTA_HEADER, "|---|---|---|---|---|---|"]
    for i, c in enumerate(candidates, 1):
        rows.append("| %d | %s | `%s` | %s | %s | %s |" % (
            i, cell(c["change"]), cell(c["target"]), cell(_evidence_cell(c["evidence"])),
            c["type"], c["priority"]))
    return "\n".join(rows)


def _summary(audit, journal, since):
    outcomes = {}
    for e in audit:
        outcomes[e.get("outcome")] = outcomes.get(e.get("outcome"), 0) + 1
    verdicts = {}
    for v in journal:
        key = v.get("verdict") if v.get("state") != "in-doubt" else "in-doubt"
        verdicts[key] = verdicts.get(key, 0) + 1
    fmt = lambda d: ", ".join("%s %d" % (_safe(k), n) for k, n in sorted(
        d.items(), key=lambda kv: str(kv[0]))) or "none"
    lines = ["- Window: %s" % ("since %s" % since if since else "whole state directory"),
             "- Audit entries: %d (%s)" % (len(audit), fmt(outcomes)),
             "- Journaled batches: %d (%s)" % (len(journal), fmt(verdicts))]
    return "\n".join(lines)


def render(candidates, station, date, audit=(), journal=(), since=None, template=None):
    """Retro markdown from the template; the honesty line replaces an empty table."""
    with open(template or TEMPLATE, encoding="utf-8") as fh:
        text = fh.read()
    if candidates:
        deltas = _delta_table(candidates)
        verdict = ("Draft with %d candidate delta(s) derived mechanically from the audit and "
                   "journal. The reviewer must confirm each is a genuine kit gap rather than "
                   "operator error before keeping it; drop the rest. A retro that always finds "
                   "something is noise (honesty clause)." % len(candidates))
        covered = "- (reviewer: list the guards that worked as designed and the rules the kit " \
                  "already encodes)"
    else:
        deltas = HONESTY
        verdict = HONESTY
        covered = "- none recorded: no friction was found in the audit or the journal."
    values = {"STATION": station, "DATE": date, "DELTAS": deltas, "VERDICT": verdict,
              "COVERED": covered, "SUMMARY": _summary(list(audit), list(journal), since)}
    for key, value in values.items():
        text = text.replace("{{%s}}" % key, value)
    return text


def draft(state_dir, station="unknown", date=None, since=None, observations=(),
          template=None):
    """`{"markdown", "candidates", "station", "date"}` for the entries in `state_dir`."""
    audit, journal = load(state_dir)
    windowed, views, seen, by_batch = _window(audit, journal, since, observations)
    candidates = _derive(windowed, views, seen, by_batch)
    date = date or datetime.datetime.now(datetime.timezone.utc).date().isoformat()
    markdown = render(candidates, station, date, windowed, views, since, template)
    return {"markdown": markdown, "candidates": candidates, "station": station, "date": date}


def count_deltas(markdown):
    """Table rows under `## Proposed kit deltas`; an honesty line counts as zero."""
    block = re.search(r"^## Proposed kit deltas\n(.*?)(?=^## |\Z)", markdown, re.S | re.M)
    if not block:
        return 0
    rows = [ln for ln in block.group(1).splitlines() if ln.startswith("|")]
    return sum(1 for ln in rows[2:]) if len(rows) >= 2 and rows[1].startswith("|---") else 0
