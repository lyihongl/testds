package game

import (
	"math"
	"testing"
)

// stepN runs n frames of dt seconds each (events discarded).
func stepN(gs *GameState, n int, dt float64) {
	for i := 0; i < n; i++ {
		SimulationStep(gs, Intent{}, dt)
	}
}

func hasWarning(ev []Event, text string) bool {
	for _, e := range ev {
		if e.Kind == EventWarning && e.Text == text {
			return true
		}
	}
	return false
}

func TestSpawnEntersAtFogBorder(t *testing.T) {
	gs := NewGameState()
	SimulationStep(gs, Intent{ToggleSpawn: true}, 0)
	// First spawn after the grace period; stay short of the second (SpawnEvery).
	stepN(gs, int((gs.Tune.Grace+0.5)/0.016), 0.016)
	if len(gs.Enemies) != 1 || gs.Metrics.Spawned != 1 {
		t.Fatalf("want exactly 1 spawned enemy, got %d enemies, %d spawned", len(gs.Enemies), gs.Metrics.Spawned)
	}
	// The enemy origin (a fog-border cell) is pinned by
	// TestFogBoundaryIsTheSingleChunkBorder; here it must be inside the chunk.
	e := gs.Enemies[0]
	if e.FX < 0 || e.FX > CHUNK_SIZE || e.FY < 0 || e.FY > CHUNK_SIZE {
		t.Fatalf("enemy at (%v,%v) outside the demo chunk", e.FX, e.FY)
	}
}

// TestEnemyParksNearCoreAndChews pins the prototype's chew behavior: an enemy
// never quite reaches the core center — it parks ~0.5 cells out (where its
// next step would land in the core cell) and chews the core at full DPS
// forever, until the core dies or a turret kills it.
func TestEnemyParksNearCoreAndChews(t *testing.T) {
	gs := NewGameState()
	gs.Enemies = append(gs.Enemies, Enemy{FX: 8.5, FY: 13.5, HP: 30})
	stepN(gs, 60, 0.1) // 6s: walks 4.45 cells, then chews from ~0.55 out
	e := gs.Enemies[0]
	cx, cy := gs.CoreCell()
	dc := math.Hypot(e.FX-(float64(cx)+0.5), e.FY-(float64(cy)+0.5))
	if dc > 0.6 {
		t.Fatalf("enemy parked at dist %.2f from core center, want ~0.5", dc)
	}
	if hp := gs.Core().HP; hp >= 95 {
		t.Fatalf("core barely chewed: hp=%v (want < 95 after ~1.5s of 5dps)", hp)
	}
}

func TestWallBlocksAndGetsChewed(t *testing.T) {
	gs := NewGameState()
	gs.Enemies = append(gs.Enemies, Enemy{FX: 8.5, FY: 12.5, HP: 30})
	gs.SetStructure(8, 12, &Structure{Kind: KindWall, HP: 50})
	stepN(gs, 110, 0.1) // 10s of chewing (5dps) kills the wall, then it moves on
	if s := gs.StructureAt(8, 12); s != nil {
		t.Fatal("wall survived 10s of chewing")
	}
	if e := gs.Enemies[0]; e.FY >= 12.5 {
		t.Fatalf("enemy did not resume moving after the wall fell: fy=%v", e.FY)
	}
}

func TestEnergyProduction(t *testing.T) {
	gs := NewGameState()
	SimulationStep(gs, Intent{Place: true, Kind: KindEnergyProducer, CellX: 2, CellY: 2}, 0)
	if gs.Stockpile.El != 27 {
		t.Fatalf("build cost not charged: el=%d, want 27", gs.Stockpile.El)
	}
	stepN(gs, 10, 0.1) // 1s
	if gs.Stockpile.En < 9.5 || gs.Stockpile.En > 10.5 {
		t.Fatalf("en=%v, want ~10 (10/s)", gs.Stockpile.En)
	}
}

