package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrompt_AtHome(t *testing.T) {
	app := NewApp()
	// Default start: all standard defaults — ShortAddress omits Planet/Country/Region.
	assert.Equal(t, "[Leeds/Home] > ", app.Prompt())
}

func TestPrompt_AfterTravelToStation(t *testing.T) {
	app := NewApp()
	app.Execute("travel station")

	assert.Equal(t, "[Leeds/Station] > ", app.Prompt())
}

func TestPrompt_AfterQuantumShift(t *testing.T) {
	app := NewApp()
	app.Execute("shift")

	// Q1 is non-default; appears before spatial segments.
	assert.Equal(t, "[Q1/Leeds/Home] > ", app.Prompt())
}

func TestPrompt_AfterTimelineJump(t *testing.T) {
	app := NewApp()
	app.Execute("jump")

	// T1 is non-default; appears before spatial segments.
	assert.Equal(t, "[T1/Leeds/Home] > ", app.Prompt())
}

func TestPrompt_AfterQuantumShiftAndTimelineJump(t *testing.T) {
	app := NewApp()
	app.Execute("shift")
	app.Execute("jump")

	// Timeline jump resets quantum to Q0; only T1 appears.
	assert.Equal(t, "[T1/Leeds/Home] > ", app.Prompt())
}

func TestPrompt_AfterTravelThenShiftThenJump(t *testing.T) {
	app := NewApp()
	app.Execute("travel station")
	app.Execute("shift")
	app.Execute("jump")

	// Timeline jump resets quantum — only T1.
	assert.Equal(t, "[T1/Leeds/Station] > ", app.Prompt())
}

func TestPrompt_ShiftBackResetsQuantum(t *testing.T) {
	app := NewApp()
	app.Execute("shift")
	app.Execute("shift back")

	// Back at Q0 — no exotic axes in prompt.
	assert.Equal(t, "[Leeds/Home] > ", app.Prompt())
}

func TestPrompt_JumpBackResetsTimeline(t *testing.T) {
	app := NewApp()
	app.Execute("jump")
	app.Execute("jump back")

	// Back at Prime — no exotic axes in prompt.
	assert.Equal(t, "[Leeds/Home] > ", app.Prompt())
}

func TestPrompt_WhereShowsFullCoordinateAfterShiftBack(t *testing.T) {
	app := NewApp()
	app.Execute("travel station")
	app.Execute("shift")
	app.Execute("shift back")

	// After shift back to station, where must show all fields populated.
	output := app.Execute("where")
	assert.Contains(t, output, "Earth")
	assert.Contains(t, output, "United Kingdom")
	assert.Contains(t, output, "Leeds")
	assert.Contains(t, output, "Station")
}

func TestPrompt_WhereShowsFullCoordinateAfterJumpBack(t *testing.T) {
	app := NewApp()
	app.Execute("travel station")
	app.Execute("jump")
	app.Execute("jump back")

	output := app.Execute("where")
	assert.Contains(t, output, "Earth")
	assert.Contains(t, output, "United Kingdom")
	assert.Contains(t, output, "Leeds")
	assert.Contains(t, output, "Station")
}
