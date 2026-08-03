package cli

import "os"

const (
	// AppVersion is the current version string shown at startup.
	AppVersion = "Onto Explorer v0.1"

	// QuantumShiftCost is the cost of a single quantum branch jump.
	QuantumShiftCost = 20.0

	defaultDataFile      = "data/locations.json"
	defaultStartLocation = "home"
)

// dataFile returns the path to the locations JSON file.
// Override with the ONTO_DATA_FILE environment variable.
func dataFile() string {
	if v := os.Getenv("ONTO_DATA_FILE"); v != "" {
		return v
	}
	return defaultDataFile
}

// startLocation returns the ID of the location the app starts at.
// Override with the ONTO_START_LOCATION environment variable.
func startLocation() string {
	if v := os.Getenv("ONTO_START_LOCATION"); v != "" {
		return v
	}
	return defaultStartLocation
}
