package game

import "math"

// Systems implement the prototype's update loop (ticket #17) as per-frame
// functions over GameState (ADR 0002: behavior in system-shaped functions).
// Frame order in SimulationStep: spawn, enemies, then a single AllStructures
// snapshot (grill Q7), then energy, machines (production + drone requests),
// drones (trips + delivery), turrets.

// gridCell is a plain integer cell coordinate.
type gridCell struct{ X, Y int64 }

// ---------- spawn: pressure enters at the de-fogged region's border ----------

func stepSpawn(gs *GameState, dt float64) {
	if !gs.Spawn.On {
		return
	}
	gs.Spawn.Timer -= dt
	if gs.Spawn.Timer > 0 {
		return
	}
	gs.Spawn.Timer = gs.Tune.SpawnEvery
	x, y, ok := gs.randomSpawnCell()
	if !ok {
		return
	}
	gs.Enemies = append(gs.Enemies, Enemy{
		FX: float64(x) + 0.5,
		FY: float64(y) + 0.5,
		HP: gs.Tune.EnemyHP,
	})
	gs.Metrics.Spawned++
}

// randomSpawnCell picks a random empty cell on the boundary of the de-fogged
// region (grill Q3): a region's outer edge cells are those whose neighbor
// chunk across the edge is still fogged. The demo world de-fogs one chunk, so
// this is that chunk's border. Falls back to occupied boundary cells if none
// are free; ok=false only if the region has no boundary at all.
func (gs *GameState) randomSpawnCell() (int64, int64, bool) {
	var empty, all []gridCell
	for cp := range gs.Fog {
		x0, y0 := int64(cp.X)*CHUNK_SIZE, int64(cp.Y)*CHUNK_SIZE
		for i := int64(0); i < CHUNK_SIZE; i++ {
			edges := [4]struct {
				cell   gridCell
				nx, ny int32
			}{
				{gridCell{x0 + i, y0}, cp.X, cp.Y - 1},
				{gridCell{x0 + i, y0 + CHUNK_SIZE - 1}, cp.X, cp.Y + 1},
				{gridCell{x0, y0 + i}, cp.X - 1, cp.Y},
				{gridCell{x0 + CHUNK_SIZE - 1, y0 + i}, cp.X + 1, cp.Y},
			}
			for _, e := range edges {
				if gs.Fog[ChunkPos{X: e.nx, Y: e.ny}] {
					continue // interior edge between two de-fogged chunks
				}
				all = append(all, e.cell)
				if gs.StructureAt(e.cell.X, e.cell.Y) == nil {
					empty = append(empty, e.cell)
				}
			}
		}
	}
	pool := empty
	if len(pool) == 0 {
		pool = all
	}
	if len(pool) == 0 {
		return 0, 0, false
	}
	c := pool[gs.rng.IntN(len(pool))]
	return c.X, c.Y, true
}

// ---------- enemies: walk toward the core, chew whatever blocks them ----------

func stepEnemies(gs *GameState, dt float64) {
	tc := gs.coreCenter()
	speed := gs.Tune.EnemySpeed
	dps := gs.Tune.EnemyDPS
	for i := range gs.Enemies {
		e := &gs.Enemies[i]
		dx, dy := tc.X-e.FX, tc.Y-e.FY
		dist := math.Hypot(dx, dy)
		if dist < 0.05 {
			e.FX, e.FY = tc.X, tc.Y
			continue
		}
		step := speed * dt / dist
		nx, ny := e.FX+dx*step, e.FY+dy*step
		s := gs.StructureAt(int64(nx), int64(ny))
		if s != nil {
			// The cell being entered is occupied: chew it and hold position.
			s.HP -= dps * dt
			if s.HP <= 0 {
				gs.SetStructure(int64(nx), int64(ny), nil)
				if s.Kind == KindCore {
					gs.core = nil // core destroyed: game over
				}
			}
		} else {
			e.FX, e.FY = nx, ny
		}
	}
}

// ---------- energy: flat global production ----------

func stepEnergy(gs *GameState, placed []Placed, dt float64) {
	e, ok := gs.Reg.Get(KindEnergyProducer)
	if !ok {
		return
	}
	rate := e.Stats.EnergyPerSec
	for _, p := range placed {
		if p.S.Kind == KindEnergyProducer {
			gs.Stockpile.En += rate * dt
		}
	}
}

// ---------- machines: production + logistics requests ----------

