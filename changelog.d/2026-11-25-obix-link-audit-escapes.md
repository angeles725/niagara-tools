### Fixed
- obix-link-audit refuses a `--component` holding a raw space, quote, `&` or non-ASCII character and names the
  escaped form (`$20` for a space); it used to send it URL-encoded and report the 404 as "GET failed" (#938).
