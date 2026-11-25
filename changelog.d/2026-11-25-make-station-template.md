### Changed

- mcp-n4: new write tool `n4_make_station_template(use_minor_version?, service_ord?, wait_seconds?)` (#935,
  retro R13 ΔM6). It invokes the `template:TemplateService`'s hidden `makeStationTemplate` action, waits for the
  `BMakeStationTemplateJob` under `Services/JobService` and returns `job_state`, the job `log` and `file_ord`,
  the `file:^temp/...` file the job saved, which `n4_read_file` fetches. Station code cited in the tool's doc
  comment (`BTemplateService.doMakeStationTemplate`, `BMakeTemplateJob`).
- It writes a file on the station, so it is a dry run by default under a confirmation token, scoped on the
  TemplateService ORD, journaled, and the plan names the `^temp` file and says there is no inverse. Only a super
  user gets a job: any other user gets null and an `unverified` read-back. A failed job or one with no file is a
  `mismatch`; a station without the template module is refused at the dry run.
