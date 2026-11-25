### Changed

- mcp-n4 docs: the permissions `n4_ack_alarm` needs, measured live with an operator-only user on 4.14.0.162: operatorWrite on the alarm class is enough to acknowledge; the record reads need operatorRead on the alarm database, which stays in the default category `User` even with an `alarm:` line in the `CategoryService` ordMap. (#986 D18)
