package universe

import "fmt"

// BranchObserver creates a new observer perspective from the given
// location, including a coordinate-matched physical subgraph.
func BranchObserver(u *Aggregate, fromID string, fromCoord CoordinateVO, fromName, destID, observer string) error {
	destCoord := fromCoord
	destCoord.Observer = observer
	return BranchContextual(u, fromID, destCoord, fromName, destID,
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
