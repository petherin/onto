package cli

type Command struct {
	Name        string
	Description string
}

func (a *App) Commands() []Command {
	return []Command{
		{Name: cmdWhere, Description: "Show the current reality coordinate"},
		{Name: cmdLook, Description: "Inspect the current location"},
		{Name: cmdList, Description: "List connected locations"},
		{Name: cmdRoute, Description: "Plan a route to a destination"},
		{Name: cmdTravel, Description: "Travel to a destination"},
		{Name: cmdCost, Description: "Show travel cost information"},
		{Name: cmdShift, Description: "Jump to the nearest quantum branch of your current location"},
		{Name: cmdExit, Description: "Exit the CLI"},
	}
}
