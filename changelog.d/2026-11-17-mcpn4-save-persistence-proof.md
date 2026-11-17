### Changed — n4_save_station proves persistence without --station-home, and retries a config.bog caught mid-rewrite

- Without `--station-home`, `n4_save_station` now reads the station's own save job: the `baja:StationSaveJob`
  children of `Services/JobService` are listed before the save and polled after it. A new job that ends `success`
  gives `persisted: true` (`evidence.route: station save job`, `evidence.job`); `failed` or `canceled` gives
  `false`; no new job, a job still running at the timeout or an unreadable JobService stays `unknown` with a hint
  (overamps retro row M5). The file servlet never serves `.bog` files, so config.bog metadata is not readable
  remotely.
- With `--station-home`, a `config.bog` that is unreadable right after the save is read once more after 1 s
  (`evidence.retry`), so a save caught between the backup move and the rename of `config.bog.working` is no longer
  reported `unknown` (overamps retro row M9).
