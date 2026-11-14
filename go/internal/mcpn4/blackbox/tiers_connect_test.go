package blackbox

// The version tier reported at connect (port of test_tiers.TestConnectReportsTier; METHODOLOGY
// section 5).

import (
	"strings"
	"testing"
)

func TestTierAStation(t *testing.T) {
	r := NewRig(t, nil)
	out := r.Connect(nil)
	if out["version"] != "4.14.0.162" || out["tier"] != "A" || out["tier_writes"] != "allowed" {
		t.Errorf("connect = %v", out)
	}
	desc := r.Srv.OK("n4_describe_session", nil)
	if desc["version"] != "4.14.0.162" || desc["tier"] != "A" {
		t.Errorf("describe = %v", desc)
	}
}

func TestUnknownVersionIsTierC(t *testing.T) {
	r := NewRig(t, nil)
	r.Station.SetVersion("")
	out := r.Connect(nil)
	if v, has := out["version"]; !has || v != nil || out["version_source"] != nil || out["tier"] != "C" || out["tier_writes"] != "refused" {
		t.Errorf("connect = %v", out)
	}
}

func TestADetectedVersionCarriesNoVersionError(t *testing.T) {
	out := NewRig(t, nil).Connect(nil)
	if _, has := out["version_error"]; has || out["version"] == nil {
		t.Errorf("connect = %v", out)
	}
}

func TestAMissingAboutEndpointIsReported(t *testing.T) {
	r := NewRig(t, nil)
	r.Station.SetVersion("")
	if got, _ := r.Connect(nil)["version_error"].(string); !strings.Contains(got, "HTTP 404") {
		t.Errorf("version_error %q", got)
	}
}

func TestTierBStationReportsRefusedWithoutOptIn(t *testing.T) {
	r := NewRig(t, nil)
	r.Station.SetVersion("4.15.3.28")
	if out := r.Connect(nil); out["tier"] != "B" || out["tier_writes"] != "refused" {
		t.Errorf("connect = %v", out)
	}
}

func TestDescribeWithoutSessionHasNoTier(t *testing.T) {
	desc := NewRig(t, nil).Srv.OK("n4_describe_session", nil)
	tier, hasTier := desc["tier"]
	version, hasVersion := desc["version"]
	if !hasTier || tier != nil || !hasVersion || version != nil {
		t.Errorf("describe = %v", desc)
	}
}
