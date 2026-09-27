package version

import "testing"

func TestParseReadsAReleaseAndRefusesEverythingElse(t *testing.T) {
	for _, tc := range []struct {
		text string
		want Version
		ok   bool
	}{
		{text: "0.19.0", want: Version{0, 19, 0}, ok: true},
		{text: "v0.19.0", want: Version{0, 19, 0}, ok: true},
		{text: " 1.2.3 ", want: Version{1, 2, 3}, ok: true},
		{text: "0.19.0-dev (abc1234)", ok: false},
		{text: "0.19.0-dev", ok: false},
		{text: "dev", ok: false},
		{text: "0.19", ok: false},
		{text: "0.19.0.1", ok: false},
		{text: "0.19.x", ok: false},
		{text: "", ok: false},
		{text: "0.19.-1", ok: false},
	} {
		got, ok := Parse(tc.text)
		if ok != tc.ok || got != tc.want {
			t.Errorf("Parse(%q) = %v, %v, want %v, %v", tc.text, got, ok, tc.want, tc.ok)
		}
	}
}

func TestCompareOrdersByEachNumberInTurn(t *testing.T) {
	for _, tc := range []struct {
		a, b Version
		sign int
	}{
		{Version{0, 19, 0}, Version{0, 19, 0}, 0},
		{Version{0, 18, 0}, Version{0, 19, 0}, -1},
		{Version{0, 19, 1}, Version{0, 19, 0}, 1},
		{Version{1, 0, 0}, Version{0, 99, 99}, 1},
		{Version{0, 9, 0}, Version{0, 10, 0}, -1},
	} {
		got := Compare(tc.a, tc.b)

		switch {
		case tc.sign == 0 && got != 0, tc.sign < 0 && got >= 0, tc.sign > 0 && got <= 0:
			t.Errorf("Compare(%v, %v) = %d, want sign %d", tc.a, tc.b, got, tc.sign)
		}
	}
}

func TestStringWritesTheTagWithoutItsPrefix(t *testing.T) {
	if got := (Version{0, 19, 0}).String(); got != "0.19.0" {
		t.Errorf("String() = %q, want 0.19.0", got)
	}
}
