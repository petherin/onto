package navigation

type Cost struct {
	Base              float64
	TerrainMultiplier float64
}

func (c Cost) Total() float64 {
	if c.TerrainMultiplier <= 0 {
		return c.Base
	}
	return c.Base * c.TerrainMultiplier
}
