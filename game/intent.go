package game

// Intent is everything the sim must act on for one frame (ADR 0003: game owns
// the contract; ui translates input into it). The zero value means "do
// nothing" — the sim must tolerate it every frame.
type Intent struct {
	// Place requests placing the selected Kind at Cell (validity is the
	// sim's call — costs land with the config registry).
	Place bool
	Kind  string
	CellX int64
	CellY int64

	// ToggleSpawn flips the enemy spawn gate.
	ToggleSpawn bool

	// Restart returns the world to its starting state.
	Restart bool
}
