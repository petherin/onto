package cli

import (
	"strings"
	"testing"
)

func TestAppWhere(t *testing.T) {
	app := NewApp()
	output := app.Execute("where")

	if !strings.Contains(output, "Reality Coordinate") {
		t.Fatalf("expected coordinate header, got %q", output)
	}

	if !strings.Contains(output, "Earth") {
		t.Fatalf("expected Earth in output, got %q", output)
	}
}

func TestAppRouteToStation(t *testing.T) {
	app := NewApp()
	output := app.Execute("route station")

	if !strings.Contains(output, "Route") {
		t.Fatalf("expected route output, got %q", output)
	}

	if !strings.Contains(output, "Station") {
		t.Fatalf("expected station in route output, got %q", output)
	}
}

func TestAppTravelToStation(t *testing.T) {
	app := NewApp()
	output := app.Execute("travel station")

	if !strings.Contains(output, "Arrived") {
		t.Fatalf("expected arrival message, got %q", output)
	}
}

func TestAppTravelShowsPossibleJourneys(t *testing.T) {
	app := NewApp()
	output := app.Execute("travel station")

	if !strings.Contains(output, "Possible journeys") {
		t.Fatalf("expected nearby journey suggestions, got %q", output)
	}
}

func TestAppSuggestsSimilarDestination(t *testing.T) {
	app := NewApp()
	output := app.Execute("route parl")

	if !strings.Contains(output, "Did you mean") {
		t.Fatalf("expected destination suggestion, got %q", output)
	}

	if !strings.Contains(output, "park") {
		t.Fatalf("expected park suggestion, got %q", output)
	}
}
