package box

import (
	"errors"
	"testing"
)

// Live (4.14.0.162 lab station): with Drivers/ObixNetwork disabled, GET /obix/about/ answers 410
// Gone. About keeps the code so n4_connect can name the cause; a 410 never latches.
func TestAbout410IsAStatusErrorWithItsCode(t *testing.T) {
	err, latched := aboutWith(t, 410)
	var be *Error
	if !errors.As(err, &be) || be.Kind != KindHTTPStatus || be.Code != 410 || be.IsAuth() {
		t.Fatalf("410: %#v", err)
	}
	if len(latched) != 0 {
		t.Errorf("a 410 on /obix/about/ latched: %v", latched[0].Msg)
	}
}
