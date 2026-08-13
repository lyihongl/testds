package game

import (
	"encoding/json"
	"reflect"
	"testing"
)

func mustMarshal(t *testing.T, sf SaveFile) []byte {
	t.Helper()
	b, err := json.Marshal(sf)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// TestSnapshotRestoreRoundTrip populates a GameState, snapshots it, pushes it
// through JSON, restores into a fresh state, and re-snapshots: the two
// SaveFiles must be identical. The re-snapshot comparison catches every field
// that failed to survive the round trip.
func TestSnapshotRestoreRoundTrip(t *testing.T) {
	gs := NewGameState() // a fresh, reset game: core at the chunk center, fog {(0,0)}
	gs.Stockpile = Stockpile{El: 30, En: 12, Am: 4}
	gs.Time = 42.5
	gs.Spawn = SpawnState{Timer: 1.25, On: true}
	gs.Metrics = Metrics{Killed: 3, Spawned: 8, Shots: 11}
	gs.Fog[ChunkPos{X: 2, Y: -2}] = true // extra de-fogged chunk
	gs.SetStructure(2, 3, &Structure{Kind: KindEnergyProducer, HP: 50})
	gs.SetStructure(-1, -1, &Structure{Kind: KindTurret, HP: 50, Cooldown: 0.3, Buffer: Buffer{Am: 3}})
	// Negative + far coordinates exercise chunk wrapping and multiple chunks.
	gs.SetStructure(33, -20, &Structure{Kind: KindFactory, HP: 50, Timer: 0.7, Buffer: Buffer{El: 2, Am: 1}})
	gs.Enemies = append(gs.Enemies, Enemy{FX: 3.5, FY: 4.25, HP: 30})
	gs.Drones = append(gs.Drones, Drone{Item: "el", ToDepot: true, SX: 1, SY: 1, TX: 5, TY: 5, T: 0.5, Dur: 2.0, GX: 5, GY: 5})
	gs.RNG().Float64() // a draw must not leak into the save

	sf := gs.Snapshot()
	var sf2 SaveFile
	if err := json.Unmarshal(mustMarshal(t, sf), &sf2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	gs2 := NewGameState()
	if err := gs2.Restore(sf2); err != nil {
		t.Fatalf("restore: %v", err)
	}

	if got := gs2.Snapshot(); !reflect.DeepEqual(got, sf) {
		t.Errorf("round trip changed state:\n got  %+v\n want %+v", got, sf)
	}
}

// TestRestoreReDerivesCoreAnchor pins the grill Q2 contract: the core pointer
// and cell are caches, never saved — Restore must re-find them.
func TestRestoreReDerivesCoreAnchor(t *testing.T) {
	gs := NewGameState()
	sf := gs.Snapshot()
	gs2 := NewGameState()
	if err := gs2.Restore(sf); err != nil {
		t.Fatal(err)
	}
	cx, cy := gs2.CoreCell()
	if gs2.Core() == nil || cx != int64(CHUNK_SIZE/2) || cy != int64(CHUNK_SIZE/2) {
		t.Fatalf("core anchor not re-derived: cell (%d,%d) ptr %v", cx, cy, gs2.Core())
	}
}

// TestSnapshotExcludesRNG pins the contract: the RNG is sim-only and must
// never appear in the save (ADR 0002 reseeds on restore).
func TestSnapshotExcludesRNG(t *testing.T) {
	gs := NewGameState()
	var m map[string]any
	if err := json.Unmarshal(mustMarshal(t, gs.Snapshot()), &m); err != nil {
		t.Fatalf("unmarshal save: %v", err)
	}
	if _, ok := m["rng"]; ok {
		t.Fatal("save contains rng; it must be reseeded on restore, never saved")
	}
}

// TestRestoreRejectsUnknownVersion pins the version gate.
func TestRestoreRejectsUnknownVersion(t *testing.T) {
	gs := NewGameState()
	if err := gs.Restore(SaveFile{Version: 999}); err == nil {
		t.Fatal("expected a version error, got nil")
	}
}
