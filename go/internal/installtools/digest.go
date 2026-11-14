package installtools

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
)

// SHA256 is the hex sha256 of the file's content, as `sha256sum <path>` prints it.
func SHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// SameContent reports whether the two files hold the same bytes (`cmp -s a b`). A file that cannot be
// read is an error, never "same".
func SameContent(a, b string) (bool, error) {
	x, err := os.ReadFile(a)
	if err != nil {
		return false, err
	}
	y, err := os.ReadFile(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(x, y), nil
}
