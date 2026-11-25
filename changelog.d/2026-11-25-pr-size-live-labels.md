### Changed

- PR size reads the pull request's current labels through the API, so rerunning the check after adding `size:exception` passes; a failed lookup keeps the event payload (#1027).
