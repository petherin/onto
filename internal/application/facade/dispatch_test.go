package facade

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDispatch_UsageMessages covers the arg-validation branches of dispatch:
// commands that require an argument report their usage when given none, and the
// two no-arg commands report usage when handed a stray argument.
func TestDispatch_UsageMessages(t *testing.T) {
	app := newGameApp(t)
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"route without destination", "route", "Usage: route <destination>"},
		{"travel without destination", "travel", "Usage: travel <destination>"},
		{"observe without observer", "observe", "Usage: observe <observer>"},
		{"time without argument", "time", "Usage: time <RFC3339> or time back"},
		{"save with stray argument", "save now", "Usage: save"},
		{"quest with stray argument", "quest now", "Usage: quest"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, app.Execute(tt.input))
		})
	}
}

// TestDispatch_EmptyInputIsNoop confirms blank input dispatches to nothing.
func TestDispatch_EmptyInputIsNoop(t *testing.T) {
	app := newGameApp(t)
	assert.Equal(t, "", app.Execute("   "))
}

// TestDispatch_Exit returns the goodbye sentinel.
func TestDispatch_Exit(t *testing.T) {
	app := newGameApp(t)
	assert.Equal(t, "Goodbye.", app.Execute("exit"))
}

// TestDispatch_UnknownCommandSuggests covers the did-you-mean branch: an input
// close to a real command reports it as unknown and suggests the nearest match,
// followed by the help text.
func TestDispatch_UnknownCommandSuggests(t *testing.T) {
	app := newGameApp(t)
	out := app.Execute("wher")
	assert.Contains(t, out, "Unknown command: wher")
	assert.Contains(t, out, "Did you mean 'where'?")
	assert.Contains(t, out, app.Help())
}

// TestDispatch_UnknownCommandNoSuggestion covers the fallback branch: an input
// far from every command reports it as unknown with no suggestion, followed by
// the help text.
func TestDispatch_UnknownCommandNoSuggestion(t *testing.T) {
	app := newGameApp(t)
	out := app.Execute("zzzzzzzz")
	assert.Contains(t, out, "Unknown command: zzzzzzzz")
	assert.NotContains(t, out, "Did you mean")
	assert.Contains(t, out, app.Help())
}
