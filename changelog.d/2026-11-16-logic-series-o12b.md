### Fixed — logic series: BOM header, out-of-range time, ASCII tolerance

- `n4kit.series` ignores a UTF-8 byte order mark before the header (a series saved by a Windows editor).
- A row time beyond the long range is refused with its line (`time '…' is out of range (ms)`), never a raw
  `NumberFormatException`.
- A mismatch message shows the tolerance as `+/-<tol>` (ASCII), so a console without UTF-8 prints it intact.
- The header parser reads each column in its own method (readability; behavior unchanged).
