package loopback

import "testing"

func TestCheckAcceptsOnlyAnAddressOfThisMachine(t *testing.T) {
	t.Parallel()
	for _, addr := range []string{"127.0.0.1:1502", "127.8.9.10:0", "[::1]:1502", "localhost:1502"} {
		if err := Check(addr); err != nil {
			t.Errorf("Check(%q) = %v, want nil", addr, err)
		}
	}
	for _, addr := range []string{"0.0.0.0:1502", ":1502", "[::]:1502", "192.0.2.10:1502", "device.example:1502", "127.0.0.1", ""} {
		if err := Check(addr); err == nil {
			t.Errorf("Check(%q) = nil, want a refusal", addr)
		}
	}
}