func stepMachines(gs *GameState, placed []Placed, dt float64) {
	for _, p := range placed {
		switch p.S.Kind {
		case KindElementMachine:
			st, _ := gs.Reg.Get(KindElementMachine)
			s := p.S
			s.Timer += dt
			if s.Timer >= st.Stats.ProduceEvery {
				s.Timer -= st.Stats.ProduceEvery
				s.Buffer.El = min(s.Buffer.El+1, int(st.Stats.Buffer))
			}
			// Ship elements to the stockpile (core) as they accumulate.
			if s.Buffer.El > 0 {
				requestDrone(gs, p.X, p.Y, "el", true)
			}
		case KindFactory:
			st, _ := gs.Reg.Get(KindFactory)
			s := p.S
			s.Timer += dt
			if s.Timer >= st.Stats.ProduceEvery && s.Buffer.El >= int(st.Stats.FactoryEl) {
				s.Timer -= st.Stats.ProduceEvery
				s.Buffer.El -= int(st.Stats.FactoryEl)
				s.Buffer.Am = min(s.Buffer.Am+int(st.Stats.FactoryAmmo), int(st.Stats.Buffer))
			}
			// Ship ammo to the stockpile (core) as it accumulates.
			if s.Buffer.Am > 0 {
				requestDrone(gs, p.X, p.Y, "am", true)
			}
			// Import elements from the stockpile for conversion.
			if s.Buffer.El < int(st.Stats.Buffer) && gs.Stockpile.El > 0 {
				requestDrone(gs, p.X, p.Y, "el", false)
			}
		case KindTurret:
			st, _ := gs.Reg.Get(KindTurret)
			s := p.S
			// Import ammo from the stockpile when the turret has capacity.
			if s.Buffer.Am < int(st.Stats.Buffer) && gs.Stockpile.Am > 0 {
				requestDrone(gs, p.X, p.Y, "am", false)
			}
		}
	}
}

// ---------- turrets: nearest enemy in range, consume ammo ----------

func stepTurrets(gs *GameState, placed []Placed, dt float64) []Event {
	st, ok := gs.Reg.Get(KindTurret)
	if !ok {
		return nil
	}
	var ev []Event
	for _, p := range placed {
		s := p.S
		if s.Kind != KindTurret {
			continue
		}
		s.Cooldown -= dt
		if s.Cooldown > 0 {
			continue
		}
		if s.Buffer.Am < int(st.Stats.TurretAmmoCost) {
			s.Cooldown = 0
			continue
		}
		tx, ty := float64(p.X)+0.5, float64(p.Y)+0.5
		best, bd := -1, 0.0
		for i := range gs.Enemies {
			e := &gs.Enemies[i]
			d := math.Hypot(e.FX-tx, e.FY-ty)
			if d <= st.Stats.TurretRange && (best < 0 || d < bd) {
				best, bd = i, d
			}
		}
		if best >= 0 {
			e := &gs.Enemies[best]
			e.HP -= st.Stats.TurretDmg
			s.Buffer.Am -= int(st.Stats.TurretAmmoCost)
			s.Cooldown = 1.0 / st.Stats.TurretShots
			gs.Metrics.Shots++
			ev = append(ev, Event{
				Kind: EventTracer,
				A:    Vec{X: tx, Y: ty},
				B:    Vec{X: e.FX, Y: e.FY},
				Dur:  0.12,
			})
		}
	}
	// Dead enemies leave the sim; count them as kills.
	before := len(gs.Enemies)
	alive := gs.Enemies[:0]
	for _, e := range gs.Enemies {
		if e.HP > 0 {
			alive = append(alive, e)
		}
	}
	gs.Enemies = alive
	gs.Metrics.Killed += before - len(gs.Enemies)
	return ev
}

// ---------- placement ----------

// placeStructure validates and applies one placement intent: the cell must be
// free and the costs (el/en from the stockpile) affordable. Failures warn via
// events. The core is not placeable — only Reset creates it.
func placeStructure(gs *GameState, kind string, gx, gy int64) []Event {
	if kind == KindCore {
		return nil
	}
	if gs.StructureAt(gx, gy) != nil {
		return []Event{{Kind: EventWarning, Text: "CELL OCCUPIED", Dur: 1.0}}
	}
	e, ok := gs.Reg.Get(kind)
	if !ok {
		return nil
	}
	st := e.Stats
	if gs.Stockpile.El < int(st.CostEl) || gs.Stockpile.En < st.CostEn {
		return []Event{{Kind: EventWarning, Text: "INSUFFICIENT STOCKPILE", Dur: 1.0}}
	}
	gs.Stockpile.El -= int(st.CostEl)
	gs.Stockpile.En -= st.CostEn
	gs.SetStructure(gx, gy, &Structure{Kind: kind, HP: st.HP})
	return nil
}
