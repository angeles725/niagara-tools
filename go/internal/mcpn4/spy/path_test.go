package spy

import (
	"strings"
	"testing"
)

func TestCleanPathKeepsPlainNamesBelowTheRoot(t *testing.T) {
	for in, want := range map[string]string{"": "", "/": "", "metrics": "metrics", "/metrics/": "metrics",
		"util/JMX Info": "util/JMX Info", "sysManagers/engineManager": "sysManagers/engineManager"} {
		if got, err := CleanPath(in); err != nil || got != want {
			t.Errorf("CleanPath(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestCleanPathRefusesWhatCouldLeaveTheRootOrNameAnAction(t *testing.T) {
	for in, why := range map[string]string{
		"spy:/metrics": "scheme", "https://other.invalid/spy": "scheme", "//other.invalid/x": "empty segment",
		"a//b": "empty segment", "../metrics": "'..'", "util/../../x": "'..'", "./metrics": "'.'",
		"metrics|x": "ORD chain", "logSetup?level=FINE": "query", "a&b": "query", "a=b": "query",
		"a#b": "query", "util/%2e%2e": "percent", "a;jsessionid": "path parameter", `a\b`: "backslash",
		"a\nb": "control", "a\x00": "control", "util/gc": "action", "/Util/GC/": "action",
		"util/forceThreadDump": "action", " metrics": "space", strings.Repeat("a", MaxPathLen+1): "longer"} {
		got, err := CleanPath(in)
		if err == nil || got != "" || !strings.Contains(err.Error(), why) {
			t.Errorf("CleanPath(%.40q) = %q, %v; want a refusal naming %q", in, got, err, why)
		}
	}
}

func TestURLPathEscapesEverySegmentUnderTheRoot(t *testing.T) {
	for in, want := range map[string]string{"": "/ord?spy:/", "metrics": "/ord?spy:/metrics",
		"util/JMX Info": "/ord?spy:/util/JMX%20Info", "a+b,c": "/ord?spy:/a%2Bb%2Cc"} {
		if got := URLPath(in); got != want || !IsURLPath(got) {
			t.Errorf("URLPath(%q) = %q (IsURLPath %v), want %q", in, got, IsURLPath(got), want)
		}
	}
}

func TestIsURLPathAcceptsOnlyWhatURLPathBuilds(t *testing.T) {
	for _, bad := range []string{"/ord?spy:/util/gc", "/ord?spy:/a|b", "/ord?spy:/..", "/ord?spy:/a%7Cb",
		"/ord?spy:/metrics?x=1", "/ord?slot:/", "/file/x", "https://other.invalid/ord?spy:/", "/ord?spy:/a b"} {
		if IsURLPath(bad) {
			t.Errorf("IsURLPath(%q) = true", bad)
		}
	}
}
