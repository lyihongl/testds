package game

import "testing"

func TestSimulationStepAdvancesTime(t *testing.T) {
	gs := NewGameState()
	SimulationStep(gs, Intent{}, 0.5)
	if gs.Time != 0.5 {
		t.Fatalf("time = %v, want 0.5", gs.Time)
	}
}

func TestToggleSpawnFlipsSpawnState(t *testing.T) {
	gs := NewGameState()
	SimulationStep(gs, Intent{ToggleSpawn: true}, 0)
	if !gs.Spawn.On {
		t.Fatal("spawn not toggled on")
	}
	SimulationStep(gs, Intent{ToggleSpawn: true}, 0)
	if gs.Spawn.On {
		t.Fatal("spawn not toggled off")
	}
}

func TestRestartResetsWorld(t *testing.T) {
	gs := NewGameState()
	gs.SetStructure(0, 0, &Structure{Kind: KindWall, HP: 50})
	gs.Stockpile.El = 100
	gs.Time = 99
	SimulationStep(gs, Intent{Restart: true}, 0)
	if gs.Time != 0 || gs.Stockpile.El != 30 {
		t.Fatalf("restart left state: time=%v el=%v", gs.Time, gs.Stockpile.El)
	}
	cx, cy := gs.CoreCell()
	if gs.Core() == nil || cx != int64(CHUNK_SIZE/2) || cy != int64(CHUNK_SIZE/2) {
		t.Fatalf("core not reseeded at the chunk center: cell (%d,%d)", cx, cy)
	}
	if gs.StructureAt(0, 0) != nil {
		t.Fatal("old structure survived restart")
	}
	if len(gs.Fog) != 1 {
		t.Fatalf("fog not reseeded: %v", gs.Fog)
	}
}

func TestPlaceFillsEmptyCellOnly(t *testing.T) {
	gs := NewGameState()
	SimulationStep(gs, Intent{Place: true, Kind: KindWall, CellX: 2, CellY: 3}, 0)
	if s := gs.StructureAt(2, 3); s == nil || s.Kind != KindWall {
		t.Fatal("structure not placed")
	}
	SimulationStep(gs, Intent{Place: true, Kind: KindTurret, CellX: 2, CellY: 3}, 0)
	if s := gs.StructureAt(2, 3); s.Kind != KindWall {
		t.Fatal("place overwrote an occupied cell")
	}
}
