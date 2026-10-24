<!-- ledger: kit -->
<!-- date: 2026-10-24 -->
<!-- last-change-note: quality round 10: slotomatic skip scan ignores srcTest/ -->
log: 2026-10-24 · quality round 10 → `toolbelt/lib/slotomatic-skip.sh` ignores a Slot-o-Matic skip under `srcTest/` (a plain `B*Test.java` has no metadata block and test sources never reach the module jar), so CompPan no longer exits 34 on its tests; a `src/` skip still FAILs (bats SK-srctest-only, SK-srctest-mixed). [ev: retro r10-slotomatic-skip-srctest]