func TestElementMachineShipsElementsToStockpile(t *testing.T) {
	gs := NewGameState()
	SimulationStep(gs, Intent{Place: true, Kind: KindElementMachine, CellX: 2, CellY: 2}, 0)
	// Produce one element (2s), then let the drone make the trip (~0.94s).
	stepN(gs, 33, 0.1) // 3.3s: produced at 2.0, delivered ~2.94, nothing new until 4.0
	if gs.Stockpile.El != 28 {
		t.Fatalf("stockpile el=%d, want 28 (27 after build + 1 delivered)", gs.Stockpile.El)
	}
	if m := gs.StructureAt(2, 2); m.Buffer.El != 0 {
		t.Fatalf("machine buffer not drained: el=%d", m.Buffer.El)
	}
	if len(gs.Drones) != 0 {
		t.Fatalf("expected no drone in flight, got %d", len(gs.Drones))
	}
}

func TestTurretShootsNearestEnemy(t *testing.T) {
	gs := NewGameState()
	gs.Stockpile.En = 100 // bank enough energy for the turret build
	SimulationStep(gs, Intent{Place: true, Kind: KindTurret, CellX: 8, CellY: 6}, 0)
	turret := gs.StructureAt(8, 6)
	turret.Buffer.Am = 3
	gs.Enemies = append(gs.Enemies,
		Enemy{FX: 8.5, FY: 7.5, HP: 30}, // dist 1
		Enemy{FX: 8.5, FY: 9.5, HP: 30}, // dist 3
	)
	ev := SimulationStep(gs, Intent{}, 1.0)
	if e := gs.Enemies[0]; e.HP != 20 {
		t.Fatalf("nearest enemy hp=%v, want 20 (one 10-dmg shot)", e.HP)
	}
	if gs.Enemies[1].HP != 30 {
		t.Fatalf("far enemy damaged: hp=%v", gs.Enemies[1].HP)
	}
	if turret.Buffer.Am != 2 {
		t.Fatalf("ammo=%d, want 2 (1 per shot)", turret.Buffer.Am)
	}
	if gs.Metrics.Shots != 1 {
		t.Fatalf("shots=%d, want 1", gs.Metrics.Shots)
	}
	found := false
	for _, e := range ev {
		if e.Kind == EventTracer {
			found = true
		}
	}
	if !found {
		t.Fatal("no tracer event emitted")
	}
}

func TestPlacementCostsAndWarnings(t *testing.T) {
	gs := NewGameState() // el 30, en 0
	ev := SimulationStep(gs, Intent{Place: true, Kind: KindTurret, CellX: 1, CellY: 1}, 0)
	if gs.StructureAt(1, 1) != nil {
		t.Fatal("turret placed without energy")
	}
	if !hasWarning(ev, "INSUFFICIENT STOCKPILE") {
		t.Fatal("no insufficient-stockpile warning")
	}
	SimulationStep(gs, Intent{Place: true, Kind: KindWall, CellX: 2, CellY: 2}, 0)
	ev = SimulationStep(gs, Intent{Place: true, Kind: KindWall, CellX: 2, CellY: 2}, 0)
	if !hasWarning(ev, "CELL OCCUPIED") {
		t.Fatal("no occupied warning")
	}
	if gs.Stockpile.El != 28 {
		t.Fatalf("el=%d, want 28 (30 - 2 for the wall)", gs.Stockpile.El)
	}
}

func TestGameOverFreezesSim(t *testing.T) {
	gs := NewGameState()
	gs.Enemies = append(gs.Enemies, Enemy{FX: 8.5, FY: 8.6, HP: 30}) // in the core cell
	stepN(gs, 30, 1.0)                                               // 30s × 5dps > 100 core hp
	if !gs.GameOver() {
		t.Fatal("game not over")
	}
	if gs.Core() != nil {
		t.Fatal("core pointer not cleared on destruction")
	}
	if s := gs.StructureAt(8, 8); s != nil {
		t.Fatal("core cell not emptied")
	}
	t0 := gs.Time
	SimulationStep(gs, Intent{}, 1.0)
	if gs.Time != t0 {
		t.Fatalf("time advanced after game over: %v -> %v", t0, gs.Time)
	}
}

func TestFogBoundaryIsTheSingleChunkBorder(t *testing.T) {
	gs := NewGameState()
	// With one de-fogged chunk, every boundary cell sits on the 0/15 edges.
	for i := 0; i < 100; i++ {
		x, y, ok := gs.randomSpawnCell()
		if !ok {
			t.Fatal("no boundary cell found")
		}
		if x != 0 && x != CHUNK_SIZE-1 && y != 0 && y != CHUNK_SIZE-1 {
			t.Fatalf("boundary cell (%d,%d) not on the chunk border", x, y)
		}
	}
}
