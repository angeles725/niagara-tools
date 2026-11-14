### Fixed — atomicfile keeps private and mode-000 targets private
- An existing target with mode 000 keeps 000 (it was rewritten as 0644), and the temporary copy of an existing target is created 0600, so a private file's new content is never readable wider than the file while it is written.
