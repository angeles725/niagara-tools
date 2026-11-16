<!-- ledger: kit -->
<!-- date: 2026-11-16 -->
<!-- last-change-note: lint-ntpl-manifest-deps stops a run after 100,000 entries read, not only after 128 MiB inflated -->
log: 2026-11-16 · ntpl lint T34 → `lint-ntpl-manifest-deps` stops a run after 100,000 entries read as well as after 128 MiB inflated, with the row "100000 entries read in all, the bound of one run -- <entry> and everything after it not read" (tiny entries drew nothing from the byte budget: 100 templates of 10,000 four-byte Px entries took 12.4 s, now 0.78 s); the amplifying-jar test derives its stop from the bounds (#716 review advisories). Retro t34-ntpl-advisories (Δ1-Δ2 applied, pending review). [ev: retro t34-ntpl-advisories Δ1] [ev: retro t34-ntpl-advisories Δ2]
