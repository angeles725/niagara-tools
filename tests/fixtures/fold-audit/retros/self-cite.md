<!-- review-status: folded -->
# Self-cite only retro — fold-audit fixture

This retro cites itself [ev: retro self-ref-only] but this file lives inside
retros/ which is excluded from the corpus scan. The fold-audit script must NOT
find this citation — the stem self-ref-only must still produce a WARN.
