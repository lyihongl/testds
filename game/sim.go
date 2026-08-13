package game

// SimulationStep advances the world by dt seconds, applies the frame's Intent,
// and returns the cosmetic events produced. It is the single contract between
// the game (sim) and ui packages (ADR 0003).
//
// Only the loop skeleton lives here so far — time, spawn toggle, restart, and
// empty-cell placement. The real systems (spawn, enemies, energy, machines,
// drones, turrets) land with the "Implement the game-loop systems" ticket.
func SimulationStep(gs *GameState, in Intent, dt float64) []Event {
	if in.Restart {
		gs.Reset()
	}
	if in.ToggleSpawn {
		gs.Spawn.On = !gs.Spawn.On
	}
	if in.Place {
		e, ok := gs.Reg.Get(in.Kind)
		if ok && gs.StructureAt(in.CellX, in.CellY) == nil {
			gs.SetStructure(in.CellX, in.CellY, &Structure{Kind: in.Kind, HP: e.Stats.HP})
		}
	}
	gs.Time += dt
	return nil
}

// Reset returns the world to its starting state, reading live tuning (core
// HP, starting elements, spawn grace). Costs and per-kind placement rules land
// with the game-loop systems.
func (gs *GameState) Reset() {
	t := gs.Tune
	gs.Chunks = make(map[ChunkPos]*Chunk)
	gs.Stockpile = Stockpile{El: int(t.StartEl)}
	gs.Enemies = nil
	gs.Drones = nil
	gs.Time = 0
	// Spawn off by default (G toggles it); grace before the first spawn.
	gs.Spawn = SpawnState{Timer: t.Grace, On: false}
	gs.Metrics = Metrics{}
	// Core at the prototype's center cell.
	if e, ok := gs.Reg.Get(KindCore); ok {
		gs.SetStructure(5, 5, &Structure{Kind: KindCore, HP: e.Stats.HP})
	}
}
