### Security — the scope-root rollback exception covers create batches only

- The same-scope rollback of the write-scope root (added the same day) also admitted copy, replicate and
  deploy batches that named a component after the root, and could delete the root with hand-made children
  inside. It now applies only to `n4_create_component` batches, whose rollback refuses when anything else
  sits inside the root.
