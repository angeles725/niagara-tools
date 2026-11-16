<!-- ledger: kit -->
<!-- date: 2026-11-16 -->
<!-- last-change-note: alarm guidance: toNormal rewrites msgText; event values live in persisted slots (logic.md LC9) -->
log: 2026-11-16 · alarm doctrine → `types/logic.md` LC9: the toNormal transition merges the normal facets over the same alarm record, and `BAlarmSourceExt` puts `msgText` = `toNormalText` (empty by default), so the offnormal text is gone after normalization; a value that must outlive the event lives in a persisted `SUMMARY | READONLY` slot reset by acknowledge and reaches both texts through an `alarmData` key; a sub-delay spike leaves no value. The same rule as a live-rules row in `mcp-n4-kit/METHODOLOGY.md` § 3. Retro retros/2026-11-16-alarm-msgtext-tonormal.md (pending). [ev: retro alarm-msgtext-tonormal]
open_issue: the Pattern A route that copies an event value into the alarm ext's `metaData` facets is unproven live (retro alarm-msgtext-tonormal Δ1).
