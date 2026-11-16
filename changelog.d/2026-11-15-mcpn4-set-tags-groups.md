### Added — n4_set_tags applies tag groups and relations

- `add_groups` / `remove_groups` apply or remove a tag group an installed dictionary defines (an `n:tagGroup`
  relation to the group's definition, flag `r`); the read-back lists the target's implied tags and any group
  tag missing from them.
- `add_relations` / `remove_relations` add or remove an outbound relation to another component; the inverse of
  an added relation removes the slot the station named.
