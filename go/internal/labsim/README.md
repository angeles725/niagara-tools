# lab-sim

A field-device simulator for the laboratory. The laboratory station has no field device, so
device discovery, proxy points, polling and writes to a device cannot be proven live. `lab-sim`
is a device we control: a scenario file says which devices and points exist and how they
behave, a request log shows what a driver really put on the wire, and a control port lets a
test read or change a point without going through the protocol under test.

Two protocols: Modbus TCP, and BACnet/IP (one device per UDP socket). The BACnet side was written
from the rules of the public standard and **has not been validated against another BACnet
stack**, nor against a station: read the BACnet section before trusting it. Standard library only.

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
`stop`, `probe` and `bacnet-probe` must run on the side the simulator runs on, because they only
talk to loopback.

## Commands

| Command | Does |
|---|---|
| `lab-sim run [--listen HOST:PORT] [--bacnet-listen [DEVICE=]IP:PORT[,...]] [--bacnet-broadcast IP:PORT] [--control HOST:PORT] [--log FILE\|-] <scenario.json>` | serves the scenario until Ctrl-C, SIGTERM or `lab-sim stop` |
| `lab-sim check <scenario.json>` | validates the scenario (one line per fault, exit 1) and prints the register map and, for BACnet devices, the object list |
| `lab-sim get [--control HOST:PORT] <device> <point>` | prints the point's value now |
| `lab-sim set [--control HOST:PORT] <device> <point> <value>` | holds the point at the value (it replaces a behaviour, and works on read-only points) |
| `lab-sim status [--control HOST:PORT]` | one JSON line: the address of every front end (`front_ends`), elapsed time, request and connection counters, each device with its reads and whether it is offline |
| `lab-sim stop [--control HOST:PORT]` | stops the simulator |
| `lab-sim probe [--addr HOST:PORT] [--unit N] [--timeout-ms N] --fn F --address A (--count N \| --values V[,V...])` | sends one Modbus request to a simulator on this machine and prints the answer as JSON |
| `lab-sim bacnet-probe [--addr HOST:PORT] [--timeout-ms N] [--max-apdu N] <action>` | sends BACnet requests to a simulator on this machine and prints each answer as one JSON line (actions below) |

Exit codes: 0 done, 1 failed (invalid scenario, refused address, no answer, an exception
response, a BACnet error, reject or abort), 2 usage.

`run` prints one line when it is ready, with the addresses it really listens on (useful with
port 0):

```
{"control":"127.0.0.1:1503","event":"listening","listen":"127.0.0.1:1502","scenario":"air-handling unit"}
```

With BACnet devices the line also has `"bacnet":[{"address":"127.0.0.1:47809","device":"ahu-1"}]`.

**Binding.** The scenario's `listen` address is used only when it is loopback. Anything else
(another interface, `0.0.0.0`) needs an explicit `--listen HOST:PORT` on the command line, so a
scenario file alone never exposes the simulator. The same rule holds for each BACnet device with
`--bacnet-listen DEVICE=IP:PORT` (several separated by commas; a bare `IP:PORT` when the scenario
has one BACnet device). The control port is loopback-only, with no exception: `run`, `get`,
`set`, `status` and `stop` refuse any other control address, and `probe` and `bacnet-probe`
refuse any other target.

## Scenario file

JSON; an unknown field is a fault. `examples/ahu.json` (an air-handling unit, mixed types) and
`examples/meter.json` (energy meters with float32 registers, one slow and one flaky) are
complete examples; `examples/replay.json` replays two recorded series (`replay-supply.csv`, a
history export, and `replay-return.csv`, a plain `time,value` file).

| Field | Default | Meaning |
|---|---|---|
| `name` | | shown by `check`, `status` and the ready line |
| `listen` | `127.0.0.1:1502` | Modbus TCP address |
| `control` | `127.0.0.1:1503` | control port, loopback only |
| `unknown_unit` | `silent` | a request for a unit id no device has: `silent` (no response) or `gateway_exception` (exception 0x0B) |
| `max_connections` | 8 | connections served at once (1-1024); one more is closed at once |
| `idle_timeout_ms` | 60000 | how long a connection may take to deliver its next whole frame before it is closed |
| `devices` | | at least one |

