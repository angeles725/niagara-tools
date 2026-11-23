### Added — lint-mqtt-tls-port

- New `toolbelt/lint-mqtt-tls-port.sh [--strict] <station-dir>...` (WARN, advisory, ledger item G87): a
  `GenericMqttAuthenticator` of the `abstractMqttDriver` module that connects over TLS (the default
  `UserLoginOverSSL`, or `AnonymousOverSSL`) and leaves `brokerPort` at 1883 (`types/cloud-connector.md` § 9). Reads
  every `*.bog` under the station directory. Not run by `report-module`.
- `xmlsrc.Doc` gains `SlotPath` and `SlotValue`, which `lint-bacnet-status-flags` now uses instead of private copies.
