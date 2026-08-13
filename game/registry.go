package game

// Entry is one structure kind's registration: its stats plus its per-tick
// behavior hook (the type seam, ADR 0004). Tick is filled by the game-loop
// systems ticket; nil means the kind has no per-tick behavior (core, wall).
type Entry struct {
	Kind  string
	Stats *Stats
	Tick  func(gs *GameState, s *Structure, st *Stats, dt float64)
}

// Registry is the runtime table of structure kinds. Adding a new kind is one
// entry in the tuning table (data) plus one registered Tick (behavior) — the
// loop never learns new kinds.
type Registry struct {
	Entries map[string]Entry
}

// NewRegistry builds the runtime registry from a Tuning's stats table.
// Entries share the same *Stats the tuning holds, so live tuning (the debug
// sliders) is visible to the sim immediately — single source, no copies.
func NewRegistry(t *Tuning) *Registry {
	r := &Registry{Entries: make(map[string]Entry, len(t.Kinds))}
	for kind, st := range t.Kinds {
		r.Entries[kind] = Entry{Kind: kind, Stats: st}
	}
	return r
}

// Get returns the entry for a structure kind.
func (r *Registry) Get(kind string) (Entry, bool) {
	e, ok := r.Entries[kind]
	return e, ok
}