A device: `name` (unique), `unit` (1-255, unique), `points`, optional `faults`, optional `bacnet`
(the BACnet section below).

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
| `bacnet` | | the BACnet object the point is (the BACnet section below) |

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
| `replay` | `file`; `speed` (1), `end` (`hold_last`), `start_offset_ms` (0) | plays a recorded series (below; `value` is not used) |

### Replay a recorded series

`replay` plays the records of a CSV file into the point, so a driver or a logic sees a real day
of a real sensor instead of a formula:

```json
{"kind": "replay", "file": "replay-supply.csv", "speed": 60, "end": "loop", "start_offset_ms": 0}
```

| Field | Default | Meaning |
|---|---|---|
| `file` | | the CSV, relative to the scenario file (an absolute path also works) |
| `speed` | 1 | how many times faster than recorded it plays: 60 plays a recorded minute in a second; above 0 |
| `end` | `hold_last` | at the end of the series: `hold_last` keeps the last record, `loop` starts again once the last record has held as long as the one before it |
| `start_offset_ms` | 0 | how far into the recorded series (recorded time) the replay starts; at most the series' span |

The file has a header row. Its columns are found by name, ignoring case and anything that is not
a letter: a time column (`time`, or a name starting with `timestamp`), a value column (a name
starting with `value`) and an optional `status` column; other columns are ignored. So both a plain
`time,value` file and the `Timestamp,Value,Status` columns of a history read as ITableToCsv are
read, in either order of time (a history read comes newest first; the records are sorted).

- A time carries its zone: `2026-11-15T08:00:00.000-06:00` or `2026-11-15T14:00:00Z`. The
  localized display form (`15 nov 2026 08:00:00 CST`) is refused: read the history with
  `select timestamp.encodeToString, value.encodeToString, status` (the query `n4_read_history`
  uses for its summary).
- A value is a plain finite number (`21.5`); a localized `21,5`, units or `NaN` are refused.
- **A record whose status is not ok is skipped**: with a status column, only `ok`, `{ok}` or an
  empty status is played; `{stale}`, `{fault,down}` and every other status are left out and
  counted. A file with no ok record is a fault. `lab-sim check` shows the records played and
  skipped.
- A record holds until the next one, as a history sample does: there is no interpolation.
- At most 1,000,000 records. The values must fit the point's type, as for the other behaviours.

A fault in the file is one line naming the row (the header is row 1) and the column:

```
device 1 "plant-1", point 1 "supply_water_temp_x10": replay file "replay-supply.csv": row 5, column 2 (Value): "21,5" is not a number such as 21.5 (read a history with value.encodeToString)
```

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

## BACnet/IP

**Not validated.** The codec, the object model and the server were written test-first from the
rules of the public standard, with no other BACnet stack and no station at hand. Nothing here
has exchanged a packet with an implementation other than its own probe. The tests that use
frames taken from the standard are listed under "Where the test frames come from"; the first
live run against a station decides what is really right.

A device of the scenario becomes a BACnet device when it has a `bacnet` section, and each of its
points becomes an object when the point has one. BACnet/IP has one device per address, so every
BACnet device has its own UDP socket.

Device `bacnet`:

| Field | Default | Meaning |
|---|---|---|
| `device_instance` | required | 0-4194302, unique in the scenario; the Device object and the I-Am carry it |
| `listen` | `127.0.0.1:47809` | IPv4 address and UDP port, unique in the scenario; loopback only unless `--bacnet-listen` |
| `vendor_id` | 999 | Vendor_Identifier (0-65535); 999 is not a registered vendor of ours, it is a placeholder |
| `vendor_name`, `model` | `lab-sim` | Vendor_Name, Model_Name |
| `firmware`, `application` | `1` | Firmware_Revision, Application_Software_Version |
| `description` | none | Description of the Device object |
| `max_apdu` | 1476 | Max_APDU_Length_Accepted: 50, 128, 206, 480, 1024 or 1476 |
| `cov` | false | accept SubscribeCOV; when false it is answered with an error |

Point `bacnet`:

| Field | Default | Meaning |
|---|---|---|
| `object_type` | required | `analog-input`, `analog-output`, `analog-value`, `binary-input`, `binary-output`, `binary-value`, `multi-state-input`, `multi-state-output`, `multi-state-value` |
| `instance` | required | 0-4194302, unique for the object type in the device |
| `description` | none | Description |
| `units` | `no-units` | analog only: a unit name (`degrees-celsius`, `percent`, `pascals`, `kilowatt-hours`, ...) or its number |
| `commandable` | false | outputs and values: a 16-level priority array and a relinquish default |
| `active_text`, `inactive_text` | none | binary only |
| `state_texts` | required for multi-state | one text per state; the count is Number_Of_States |
| `min_pres_value`, `max_pres_value`, `resolution` | none | analog only; a write outside min-max is refused |
| `cov_increment` | 0 | analog only: the change that causes a notification (0: any change) |

