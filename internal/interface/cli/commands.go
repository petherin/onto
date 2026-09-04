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
		{Name: cmdHome, Description: "Return home, unwinding quantum branches and Hubble-volume jumps as needed"},
		{Name: cmdCost, Description: "Show travel cost information"},
		{Name: cmdShift, Description: "Jump to the next quantum branch of your current location"},
		{Name: cmdJump, Description: "Jump drive: thread a wormhole to a distant Hubble volume sharing your location's geography but not its history"},
		{Name: cmdUniverse, Description: "Shift to the next bubble universe of your current location"},
		{Name: cmdMathematical, Description: "Shift to the next mathematical structure of your current location"},
		{Name: cmdSimulate, Description: "Enter the next nested simulation layer"},
		{Name: cmdDrift, Description: "Enter the next consensus divergence"},
		{Name: cmdAlign, Description: "Return one level toward shared consensus"},
		{Name: cmdObserve, Description: "Change observer perspective"},
		{Name: cmdTime, Description: "Enter a temporal branch at an RFC3339 timestamp"},
		{Name: cmdSave, Description: "Persist the current universe graph to disk"},
		{Name: cmdExit, Description: "Exit the CLI"},
	}
}
