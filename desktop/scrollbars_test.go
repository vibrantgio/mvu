package desktop

import "testing"

// TestScrollBarsAnswersAKnownSetting asserts the bridge answers one of the
// platform's three settings on whatever platform the tests run on. The value
// itself is the machine's own and cannot be asserted.
func TestScrollBarsAnswersAKnownSetting(t *testing.T) {
	switch got := ScrollBars(); got {
	case ScrollBarsAutomatic, ScrollBarsWhenScrolling, ScrollBarsAlways:
	default:
		t.Fatalf("ScrollBars() = %d, want one of the three settings", int(got))
	}
}

// TestScrollBarSettingNames pins the names to the platform's own spelling of
// them, which is what the darwin reader matches the preference against.
func TestScrollBarSettingNames(t *testing.T) {
	for _, c := range []struct {
		setting ScrollBarSetting
		want    string
	}{
		{ScrollBarsAutomatic, "Automatic"},
		{ScrollBarsWhenScrolling, "WhenScrolling"},
		{ScrollBarsAlways, "Always"},
	} {
		if got := c.setting.String(); got != c.want {
			t.Errorf("%d.String() = %q, want %q", int(c.setting), got, c.want)
		}
	}
}
