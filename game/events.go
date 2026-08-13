package game

// Vec is a continuous cell-space coordinate (enemies, drones, events).
type Vec struct {
	X float64
	Y float64
}

// EventKind discriminates cosmetic events emitted by the sim.
type EventKind int

const (
	EventTracer EventKind = iota
	EventWarning
)

// Event is one cosmetic effect produced during a sim step. Per ADR 0002
// cosmetics are events, never state: the sim emits them, the ui fx layer
// animates them, and they are never serialized.
type Event struct {
	Kind EventKind
	A    Vec    // tracer start / origin
	B    Vec    // tracer end
	Text string // warning text
	Dur  float64
}
