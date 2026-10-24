"""Regenerate the Python-written state dir the Go port must read identically.

Runs the Python server (writes allowed) against FakeStation and copies the journal.jsonl and
audit.jsonl it wrote into python-written/. The batches cover a completed create, a rollback of
it, a remove and its rollback (relink and component write-ahead records), an execution whose
send failed after the token was spent (an intent without a result) and a link whose reply was
ambiguous (a result flagged in_doubt). Run from anywhere: python3 generate.py
"""
import json
import os
import shutil
import sys
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
TESTS = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, os.path.dirname(TESTS))
sys.path.insert(0, TESTS)

from fake_station import FakeStation  # noqa: E402
from mcp_n4 import box, server  # noqa: E402

FOLDER = "station:|slot:/Folder"
NO_SLEEP = dict(sleep=lambda s: None)


def main():
    fake = FakeStation(password="pw").start()
    tmp = tempfile.mkdtemp()
    state = os.path.join(tmp, "state")
    srv = server.Server(allow_writes=True, allow_http=True, state_dir=state,
                        env={"MCP_N4_USER": "admin", "MCP_N4_PASSWORD": "pw"},
                        stations={"FakeStation": fake.url}, write_scopes=[FOLDER])
    seq = iter(range(1, 1000))

    def call(tool, **args):
        msg = {"jsonrpc": "2.0", "id": next(seq), "method": "tools/call",
               "params": {"name": tool, "arguments": args}}
        return srv.dispatch(msg)["result"]

    def write(tool, **args):
        plan = call(tool, **args)["structuredContent"]
        return call(tool, dry_run=False, confirmation_token=plan["confirmation_token"], **args)

    try:
        call("n4_connect", station="FakeStation", expected_station="FakeStation")
        client = box.BoxClient(fake.url, "admin", "pw", allow_http=True)
        client.open()
        a = write("n4_create_component", parent_ord=FOLDER, name="A",
                  type="kitControl:NumericConst")["structuredContent"]
        write("n4_rollback", batch_id=a["batch_id"])
        nn = client.add_component("3", "Grp", "baja:Folder", ws=box.ws_annotation(1, 2, 3))["nn"]
        gh = client.load_tree(FOLDER + "/" + nn, depth=1, **NO_SLEEP)[""]["h"]
        client.add_component(gh, "Src", "kitControl:NumericConst")
        client.add_component(gh, "Tgt", "kitControl:NumericConst")
        nodes = client.load_tree(FOLDER + "/" + nn, depth=2, **NO_SLEEP)
        client.check_links(nodes["Src"]["h"], "out", nodes["Tgt"]["h"], "in10")
        removed = write("n4_remove_component", parent_ord=FOLDER, name=nn)["structuredContent"]
        write("n4_rollback", batch_id=removed["batch_id"])
        fake.hook = lambda frame: (500, b"boom", {}) if any(
            m.get("b", {}).get("sck") == "syncTo" for m in frame["m"]) else None
        write("n4_create_component", parent_ord=FOLDER, name="C", type="kitControl:NumericConst")
        fake.hook = None
        src = client.add_component("3", "L1", "kitControl:NumericConst")["nn"]
        tgt = client.add_component("3", "L2", "kitControl:NumericConst")["nn"]
        fake._check_link = lambda arg: [{"v": True, "r": None, "s": ""}]
        write("n4_create_link", source_ord=FOLDER + "/" + src, source_slot="out",
              target_ord=FOLDER + "/" + tgt, target_slot="in10")
        client.close()
        out = os.path.join(HERE, "python-written")
        os.makedirs(out, exist_ok=True)
        for name in ("journal.jsonl", "audit.jsonl"):
            shutil.copyfile(os.path.join(state, name), os.path.join(out, name))
        with open(os.path.join(state, "journal.jsonl")) as fh:
            phases = [json.loads(line)["phase"] for line in fh]
        print("wrote %s: %d journal lines (%s)" % (out, len(phases), ", ".join(phases)))
    finally:
        srv.ctx.close()
        fake.stop()
        shutil.rmtree(tmp)


if __name__ == "__main__":
    main()
