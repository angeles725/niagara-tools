### Added — n4_set_flags add and remove

- `n4_set_flags` takes `add` and/or `remove` (flag letters) instead of `flags`: the letters change in
  the slot's current set and the others are kept, so making a slot a summary slot (`add: "s"`) no
  longer drops a readonly `r` the caller did not restate. One form per call; a letter in both `add`
  and `remove`, an unknown or repeated letter, or an empty edit is refused before the station is
  read. The plan note still shows the current and the resulting set, and the inverse stays the exact
  previous string. `flags` is no longer a required argument; the whole-set form is unchanged.
