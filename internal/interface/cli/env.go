package cli

import "github.com/petherin/onto/internal/application/facade"

// AppVersion re-exports the application version for callers that already
// import the cli package (e.g. existing tests).
const AppVersion = facade.AppVersion

const (
	msgGoodbye            = "Goodbye."
	msgSaved              = "Saved."
	fmtUnknownDestSuggest = "Unknown destination: %s\n\nDid you mean '%s'?"
	fmtExitSaveWarning    = "Warning: failed to save before exit: %v"

	cmdHelp      = "help"
	cmdWhere     = "where"
	cmdLook      = "look"
	cmdList      = "ls"
	cmdRoute     = "route"
	cmdTravel    = "travel"
	cmdHome      = "home"
	cmdCost      = "cost"
	cmdShift     = "shift"
	cmdJump      = "jump"
	cmdUniverse  = "universe"
	cmdStructure = "structure"
	cmdSimulate  = "simulate"
	cmdDrift     = "drift"
	cmdAlign     = "align"
	cmdObserve   = "observe"
	cmdTime      = "time"
	cmdSave      = "save"
	cmdExit      = "exit"
	argBack      = "back"
)
