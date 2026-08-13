package game

// SimulationStep advances the world by dt seconds, applies the frame's Intent,
// and returns the cosmetic events produced. It is the single contract between
// the game (sim) and ui packages (ADR 0003).
//
// Frame order (prototype parity): placement, spawn, enemies, then one
// AllStructures snapshot (grill Q7 — taken after enemies so the later systems
// see the post-chew world), then energy, machines, drones, turrets. Game over
// freezes the sim (time included); only Restart and ToggleSpawn still act.
func SimulationStep(gs *GameState, in Intent, dt float64) []Event {
	if in.Restart {
		gs.Reset()
		return nil
	}
	if in.ToggleSpawn {
		gs.Spawn.On = !gs.Spawn.On
	}
	if gs.GameOver() {
		return nil
	}
	var ev []Event
	if in.Place {
		ev = append(ev, placeStructure(gs, in.Kind, in.CellX, in.CellY)...)
	}
	stepSpawn(gs, dt)
	stepEnemies(gs, dt)
	if gs.GameOver() {
		gs.Time += dt
		return ev
	}
	placed := gs.AllStructures()
	stepEnergy(gs, placed, dt)
	stepMachines(gs, placed, dt)
	stepDrones(gs, dt)
	ev = append(ev, stepTurrets(gs, placed, dt)...)
	gs.Time += dt
	return ev
}

// Reset returns the world to its starting state, reading live tuning (core
// HP, starting elements, spawn grace). The demo world is one chunk (grill
// Q1): the core seeds at the chunk center and its chunk is the starting
// de-fogged territory (grill Q3).
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
	cc := int64(CHUNK_SIZE / 2)
	core := &Structure{Kind: KindCore, HP: t.Kinds[KindCore].HP}
	gs.SetStructure(cc, cc, core)
	gs.core = core
	gs.coreX, gs.coreY = cc, cc
	cp, _, _ := cellInChunk(cc, cc)
	gs.Fog = map[ChunkPos]bool{cp: true}
}