The object's name is the point's name. An object is writable through BACnet when it is
commandable, or when the point is `writable` and the object is not an input. A BACnet point
still needs its Modbus mapping (`table`, `address`): the scenario format has no BACnet-only
point.

`lab-sim check` prints, after the register map, the object list an integrator sees:

```
BACnet device ahu-1: instance 1001, listen 127.0.0.1:47809, vendor 999 "lab-sim", model "lab-sim", max APDU 1476, COV on
DEVICE  OBJECT TYPE    INSTANCE  NAME                 UNITS            COMMANDABLE  ACCESS
ahu-1   device         1001      ahu-1                -                -            read-only
ahu-1   binary-output  1         fan_command          -                yes          read/write
ahu-1   analog-input   3         outdoor_air_temp     degrees-celsius  no           read-only
```

### Listening and broadcasts

`--bacnet-listen DEVICE=IP:PORT[,DEVICE=IP:PORT...]` (or a bare `IP:PORT` when the scenario has
one BACnet device) replaces a device's address, and is the only way to bind anything but
loopback. IPv4 only.

The simulator never sends to a broadcast address by itself. An I-Am goes to whoever asked (by
unicast), to every registered foreign device (as a Forwarded-NPDU) and, only when
`--bacnet-broadcast IP:PORT` names one, to that address as an Original-Broadcast-NPDU. With the
defaults nothing leaves loopback.

Bounds, all fixed: a packet over 1507 octets is dropped; 16 foreign devices; 32 COV
subscriptions; 200 packets a second per source address with at most 64 sources tracked (the rest
are dropped); 64 remembered answers; 32 responses held at once by `delay_ms`.

### What the BACnet device answers

| Layer | Answered | Not answered |
|---|---|---|
| BVLL | Original-Unicast-NPDU, Original-Broadcast-NPDU; Register-Foreign-Device (result success; the entry lives its time to live plus 30 s); Read-Foreign-Device-Table; Delete-Foreign-Device-Table-Entry; Read-Broadcast-Distribution-Table (empty); Distribute-Broadcast-To-Network from a registered foreign device (a NAK from anyone else); Write-Broadcast-Distribution-Table (NAK) | Forwarded-NPDU received (there is no peer BBMD) and Secure-BVLL are ignored |
| NPDU | local network, and the global broadcast network 65535 | a message for another network number, and every network layer message (Who-Is-Router-To-Network included), are ignored |
| Unconfirmed | Who-Is (with or without a range) answered with I-Am (segmentation none); Who-Has by identifier or name answered with I-Have; TimeSynchronization and UTCTimeSynchronization accepted and logged | everything else is ignored |
| Confirmed | ReadProperty, ReadPropertyMultiple, WriteProperty, SubscribeCOV | everything else: Reject (unrecognized-service) |

- **ReadProperty**: an array index on the array properties (`object-list`, `priority-array`,
  `state-text`); index 0 is the length.
- **ReadPropertyMultiple**: `all`, `required` and `optional`, and an error per property inside
  the acknowledgement for a property or an object that is not there.
- **WriteProperty**: `present-value`, and `relinquish-default` of a commandable object. On a
  commandable object a priority of 1-16 (16 when absent) and NULL to relinquish; on a writable
  object that is not commandable the priority is ignored.
- **SubscribeCOV**: confirmed or unconfirmed notifications, a lifetime (0 is indefinite) and
  cancellation, on the point objects (not on the Device). One notification on subscription and
  one on each change: any change of a binary or multi-state value, a change of at least
  `cov_increment` of an analog one. Changes are noticed after a BACnet write, after a control
  `set`, and every 500 ms for behaviours and Modbus writes. A confirmed notification is sent
  once: its acknowledgement is accepted and nothing is resent without one.
- **No segmentation**: a segmented request is aborted, and a response that does not fit the
  smaller of the client's and the device's maximum APDU is answered with Abort
  (segmentation-not-supported). Read a long `object-list` by index.
