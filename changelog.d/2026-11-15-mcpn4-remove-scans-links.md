### Changed — n4_remove_component scans for outgoing links by default

- `n4_remove_component` always scans for links from the removed subtree into components outside it:
  by default the `--write-scope` prefix that holds `parent_ord`, to depth 6; `link_scan_ord` now
  narrows that root instead of enabling the scan. The note names the scanned root and says that
  links stored outside it are not detected.
- When the scan finds links, the dry run refuses and lists each one (`target (target_slot <-
  source_slot)`) until the call repeats with the new argument `acknowledge_links: true`; a remove
  that executed before now needs that acknowledgement. A subtree with no such links removes as
  before.
