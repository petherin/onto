package facade

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNeedsHomeConfirm asserts the helper treats only an actionable route plan
// (one terminated by HomeConfirmPrompt) as needing confirmation, and never the
// terminal "already home" / "no route home" messages that GoHome can also
// return. All three delivery layers gate their confirm flow on this.
func TestNeedsHomeConfirm(t *testing.T) {
	tests := []struct {
		name string
		plan string
		want bool
	}{
		{
			name: "actionable plan",
			plan: "Route home\nHome -> Station\n\nEstimated cost: 2\n\n" + HomeConfirmPrompt,
			want: true,
		},
		{name: "bare prompt", plan: HomeConfirmPrompt, want: true},
		{name: "already home", plan: MsgAlreadyHome, want: false},
		{
			name: "no route home",
			plan: "No route home from Park. There is no path back to home from here.",
			want: false,
		},
		{name: "empty", plan: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NeedsHomeConfirm(tt.plan))
		})
	}
}
