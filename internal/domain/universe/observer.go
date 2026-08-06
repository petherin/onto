package universe

import "fmt"

// BranchObserverService creates a new observer perspective from the given
// location, including a coordinate-matched physical subgraph.
func BranchObserverService(u *Aggregate, fromID string, fromCoord CoordinateVO, fromName, destID, observer string) {
	destCoord := fromCoord
	destCoord.Observer = observer
	BranchContextualService(u, fromID, destCoord, fromName, destID,
		fmt.Sprintf("The same reality as perceived through %s.", observer),
		ContextualTransitionSpec{
			Mode:               ObserverShift,
			Cost:               ObserverShiftCost,
			Label:              fmt.Sprintf("observer %s", observer),
			ForwardDescription: fmt.Sprintf("Observer shift to %s", observer),
			ReverseDescription: fmt.Sprintf("Observer shift back to %s", fromCoord.Observer),
		},
	)
}
