package cli

// Command holds the name and one-line description of a CLI command, used for
// fuzzy matching and help text generation.
type Command struct {
	Name        string
	Description string
}

// Commands returns the full list of commands supported by the App.
func (a *App) Commands() []Command {
	return []Command{
		{Name: cmdWhere, Description: "Show the current reality coordinate"},
		{Name: cmdLook, Description: "Inspect the current location"},
		{Name: cmdList, Description: "List connected locations"},
		{Name: cmdRoute, Description: "Plan a route to a destination"},
		{Name: cmdTravel, Description: "Travel to a destination"},
		{Name: cmdHome, Description: "Return home, unwinding quantum and timeline branches as needed"},
		{Name: cmdCost, Description: "Show travel cost information"},
		{Name: cmdShift, Description: "Jump to the nearest quantum branch of your current location"},
		{Name: cmdJump, Description: "Jump to the next timeline branch of your current location"},
		{Name: cmdUniverse, Description: "Shift to the next bubble universe of your current location"},
		{Name: cmdStructure, Description: "Shift to the next mathematical structure of your current location"},
		{Name: cmdDrift, Description: "Enter the next consensus divergence"},
		{Name: cmdAlign, Description: "Return one level toward shared consensus"},
		{Name: cmdObserve, Description: "Change observer perspective"},
		{Name: cmdTime, Description: "Enter a temporal branch at an RFC3339 timestamp"},
		{Name: cmdSave, Description: "Persist the current universe graph to disk"},
		{Name: cmdExit, Description: "Exit the CLI"},
	}
}