- The invoke id is echoed. A request repeated with the same invoke id from the same address
  within 10 s gets the first answer again and is not run a second time (logged
  `"duplicate":true`).

| Refusal | When |
|---|---|
| Error object / unknown-object | an object the device has not |
| Error property / unknown-property | a property the object has not |
| Error property / write-access-denied | a write to an input, to a read-only value, or to any other property |
| Error property / value-out-of-range | outside min-max, a binary value above 1, a state outside 1-N, a priority outside 1-16 |
| Error property / invalid-data-type | a value of another type, or more than one value |
| Error property / invalid-array-index, property-is-not-an-array | an index past the array, an index on a property that is no array, any index on a write |
| Error device / operational-problem | `fail_after_reads` reached (device or a touched point) |
| Error services / optional-functionality-not-supported | SubscribeCOV with `cov` false, or on the Device object |
| Error resources / no-space-to-add-list-element | the 33rd subscription |
| Reject | a request that cannot be decoded (missing-required-parameter, invalid-tag, too-many-arguments, ...) |

The faults of the scenario apply as they do on Modbus: an `offline` device answers nothing at
all (no I-Am, no notification), `delay_ms` holds the response, `fail_after_reads` turns every
request that touches the device or point into the error above.

Objects and properties:

| Object | Properties |
|---|---|
| Device | object-identifier, object-name, object-type, system-status (operational), vendor-name, vendor-identifier, model-name, firmware-revision, application-software-version, description (when given), protocol-version (1), protocol-revision (4), protocol-services-supported, protocol-object-types-supported, object-list, max-apdu-length-accepted, segmentation-supported (none), apdu-timeout, number-of-apdu-retries (0), device-address-binding (empty), database-revision |
| every point object | object-identifier, object-name, object-type, present-value, status-flags (all false), event-state (normal), reliability (no-fault-detected), out-of-service (false), description (when given) |
| analog | units; min-pres-value, max-pres-value, resolution (when given); cov-increment (when `cov`) |
| binary | polarity (normal; inputs and outputs); active-text, inactive-text (when given) |
| multi-state | number-of-states, state-text |
| commandable | priority-array, relinquish-default |

Choices that follow no reference and may be wrong for a given client: protocol revision 4 is
claimed (the oldest that covers these services, so no newer required property is missing);
`protocol-services-supported` is 40 bits and `protocol-object-types-supported` 25 bits;
character strings are sent and accepted in character set 0 only; priority 6 is not reserved for
minimum on/off; `out-of-service` cannot be written.

### Control `set` and the priority array

A point has a field value and, for a commandable object, the 16 commands of its priority array.

- An object that is not commandable: present-value is the field value. A control `set`, a
  Modbus write and a BACnet write all change that one value.
- A commandable object: the field value is its **relinquish-default**, and present-value is the
  command of the highest priority, or the relinquish-default when every level is NULL. A control
  `set` (and a Modbus write) changes the relinquish-default: it never writes a level of the
  priority array and never releases one, so it shows in present-value only when nothing is
  commanded.
- `lab-sim get`, the answer of `lab-sim set` and Modbus reads give the **effective** value
  (what present-value gives), so the control port and BACnet never disagree about what the
  device is doing.

To test "the station commands at priority 8, then releases": write through the station, see
`lab-sim get` change, release, and see it return to the value a `set` gave.

### bacnet-probe

`lab-sim bacnet-probe [--addr HOST:PORT] [--timeout-ms N] [--max-apdu N] <action>` talks to a
simulator on this machine (default `127.0.0.1:47809`; any address that is not loopback is
refused) and prints one JSON line per answer. An error, a reject or an abort is exit 1.

```
lab-sim bacnet-probe who-is [LOW HIGH]
  {"service":"i-am","from":"127.0.0.1:47809","device":"device,1001","max_apdu":1476,"segmentation":"none","vendor":999}
lab-sim bacnet-probe read device,1001 object-list 0
  {"object":"device,1001","property":"object-list","index":0,"values":[{"type":"unsigned","value":"15"}]}
lab-sim bacnet-probe read-multiple analog-output,2 all          (also: present-value,units,...)
  {"object":"analog-output,2","results":[{"property":"object-identifier","values":[...]},...]}
lab-sim bacnet-probe write analog-output,2 55 --priority 8       (null relinquishes)
  {"object":"analog-output,2","property":"present-value","priority":8,"written":"55","result":"ok"}
lab-sim bacnet-probe subscribe analog-input,4 --lifetime 120 --count 2
  {"object":"analog-input,4","subscribed":"unconfirmed","process":1,"lifetime":120}
  {"service":"unconfirmed-cov-notification","object":"analog-input,4","device":"device,1001","process":1,"remaining":120,"values":[...]}
lab-sim bacnet-probe read analog-input,99 present-value
  {"object":"analog-input,99","property":"present-value","error":{"class":"object","code":"unknown-object"}}
```

