<!-- ledger: kit -->
<!-- date: 2026-11-14 -->
<!-- last-change-note: retro double-station-save filed (per-component save ticket yields one station save per module) -->
log: 2026-11-14 · retro `double-station-save` filed from a client A station log → one operator change produced two `config.bog` saves 2 s apart, one requested by the control module and one by its dashboard module; both follow `types/logic.md` § Restart & persistence, which prescribes one debounced ticket per component and is silent on several components in one station. No kit behaviour changes in this entry.
open_issue: retro double-station-save Δ1-Δ3 pending (station-wide save rule; one shared reference scheduler with a cross-owner guard; live check counts saves)
