package bogaudit

import "math"

// Recorded from python3 3.15 (repr, str(float), float()).
var reprCases = []struct{ in, want string }{
	{"", "''"},
	{"abc", "'abc'"},
	{"it's", "\"it's\""},
	{"say \"hi\"", "'say \"hi\"'"},
	{"both '\"", "'both \\'\"'"},
	{"back\\slash", "'back\\\\slash'"},
	{"t\tn\nr\r", "'t\\tn\\nr\\r'"},
	{"\u0000", "'\\x00\\x7f\\x85'"},
	{"°C é", "'°C é'"},
	{"​ ", "'\\u200b\\u2028'"},
	{"😀", "'😀'"},
	{"󠀁", "'\\U000e0001'"},
	{"�", "'�'"},
	{"  x", "'\\xa0 x'"},
}

var floatReprCases = []struct {
	in   float64
	want string
}{
	{0.0, "0.0"},
	{math.Copysign(0, -1), "-0.0"},
	{1.0, "1.0"},
	{1.5, "1.5"},
	{30000.0, "30000.0"},
	{1e-05, "1e-05"},
	{0.0001, "0.0001"},
	{0.00012, "0.00012"},
	{1e+16, "1e+16"},
	{1.2345678901234567e+19, "1.2345678901234567e+19"},
	{1000000000000000.0, "1000000000000000.0"},
	{123456789012345.6, "123456789012345.6"},
	{-2.5e-07, "-2.5e-07"},
	{math.Inf(1), "inf"},
	{math.Inf(-1), "-inf"},
	{math.NaN(), "nan"},
	{5e-324, "5e-324"},
	{1.7976931348623157e+308, "1.7976931348623157e+308"},
	{100.0, "100.0"},
	{0.1, "0.1"},
}

var floatParseCases = []struct {
	in   string
	ok   bool
	want string
}{
	{" -1e-3 ", true, "-0.001"},
	{"1_000", true, "1000.0"},
	{"-inf", true, "-inf"},
	{"0x10", false, ""},
	{"1.2.3", false, ""},
	{"", false, ""},
	{"nan", true, "nan"},
	{"+Infinity", true, "inf"},
	{"1.", true, "1.0"},
	{"1.e5", true, "100000.0"},
	{".5", true, "0.5"},
	{"_1", false, ""},
	{"1_", false, ""},
	{"1__0", false, ""},
	{"1e", false, ""},
	{"1e+_5", false, ""},
	{"  7  ", true, "7.0"},
	{"1e999", true, "inf"},
	{"-1e999", true, "-inf"},
	{"٣", true, "3.0"},
	{"infinit", false, ""},
}

var _ = math.Inf
