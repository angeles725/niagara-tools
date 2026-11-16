package bacnet

import (
	"fmt"
	"strconv"
	"strings"
)

var kindNames = map[Kind]string{
	KindNull: "null", KindBool: "boolean", KindUnsigned: "unsigned", KindSigned: "signed", KindReal: "real",
	KindDouble: "double", KindOctets: "octet-string", KindString: "character-string", KindBits: "bit-string",
	KindEnum: "enumerated", KindDate: "date", KindTime: "time", KindOID: "object-identifier",
}

// String names the application data type.
func (k Kind) String() string { return name(kindNames, k, "type") }

// String is the value as the request log and the probe print it.
func (v Value) String() string {
	switch v.Kind {
	case KindNull:
		return "null"
	case KindBool:
		return strconv.FormatBool(v.Bool)
	case KindUnsigned, KindEnum:
		return strconv.FormatUint(uint64(v.Uint), 10)
	case KindSigned:
		return strconv.FormatInt(int64(v.Int), 10)
	case KindReal:
		return strconv.FormatFloat(float64(v.Real), 'g', -1, 32)
	case KindDouble:
		return strconv.FormatFloat(v.Double, 'g', -1, 64)
	case KindString:
		return v.Text
	case KindBits:
		var b strings.Builder
		for _, on := range v.Bits {
			if on {
				b.WriteByte('1')
			} else {
				b.WriteByte('0')
			}
		}
		return b.String()
	case KindDate:
		if len(v.Raw) == 4 { // year - 1900, month, day, day of week
			return fmt.Sprintf("%04d-%02d-%02d", 1900+int(v.Raw[0]), v.Raw[1], v.Raw[2])
		}
	case KindTime:
		if len(v.Raw) == 4 { // hour, minute, second, hundredths
			return fmt.Sprintf("%02d:%02d:%02d.%02d", v.Raw[0], v.Raw[1], v.Raw[2], v.Raw[3])
		}
	case KindOID:
		return v.Object.String()
	}
	return fmt.Sprintf("% X", v.Raw)
}