`write` writes present-value; the object type gives the type of the value (analog: a number,
binary: 0 or 1, multi-state: a state number). `subscribe` takes `--mode
unconfirmed|confirmed|cancel`, `--process N`, `--lifetime SECONDS` and `--count N` (it ends
after N notifications, the first being the one sent on subscription). The probe is a way to see
the simulator answer, written by the same hand as the server: it proves nothing about another
stack.

### Reaching it from a station on the same Windows machine

A station's BACnet network holds UDP 47808 on the adapter it is bound to, so the simulator
cannot simply take the same port on the same address. Three routes, **none of them tried**; the
first live run should record which works.

1. **Add the device by hand.** Run with the defaults (`127.0.0.1:47809`) and, in the station,
   add a BACnet device by hand with device instance and address `127.0.0.1` port 47809
   (0xBAC1), then let the driver learn its objects. Not known: whether the station's network,
   bound to another adapter, sends to loopback at all, and whether its device address accepts a
   port other than its own.
2. **The station registers as a foreign device with the simulator.** Run the simulator on an
   address the station can reach (`--bacnet-listen <adapter IP>:47809`, which exposes it to
   that network) and configure the station's BACnet/IP port as a foreign device of that
   address. The station's Who-Is then arrives as Distribute-Broadcast-To-Network and the I-Am
   returns as a Forwarded-NPDU, so discovery should work with no broadcast. Not known: whether
   the station accepts a BBMD on its own adapter address, and whether it accepts our
   Forwarded-NPDU.
3. **A second IP address with the simulator on 47808.** Add a loopback adapter (or a second
   address) to Windows, bind the simulator to it (`--bacnet-listen <second IP>:47808
   --bacnet-broadcast <broadcast of that subnet>:47808`) and put the station's BACnet port on
   the same subnet. Both are then ordinary peers on 47808 and broadcast discovery applies. Not
   known: whether broadcasts are delivered between two sockets of one machine on a loopback
   adapter, and whether the station will bind its BACnet port to that adapter.

With several BACnet devices each needs its own address: different ports for routes 1 and 2, or
one IP address each for route 3.

### Where the test frames come from

From the standard's own examples, typed from memory of ASHRAE 135 and not checked against a
copy while writing (a wrong recollection would make the test and the codec agree on a wrong
frame): the encodings of clause 20.2 (null, boolean, unsigned 72, signed 72, real 72.3, double
72.3, the octet string 1234FF, the character string "This is a BACnet string!", the bit string
10101, enumerated 0, a date, a time, the object identifier binary-input 15), and from Annex F
the ReadProperty request and acknowledgement (F.3.5), the ReadPropertyMultiple request and
acknowledgement (F.3.7), the WriteProperty request and its SimpleACK (F.3.9), and Who-Is and
I-Am (F.4.3).

Constructed by the rules, with no published frame behind them: every BVLL and NPDU frame,
SubscribeCOV and the COV notifications, Who-Has and I-Have, the Error, Reject and Abort PDUs,
extended tag numbers and lengths, and every exchange of the server tests.

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

A BACnet request has its own shape, told apart by `"protocol":"bacnet-ip"` (a Modbus line is
unchanged and has no `protocol` field):

```
{"time":"...","elapsed_ms":79,"protocol":"bacnet-ip","remote":"127.0.0.1:45839","device":"ahu-1","link":"original-unicast-npdu","pdu":"confirmed-request","service":"read-property","invoke_id":1,"object":"device,1001","property":"object-list","index":0,"result":"ok","read":["15"]}
{"time":"...","elapsed_ms":152,"protocol":"bacnet-ip","remote":"127.0.0.1:45653","device":"ahu-1","link":"original-unicast-npdu","pdu":"confirmed-request","service":"write-property","invoke_id":1,"object":"analog-output,2","property":"present-value","priority":8,"written":"55","result":"ok"}
{"time":"...","elapsed_ms":329,"protocol":"bacnet-ip","remote":"127.0.0.1:45249","device":"ahu-1","link":"original-unicast-npdu","pdu":"confirmed-request","service":"read-property","invoke_id":1,"object":"analog-input,99","property":"present-value","result":"error","error_class":"object","error_code":"unknown-object"}
{"time":"...","elapsed_ms":320,"protocol":"bacnet-ip","remote":"127.0.0.1:48036","device":"ahu-1","link":"original-unicast-npdu","pdu":"unconfirmed-request","service":"unconfirmed-cov-notification","object":"analog-input,4","result":"sent","read":["310","0000"],"to":["127.0.0.1:48036"]}
```

