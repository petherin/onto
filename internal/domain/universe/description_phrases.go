package universe

// Phrasing pools for GenerateDescription. Each pool holds interchangeable
// variants; the "%s" placeholder (where present) is filled with the relevant
// setting or axis token. Pools are kept together here so the generation logic in
// description.go stays readable, and so new phrasings can be added without
// touching that logic.

// spatialOpeners anchor a description in its physical setting (%s = place).
var spatialOpeners = []string{
	"A quiet corner of %s that few think to visit.",
	"An unremarkable spot in %s, easy to walk past and just as easy to forget.",
	"A pocket of %s where the streets seem to fold in on themselves.",
	"Somewhere on the edge of %s, between the places that still have names.",
	"A stretch of %s the maps never quite agree on.",
	"A weathered fragment of %s, half-remembered even by the people who live there.",
}

// placelessOpeners cover coordinates with no usable spatial setting.
var placelessOpeners = []string{
	"A place with no fixed address, hanging loose from the usual geography.",
	"Nowhere in particular — a location that resists being pinned to a map.",
	"An unplaced spot, adrift from any familiar landmark.",
}

// closers give a plain base-reality node a second beat when no axis clauses apply.
var closers = []string{
	"Nothing here asks to be remembered.",
	"It waits, indifferent to whether you stay.",
	"You could linger or move on; the place would not notice either way.",
	"There is a stillness to it, as if it were between uses.",
}

// mathematicsClauses describe a non-Classical mathematical structure (%s = M-level).
var mathematicsClauses = []string{
	"The very logic of the place answers to structure %s rather than the Classical frame.",
	"Geometry here obeys mathematical structure %s; straight lines are a local courtesy.",
	"Beneath everything runs formal structure %s, where even arithmetic feels negotiable.",
}

// universeClauses describe a parallel bubble universe (%s = U-level).
var universeClauses = []string{
	"The physical constants sit slightly off true — this is bubble universe %s.",
	"Light and weight behave just differently enough to notice, this far into universe %s.",
	"You are in bubble universe %s, where the fundamentals were dialled to other values.",
}

// timelineClauses describe an alternate timeline (%s = timeline token).
var timelineClauses = []string{
	"History took a different turning to reach timeline %s; the details are subtly wrong.",
	"This is timeline %s, where some old decision went the other way.",
	"An alternate history clings to the place — the branch known as %s.",
}

// quantumClauses describe a quantum branch (%s = quantum token).
var quantumClauses = []string{
	"The air carries a faint doubling, as though branch %s had almost not happened.",
	"Reality feels provisional here, one outcome among many in quantum branch %s.",
	"Objects hold a slight after-image, the residue of quantum branch %s.",
}

// simulationClauses describe a nested simulation (%s = ordinal depth).
var simulationClauses = []string{
	"Everything here is computed — the %s simulation layer down — and the seams occasionally show.",
	"You are inside the %s nested simulation; its rules can be rewritten from outside.",
	"A faint lattice underlies the %s simulated layer, as if the world were still rendering.",
}

// consensusClauses describe divergence from shared consensus (%s = ordinal depth).
var consensusClauses = []string{
	"Shared reality has thinned to the %s divergence; what you see may be yours alone.",
	"At the %s step from consensus, the place drifts toward dream, or something past it.",
	"Consensus is %s removed here — agreement about what is real no longer holds.",
}

// observerClauses describe a non-default perceptual frame (%s = observer).
var observerClauses = []string{
	"Perceived through %s, familiar shapes resolve into something stranger.",
	"Seen as %s would see it, the ordinary becomes unrecognisable.",
	"Filtered through the umwelt of %s, the place answers to different senses.",
}

// timeClauses pin the coordinate to a specific instant (%s = formatted time).
var timeClauses = []string{
	"The moment is pinned to %s, and it does not move on.",
	"Time here is fixed at %s, held like an insect in amber.",
	"You stand at %s exactly — the instant refuses to advance.",
}
