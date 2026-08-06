package cli

import "os"

const (
	// AppVersion is the current version string shown at startup.
	AppVersion = "Onto Explorer v0.1"

	defaultDataFile      = "data/locations.json"
	defaultStartLocation = "home"

	msgGoodbye            = "Goodbye."
	msgAlreadyHome        = "You are already home."
	fmtUnknownDestSuggest = "Unknown destination: %s\n\nDid you mean '%s'?"
	fmtSaveWarning        = "\n\nWarning: failed to save config: %v"

	cmdHelp    = "help"
	cmdWhere   = "where"
	cmdLook    = "look"
	cmdList    = "ls"
	cmdRoute   = "route"
	cmdTravel  = "travel"
	cmdHome    = "home"
	cmdCost    = "cost"
	cmdShift   = "shift"
	cmdJump    = "jump"
	cmdDrift   = "drift"
	cmdAlign   = "align"
	cmdObserve = "observe"
	cmdExit    = "exit"
	argBack    = "back"
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
