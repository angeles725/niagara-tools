#!/usr/bin/env bats
# RED-FIRST pins for verify-module.sh --plano (campaign 7 PR6, issue #47, from B797 · companero).
# Contract: niagara-research bloque797.md §797.2 [CERT], derived from the real DashboardPan-ux SPA.
#
# For a plano overlay in the HTML:
#   Ri = #planoImg <image>/<img> INTRINSIC size (decode the data: PNG header)
#   Rc = IMG_W / IMG_H            (const IMG_W=<a>, IMG_H=<b>)
#   Rv = vbW / vbH               (zones <svg … viewBox="0 0 <vbW> <vbH>">)
#   A  = every NUMERIC `aspect-ratio: n/m` in the file (aspect-ratio:auto is EXEMPT)
# PASS iff  Rc == Rv == Ri  AND  every r in A equals Rc  (compare by cross-multiplication w1*h2==w2*h1,
# never float ==). FAIL names the disagreeing value with its line. The real SPA FAILs: a stale
# .frame{aspect-ratio:1247/771} (=1.617) vs the 1248x891 (=1.401) image, masked by #frame{aspect-ratio:auto}.
#
# SURFACE (pin the explicit-file form): verify-module.sh --plano <index.html> -> exit 0 PASS / 1 FAIL.
#
# RED today: verify-module.sh has no --plano mode; `--plano` hits the unknown-flag branch -> exit 2, no
# PASS/FAIL row, so every pin fails for the right reason. Green once PR6 lands the check.
#
# NAMED MUTATION (post-green): a COUNT-ONLY check (all aspect-ratios equal EACH OTHER but not compared to
# the image/Rc) -> PL2 and PL4 lose their FAIL, because equality must be against Rc, not intra-aspect-ratio.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  VM="$KIT/toolbelt/verify-module.sh"
  # A 2x3 PNG (intrinsic Ri = 2/3), base64, via python3 zlib.
  B64=$(python3 - <<'PY'
import zlib,struct,base64
def chunk(t,d): return struct.pack('>I',len(d))+t+d+struct.pack('>I',zlib.crc32(t+d)&0xffffffff)
raw=b''.join(b'\x00'+b'\x80'*2 for _ in range(3))
png=b'\x89PNG\r\n\x1a\n'+chunk(b'IHDR',struct.pack('>IIBBBBB',2,3,8,0,0,0,0))+chunk(b'IDAT',zlib.compress(raw))+chunk(b'IEND',b'')
print(base64.b64encode(png).decode())
PY
)
}

# mk_html <file> <IMG_W> <IMG_H> <vbW> <vbH> <aspect-ratio-css-lines...>
mk_html() {
  local f="$1" iw="$2" ih="$3" vw="$4" vh="$5"; shift 5
  { echo "<!doctype html><html><head><style>"
    local n=0; for ar in "$@"; do echo "  .frame$n { aspect-ratio: $ar; }"; n=$((n+1)); done
    echo "</style></head><body>"
    echo "<svg class=\"zonas\" id=\"zonas\" viewBox=\"0 0 $vw $vh\" preserveAspectRatio=\"xMidYMid meet\">"
    echo "  <image id=\"planoImg\" xlink:href=\"data:image/png;base64,$B64\"/>"
    echo "</svg>"
    echo "<script>const IMG_W = $iw, IMG_H = $ih;</script>"
    echo "</body></html>"
  } > "$f"
}

@test "PL1: image 2x3, IMG_W/H 2/3, viewBox 2 3, aspect-ratio 2/3 — all four agree -> PASS (exit 0)" {
  mk_html "$BATS_TEST_TMPDIR/pl1.html" 2 3 2 3 "2/3"
  run "$VM" --plano "$BATS_TEST_TMPDIR/pl1.html"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS"* ]]
}

@test "PL2: a stale numeric aspect-ratio 1247/771 disagreeing with the 2/3 image -> FAIL, names it" {
  mk_html "$BATS_TEST_TMPDIR/pl2.html" 2 3 2 3 "1247/771"
  run "$VM" --plano "$BATS_TEST_TMPDIR/pl2.html"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"1247/771"* ]]
}

@test "PL3: aspect-ratio:auto present + a numeric 2/3 that agrees -> PASS (auto is exempt)" {
  mk_html "$BATS_TEST_TMPDIR/pl3.html" 2 3 2 3 "auto" "2/3"
  run "$VM" --plano "$BATS_TEST_TMPDIR/pl3.html"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS"* ]]
}

@test "PL4: viewBox 4 3 differs from the 2x3 image (Rv != Ri) -> FAIL" {
  mk_html "$BATS_TEST_TMPDIR/pl4.html" 2 3 4 3 "2/3"
  run "$VM" --plano "$BATS_TEST_TMPDIR/pl4.html"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
}
