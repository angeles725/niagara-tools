package signingprofile

import (
	"errors"
	"strings"
)

// The keys of a profile that name its keystore. The storepass is IN the file: a tool may hand it to the
// keystore reader, never print, log or echo it.
const (
	StorePathKey = "niagara.signing.storepath"
	StoreTypeKey = "niagara.signing.storetype"
	StorePassKey = "niagara.signing.storepass"
	// DefaultStoreType is the keystore type of a profile that names none (the plugin's default).
	DefaultStoreType = "JCEKS"
)

// ErrNotXML is Entries' error for a file that is not properties XML. It never quotes the file: the XML
// parser's own error names the text around the fault, which can be the store password.
var ErrNotXML = errors.New("not valid properties XML")

// Entries is the key/value map of a java.util.Properties XML profile, values trimmed.
func Entries(b []byte) (map[string]string, error) {
	m, err := entries(b)
	if err != nil {
		return nil, ErrNotXML
	}
	for k, v := range m {
		m[k] = strings.TrimSpace(v)
	}
	return m, nil
}
