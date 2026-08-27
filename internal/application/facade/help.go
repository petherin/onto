package facade

import "strings"

// Help returns the usage text listing all available commands.
func (a *App) Help() string {
	return strings.Join([]string{
		"Usage",
		"",
		"where                  Show your current reality coordinate",
		"look                   Describe your current location",
		"ls                     List nearby connected locations",
		"route <destination>    Plan a route to a known place",
		"travel <destination>   Move to a known place",
		"home                   Return home (jumps back timelines, shifts back quantum, then travels)",
		"cost                   Show travel cost information",
		"shift                  Jump forward to the next quantum branch",
		"shift back             Return to the previous quantum branch",
		"jump                   Jump forward to the next timeline branch",
		"jump back              Return to the previous timeline branch",
		"universe               Shift forward to the next bubble universe",
		"universe back          Return to the previous bubble universe",
		"structure              Shift forward to the next mathematical structure",
		"structure back         Return to the previous mathematical structure",
		"simulate               Enter the next nested simulation layer",
		"simulate back          Exit one simulation layer toward base reality",
		"drift                  Enter the next consensus divergence",
		"align                  Return one level toward shared consensus",
		"observe <observer>     Change observer perspective",
		"observe back           Return to the previous observer perspective",
		"time <RFC3339>         Enter a temporal branch",
		"time back              Return to the previous temporal branch",
		"save                   Persist the current universe graph to disk",
		"quest                  Roll a new random quest from the objective pool",
		"<number>               Take a numbered possible journey",
		"exit                   Leave the app",
		"",
		"Objective:",
		"When a game is in force, you start with a finite budget that every move",
		"spends against (see the per-command costs above and in 'ls'). A move you",
		"cannot afford is refused and costs nothing. Your goal is a quest chain of",
		"round trips: reach each target coordinate in order and return home after each;",
		"the last return home wins. 'where' shows your",
		"remaining budget, the objective checklist, the par (optimal cost), and, once",
		"you win, a 1-3 star rating for how close to par you played. Returning home is",
		"always allowed even if it overspends the budget.",
		"",
		"Example destinations:",
		"home, station, park, city-centre",
		"",
		"Example commands:",
		"route station",
		"travel station",
		"route park",
	}, "\n")
}

// CommandInfo holds the name and one-line description of a command.
type CommandInfo struct {
	Name        string
	Description string
}

// Commands returns the full list of commands supported by the App.
func (a *App) Commands() []CommandInfo {
	return []CommandInfo{
		{Name: "where", Description: "Show the current reality coordinate"},
		{Name: "look", Description: "Inspect the current location"},
		{Name: "ls", Description: "List connected locations"},
		{Name: "route", Description: "Plan a route to a destination"},
		{Name: "travel", Description: "Travel to a destination"},
		{Name: "home", Description: "Return home, unwinding quantum and timeline branches as needed"},
		{Name: "cost", Description: "Show travel cost information"},
		{Name: "shift", Description: "Jump to the nearest quantum branch of your current location"},
		{Name: "jump", Description: "Jump to the next timeline branch of your current location"},
		{Name: "universe", Description: "Shift to the next bubble universe of your current location"},
		{Name: "structure", Description: "Shift to the next mathematical structure of your current location"},
		{Name: "simulate", Description: "Enter the next nested simulation layer"},
		{Name: "drift", Description: "Enter the next consensus divergence"},
		{Name: "align", Description: "Return one level toward shared consensus"},
		{Name: "observe", Description: "Change observer perspective"},
		{Name: "time", Description: "Enter a temporal branch at an RFC3339 timestamp"},
		{Name: "save", Description: "Persist the current universe graph to disk"},
		{Name: "quest", Description: "Roll a new random quest from the objective pool"},
		{Name: "exit", Description: "Exit the app"},
	}
}
