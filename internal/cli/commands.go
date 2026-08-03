package cli

type Command struct {
	Name        string
	Description string
}

func (a *App) Commands() []Command {
	return []Command{
		{Name: "where", Description: "Show the current reality coordinate"},
		{Name: "look", Description: "Inspect the current location"},
		{Name: "ls", Description: "List connected locations"},
		{Name: "route", Description: "Plan a route to a destination"},
		{Name: "travel", Description: "Travel to a destination"},
		{Name: "cost", Description: "Show travel cost information"},
		{Name: "shift", Description: "Jump to the nearest quantum branch of your current location"},
		{Name: "exit", Description: "Exit the CLI"},
	}
}
