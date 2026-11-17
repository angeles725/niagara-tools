### Added — n4_station_health device_detail (down-device report)

- `n4_station_health drivers=true device_detail=true` adds, per bad device, `fault_cause`, `last_fail_cause`,
  `last_ok`, `last_fail`, `address` (known per-type slot only) and `stale_points`, and per network `port` and a
  `pattern` hint (`bus` when all but at most one device are down, `single` when one device alone is down).
  Two more BQL requests at most, whatever the device count; unreadable values are `null`. `device_detail`
  without `drivers` is refused; `drivers=true` alone is unchanged. Field retro delta U12.
