package box

import "math/big"

// statusFlagNames are the flags of a baja:Status by bit, lowest first: the names and the order
// a station shows them in ("{alarm,unackedAlarm}").
var statusFlagNames = []string{"disabled", "fault", "down", "alarm", "stale", "overridden", "null", "unackedAlarm"}

// StatusFlags names the flags set in an encoded status (the hex bits ParseStatus reads), lowest
// bit first; none for an ok status. Bits no flag names (or negative bits) follow as one hex
// entry, so the list is empty exactly when ParseStatus says ok. It fails where ParseStatus fails.
func StatusFlags(v any) ([]string, error) {
	bits, err := statusBits(v)
	if err != nil {
		return nil, err
	}
	flags := []string{}
	if bits.Sign() < 0 {
		return append(flags, bits.Text(16)), nil
	}
	for i, name := range statusFlagNames {
		if bits.Bit(i) == 1 {
			flags = append(flags, name)
		}
	}
	if rest := new(big.Int).Rsh(bits, uint(len(statusFlagNames))); rest.Sign() != 0 {
		flags = append(flags, "0x"+rest.Lsh(rest, uint(len(statusFlagNames))).Text(16))
	}
	return flags, nil
}