`link` is the BVLL function the request came in, `result` one of `ok`, `error` (with
`error_class`, `error_code`), `reject`, `abort`, `ignored`, `refused` (each with `reason`),
`no_response` (`offline`, or a held response that found no room) and `sent` (a notification the
simulator originated). `to` lists where an I-Am, I-Have or notification went, `read` the values
answered (at most 16), `duplicate` a replayed answer. A ReadPropertyMultiple is one line with a
count in `detail`, not its values.

## Control port

One TCP connection carries one JSON request line and one JSON response line:

```
{"op":"get","device":"ahu-1","point":"fan_speed_pct"}      -> {"ok":true,"value":42.5}
{"op":"set","device":"ahu-1","point":"filter_alarm","value":1}
{"op":"status"}   {"op":"stop"}                            -> {"ok":false,"error":"..."} on a refusal
```

`status` lists every front end with its address, one BACnet entry per device:
`"front_ends":[{"protocol":"modbus-tcp","address":"127.0.0.1:1502"},{"protocol":"bacnet-ip","address":"127.0.0.1:47809","device":"ahu-1"},{"protocol":"control","address":"127.0.0.1:1503"}]`.
`get` and `set` work on the same points whatever protocol serves them; for a commandable BACnet
object see "Control `set` and the priority array".

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

Modbus: TCP only, no RTU, no RTU-over-TCP, no function codes beyond the eight above (no
diagnostics, no device identification, no file records, no read/write multiple, no mask
write), no broadcast, no TLS. Timing is whole-response delay only, not inter-byte.

BACnet: BACnet/IP over IPv4 only (no MS/TP, no Ethernet, no IPv6, no BACnet/SC), no
segmentation, no routing (one local network, no router or BBMD peer), no alarms or event
notifications, no trend logs, no schedules or calendars, no files, no device management
(DeviceCommunicationControl, ReinitializeDevice), no COV property subscriptions, no
WritePropertyMultiple, no ReadRange. A confirmed COV notification is not retried. A datagram
over 1472 octets needs IP fragments, which some loopback paths drop: keep `max_apdu` at 1024 or
below if long answers go missing. Not validated against any other stack (the BACnet section).

## Layout

| Path | Holds |
|---|---|
| `command.go`, `run.go`, `run_bacnet.go`, `check.go`, `check_bacnet.go`, `ctl.go`, `probe.go`, `bacprobe*.go` | the command: dispatch, composition and the sub-commands |
| `messages.go`, `usage.txt` | every text it prints, as data |
| `modbus/` | the wire codec: MBAP frame, request and response PDUs, exceptions, limits |
| `model/` | devices and points, free of any protocol: behaviours and faults |
| `regmap/` | the Modbus register map: tables, encodings, byte orders, reads and writes of a range |
| `scenario/` | the scenario file: its shape, defaults, validation messages, the replay CSV reader, and building devices from it |
| `mbserver/` | the Modbus TCP server |
| `bacnet/` | the BACnet wire codec: tags and values, BVLL, NPDU, APDU headers, service data, and the names of services, properties, errors, rejects and aborts as data |
| `bacobj/` | the BACnet object model over the neutral points: property tables as data, reads, writes, the priority array, units |
| `bacserver/` | the BACnet/IP server of one device: link, network and application layers, foreign devices, COV, limits |
| `reqlog/` | the request log: the Modbus entry and the BACnet entry |
| `control/` | the control port, server and client |
| `loopback/` | the one loopback check |
| `examples/` | the three example scenarios and the two recorded series the replay one plays |

Another protocol reuses `model`, `control`, `loopback` and the command, as BACnet/IP does: it
adds its own fields to the scenario's devices and points, its own codec and server packages,
and its entries in the list of front ends `run.go` serves.
