package verifymodule

import "testing"

// TestMakeNumericHasUnit pins mn_has_unit on the VMN1-VMN5 shapes: a unit argument, a named unit
// constant, and the precision-only one-argument overload (a literal or a named precision).
func TestMakeNumericHasUnit(t *testing.T) {
	for s, want := range map[string]bool{
		`BFacets.makeNumeric(BUnit.getUnit("psi"), 1)`: true,
		`BFacets.makeNumeric(AMPS, PRECISION)`:         true,
		`BFacets.makeNumeric(PSI_UNIT)`:                true,
		`BFacets.makeNumeric(1)`:                       false,
		`BFacets.makeNumeric(PRECISION)`:               false,
		`BFacets.makeNumeric( 0 )`:                     false,
		`BFacets.make(BFacets.UNITS, u)`:               false,
	} {
		if got := makeNumericHasUnit(s); got != want {
			t.Errorf("makeNumericHasUnit(%s) = %v, want %v", s, got, want)
		}
	}
}
