# lab-sim

A field-device simulator for the laboratory. The laboratory station has no field device, so
device discovery, proxy points, polling and writes to a device cannot be proven live. `lab-sim`
is a device we control: a scenario file says which devices and points exist and how they
behave, a request log shows what a driver really put on the wire, and a control port lets a
test read or change a point without going through the protocol under test.

Modbus TCP is the only protocol today. Standard library only.

## Build and place

```
cd go && go build -o bin/ ./cmd/lab-sim                                  # Linux / WSL
cd go && GOOS=windows GOARCH=amd64 go build -o <dir>/lab-sim.exe ./cmd/lab-sim   # Windows
```

A station that runs on Windows reaches the simulator on the Windows loopback, so run the
Windows executable there. Copy `lab-sim.exe` and the scenario to a directory of your own on the
Windows side (for example `C:\lab\lab-sim\`), outside the Niagara installation and outside any
station home. From WSL the same executable can be started and driven through interop
(`/mnt/c/lab/lab-sim/lab-sim.exe run ...`, `lab-sim.exe get ...`): `get`, `set`, `status`,
`stop` and `probe` must run on the side the simulator runs on, because they only talk to
loopback.

## Commands

| Command | Does |
|---|---|
| `lab-sim run [--listen HOST:PORT] [--control HOST:PORT] [--log FILE\|-] <scenario.json>` | serves the scenario until Ctrl-C, SIGTERM or `lab-sim stop` |
| `lab-sim check <scenario.json>` | validates the scenario (one line per fault, exit 1) and prints the register map |
| `lab-sim get [--control HOST:PORT] <device> <point>` | prints the point's value now |
| `lab-sim set [--control HOST:PORT] <device> <point> <value>` | holds the point at the value (it replaces a behaviour, and works on read-only points) |
| `lab-sim status [--control HOST:PORT]` | one JSON line: addresses, elapsed time, request and connection counters, each device with its reads and whether it is offline |
| `lab-sim stop [--control HOST:PORT]` | stops the simulator |
| `lab-sim probe [--addr HOST:PORT] [--unit N] [--timeout-ms N] --fn F --address A (--count N \| --values V[,V...])` | sends one Modbus request to a simulator on this machine and prints the answer as JSON |

Exit codes: 0 done, 1 failed (invalid scenario, refused address, no answer, an exception
response), 2 usage.

`run` prints one line when it is ready, with the addresses it really listens on (useful with
port 0):

```
{"control":"127.0.0.1:1503","event":"listening","listen":"127.0.0.1:1502","scenario":"air-handling unit"}
```

**Binding.** The scenario's `listen` address is used only when it is loopback. Anything else
(another interface, `0.0.0.0`) needs an explicit `--listen HOST:PORT` on the command line, so a
scenario file alone never exposes the simulator. The control port is loopback-only, with no
exception: `run`, `get`, `set`, `status` and `stop` refuse any other control address, and
`probe` refuses any other target.

## Scenario file

JSON; an unknown field is a fault. `examples/ahu.json` (an air-handling unit, mixed types) and
`examples/meter.json` (energy meters with float32 registers, one slow and one flaky) are
complete examples.

| Field | Default | Meaning |
|---|---|---|
| `name` | | shown by `check`, `status` and the ready line |
| `listen` | `127.0.0.1:1502` | Modbus TCP address |
| `control` | `127.0.0.1:1503` | control port, loopback only |
| `unknown_unit` | `silent` | a request for a unit id no device has: `silent` (no response) or `gateway_exception` (exception 0x0B) |
| `max_connections` | 8 | connections served at once (1-1024); one more is closed at once |
| `idle_timeout_ms` | 60000 | how long a connection may take to deliver its next whole frame before it is closed |
| `devices` | | at least one |

A device: `name` (unique), `unit` (1-255, unique), `points`, optional `faults`.

A point:

| Field | Default | Meaning |
|---|---|---|
| `name` | | unique in the device |
| `table` | | `coil`, `discrete_input`, `input_register`, `holding_register` |
| `address` | | 0-based protocol address (holding register 0 is reference 40001) |
| `type` | `bit` for the bit tables, `uint16` for the register tables | `bit`, `int16`, `uint16`, `int32`, `uint32`, `float32`; the 32-bit types take two addresses |
| `order` | `ABCD` | where the bytes A B C D of the big-endian value go on the wire |
| `value` | 0 | initial value; an integer type rounds and saturates |
| `writable` | false | accept protocol writes (coils and holding registers only) |
| `behaviour` | constant | see below; not on a `bit`, not on a writable point |
| `faults` | | `fail_after_reads`, `delay_ms` for requests that touch this point |

Points of one table must not overlap. Only mapped addresses exist: a request that touches an
address with no point is answered with exception 2.

| `order` | First register | Also called | 16-bit value |
|---|---|---|---|
| `ABCD` | high word, high byte first | big-endian, 3-2-1-0 | AB |
| `CDAB` | low word first | word swap, 1-0-3-2 | AB |
| `BADC` | high word, bytes swapped | byte swap, 2-3-0-1 | BA |
| `DCBA` | low word, bytes swapped | little-endian, 0-1-2-3 | BA |

Behaviours follow the time since the simulator started, so a value is the same in every run:

| `kind` | Fields | Value |
|---|---|---|
| `constant` (or no `behaviour`) | | `value`, until written or `set` |
| `ramp` | `min`, `max`, `period_ms` | climbs from `min` to `max` over the period, then starts again (`value` is not used) |
| `sine` | `amplitude`, `period_ms` | `value` plus or minus `amplitude` |
| `random_walk` | `seed`, `step`, `interval_ms`, `min`, `max` | starts at `value`; every interval moves by at most `step`, within `min`-`max`; the seed fixes the whole sequence |

Faults:

| Field | On | Effect |
|---|---|---|
| `fail_after_reads: N` | device, point | after N answered reads, every request that touches it gets exception 4 (server device failure) |
| `delay_ms: D` | device, point | the response is held D ms (the longest of the device's and the touched points'); use it to exercise a driver's timeout |
| `offline: [{"from_ms": t1, "to_ms": t2}]` | device | between t1 and t2 after the start the device does not answer at all |

`lab-sim check` prints what an integrator types into a driver:

```
DEVICE  UNIT  POINT             TABLE             ADDRESS  REFERENCE    TYPE     ORDER  ACCESS      INITIAL  BEHAVIOUR
ahu-1   1     fan_command       coil              0        00001        bit      -      read/write  1        constant
ahu-1   1     outdoor_air_temp  input_register    2-3      30003-30004  float32  ABCD   read-only   5        ramp 5..25 / 600000 ms
ahu-1   1     fan_speed_pct     holding_register  2-3      40003-40004  float32  ABCD   read/write  42.5     constant
```

`ADDRESS` is the 0-based protocol address; `REFERENCE` is the 1-based notation of manuals and
many drivers (0xxxx coils, 1xxxx discrete inputs, 3xxxx input registers, 4xxxx holding
registers; six digits once the address passes 9998).

## What the Modbus server answers

| Function | | Limit |
|---|---|---|
| 1, 2 | read coils, read discrete inputs | 1-2000 |
| 3, 4 | read holding registers, read input registers | 1-125 |
| 5, 6 | write single coil, write single register | |
| 15, 16 | write multiple coils, write multiple registers | 1-1968, 1-123 |

| Exception | When |
|---|---|
| 1 illegal function | any other function code |
| 3 illegal data value | a quantity outside the limits, a coil value that is neither FF00 nor 0000, a byte count that disagrees, a truncated request |
| 2 illegal data address | an address with no point, a write that touches a read-only point, a range past 65535 |
| 4 server device failure | `fail_after_reads` reached |
| 0x0B gateway target failed to respond | unknown unit id with `unknown_unit: gateway_exception` |

The checks run in that order. Several unit ids answer behind one listener; requests of one
connection are answered one at a time, in order (a held response delays the next). A frame of
another protocol identifier is discarded; a length no frame can have closes the connection.

## Request log

`--log FILE` appends one JSON line per request (`--log -` writes them to standard output):

```
{"time":"...","elapsed_ms":4210,"remote":"127.0.0.1:53124","unit":1,"device":"ahu-1","function":6,"address":1,"quantity":1,"written":[60],"result":"ok"}
{"time":"...","elapsed_ms":4980,"remote":"127.0.0.1:53124","unit":1,"device":"ahu-1","function":4,"address":0,"quantity":2,"result":"ok","read":[181,224]}
{"time":"...","elapsed_ms":5100,"remote":"127.0.0.1:53124","unit":1,"device":"ahu-1","function":3,"address":99,"quantity":1,"result":"exception","exception":2,"reason":"illegal_data_address"}
```

`result` is `ok`, `exception` (with `exception` and `reason`) or `no_response` (`reason`
`unknown_unit` or `offline`). `written` and `read` are raw registers, or coils as 0/1;
`delay_ms` is how long the response was held.

## Control port

One TCP connection carries one JSON request line and one JSON response line:

```
{"op":"get","device":"ahu-1","point":"fan_speed_pct"}      -> {"ok":true,"value":42.5}
{"op":"set","device":"ahu-1","point":"filter_alarm","value":1}
{"op":"status"}   {"op":"stop"}                            -> {"ok":false,"error":"..."} on a refusal
```

A loopback TCP port was chosen over a file-based channel because it answers within the call
(a test reads its own write back with no polling and no stale file), it behaves the same on
Linux and Windows, and it also works when the Windows executable is driven from WSL. It has no
authentication, which is acceptable only because it is loopback-only: both ends refuse any
other address, and a line that is not a control request (an HTTP request line, for example) is
refused. Anyone who can run a program on the machine can use it; do not run the simulator on a
shared host.

## Prove a Modbus network against it

1. `lab-sim check scenario.json` and keep the register map.
2. `lab-sim run --log requests.jsonl scenario.json` on the station's machine; note the `listen`
   address of the ready line. `lab-sim probe --fn 3 --address 0 --count 1` shows it answering.
3. In the station, create a Modbus TCP network and a device at that address and unit id, then a
   proxy point per row of the map (register type, address, data type, byte order). On a
   laboratory station this is what the `mcp-n4` tools do, each write with its dry run.
4. Reads: compare each proxy point with `lab-sim get <device> <point>`; change one with
   `lab-sim set` and watch the next poll pick it up. The log shows the function code, address
   and quantity the driver really asked (and how it groups registers).
5. Writes: write a writable point from the station; `lab-sim get` shows the value and the log a
   function 5, 6, 15 or 16 with `written`.
6. Faults: a device with `delay_ms` above the driver's timeout, an `offline` window, a
   `fail_after_reads`, an unknown unit id; check what status the station gives the device and
   its points, and that it recovers.
7. `lab-sim stop`.

This outline was not run against a station when the simulator was written.

## Limits

Modbus TCP only: no RTU, no RTU-over-TCP, no function codes beyond the eight above (no
diagnostics, no device identification, no file records, no read/write multiple, no mask
write), no broadcast, no TLS. Timing is whole-response delay only, not inter-byte.

## Layout

| Path | Holds |
|---|---|
| `command.go`, `run.go`, `check.go`, `ctl.go`, `probe.go` | the command: dispatch, composition and one file per sub-command |
| `messages.go`, `usage.txt` | every text it prints, as data |
| `modbus/` | the wire codec: MBAP frame, request and response PDUs, exceptions, limits |
| `model/` | devices and points, free of any protocol: behaviours and faults |
| `regmap/` | the Modbus register map: tables, encodings, byte orders, reads and writes of a range |
| `scenario/` | the scenario file: its shape, defaults, validation messages, and building devices from it |
| `mbserver/` | the Modbus TCP server |
| `reqlog/` | the request log |
| `control/` | the control port, server and client |
| `loopback/` | the one loopback check |
| `examples/` | the two example scenarios |

Another protocol (BACnet/IP is the next) reuses `model`, `control`, `loopback` and the command;
it adds its own fields to the scenario's devices and points, its own codec and server
packages, and one more entry in the list of front ends `run.go` serves.
