### Fixed — mcp-n4: the shared argument validator bounds its did-you-mean and keeps the closing quote (retro row M11b)

- An unknown argument name is compared by edit distance only when it is at most 64 bytes long and its length is
  within the allowed distance of a declared key: a 1 MB unknown name is now refused at once, in a bounded number
  of allocations (it cost about four million before).
- A refusal clips the caller's value or argument name before quoting it, so the echo keeps its closing quote:
  `got 'xxx…'`, not `got 'xxx…`.
- `n4_add_history` (`mode`, `ext_type`, `full_policy`), `n4_add_alarm` (`kind`), `n4_read_tags` (`relations`) and
  `n4_commissioning_report` (`sections`) drop their own enum checks, unreachable since the shared validator
  refuses these values with -32602; a test pins each schema enum to the table the tool maps it with.
