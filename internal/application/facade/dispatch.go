package facade

import (
	"fmt"
	"strconv"
	"strings"
)

// Execute dispatches a raw input string to the appropriate command handler and
// appends any objective-reached or win banner triggered by the command.
func (a *App) Execute(input string) string {
	doneBefore, reachedBefore, wonBefore := a.session.ObjectiveIndex(), a.session.ReachedTarget(), a.session.Won()
	out := a.dispatch(input)
	return out + a.goalBanner(doneBefore, reachedBefore, wonBefore)
}

// dispatch routes a raw input string to the appropriate command handler. It is
// a flat switch over the fixed set of commands; funlen is silenced because
// splitting a command table into sub-dispatchers would obscure, not clarify, it.
//
//nolint:funlen // flat command-routing switch over the fixed command set
func (a *App) dispatch(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}

	parts := strings.Fields(trimmed)
	cmd := parts[0]
	args := ""
	if len(parts) > 1 {
		args = strings.Join(parts[1:], " ")
	}

	if len(parts) == 1 {
		if number, err := strconv.Atoi(cmd); err == nil {
			return a.ExecuteJourney(number)
		}
	}

	switch cmd {
	case "help":
		return a.Help()
	case "where":
		return a.Where()
	case "look":
		return a.Look()
	case "ls":
		return a.List()
	case "route":
		if args == "" {
			return "Usage: route <destination>"
		}
		return a.Route(args)
	case "travel":
		if args == "" {
			return "Usage: travel <destination>"
		}
		return a.Travel(args)
	case "home":
		return a.GoHome()
	case "cost":
		return a.Cost()
	case "shift":
		if args == "back" {
			return a.ShiftBack()
		}
		return a.Shift()
	case "jump":
		if args == "back" {
			return a.JumpBack()
		}
		return a.Jump()
	case "universe":
		if args == "back" {
			return a.UniverseBack()
		}
		return a.Universe()
	case "mathematical":
		if args == "back" {
			return a.MathematicalBack()
		}
		return a.Mathematical()
	case "simulate":
		if args == "back" {
			return a.SimulateBack()
		}
		return a.Simulate()
	case "drift":
		return a.Drift()
	case "align":
		return a.Align()
	case "observe":
		if args == "" {
			return "Usage: observe <observer>"
		}
		if args == "back" {
			return a.ObserveBack()
		}
		return a.Observe(args)
	case "time":
		if args == "" {
			return "Usage: time <RFC3339> or time back"
		}
		if args == "back" {
			return a.TimeBack()
		}
		return a.Time(args)
	case "save":
		if args != "" {
			return "Usage: save"
		}
		msg, err := a.Save()
		if err != nil {
			return err.Error()
		}
		return msg
	case "quest":
		if args != "" {
			return "Usage: quest"
		}
		return a.NewQuest()
	case "exit":
		return "Goodbye."
	default:
		if suggestion := a.suggestCommand(cmd); suggestion != "" {
			return fmt.Sprintf("Unknown command: %s\n\nDid you mean '%s'?\n\n%s", cmd, suggestion, a.Help())
		}
		return fmt.Sprintf("Unknown command: %s\n\n%s", cmd, a.Help())
	}
}
