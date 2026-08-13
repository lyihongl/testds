package game

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"time"
)

// SaveVersion is the schema version of SaveFile. Bump on breaking changes;
// Restore rejects unknown versions.
// 2: Stockpile.En is float64; SaveFile gained Fog (de-fogged chunks).
const SaveVersion = 2

// SaveFile is the wire format for a saved game (JSON). It mirrors the
// serializable subset of GameState — everything except the RNG (reseeded on
// restore), cosmetics (evented, not state), the tuning/registry (config, ADR
// 0004), the core anchor (re-derived on restore, grill Q2), and derived
// values (game_over is derived from the core's HP, never stored).
type SaveFile struct {
	Version   int          `json:"version"`
	Chunks    []SavedChunk `json:"chunks"`
	Fog       []SavedPos   `json:"fog"`
	Stockpile Stockpile    `json:"stockpile"`
	Enemies   []Enemy      `json:"enemies"`
	Drones    []Drone      `json:"drones"`
	Time      float64      `json:"time"`
	Spawn     SpawnState   `json:"spawn"`
	Metrics   Metrics      `json:"metrics"`
}

// SavedPos is a chunk position for the fog set. An array (sorted) instead of
// a map so JSON output is deterministic.
type SavedPos struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

// SavedChunk is a chunk plus its position. An array (sorted) instead of a map
// so JSON output is deterministic.
type SavedChunk struct {
	X     int32 `json:"x"`
	Y     int32 `json:"y"`
	Chunk Chunk `json:"chunk"`
}

// Snapshot deep-copies the serializable subset of gs into a SaveFile. It is
// the save-ready hook: marshal the result with encoding/json to get a file.
// The save/load *feature* itself is out of scope (map #9).
func (gs *GameState) Snapshot() SaveFile {
	sf := SaveFile{
		Version:   SaveVersion,
		Stockpile: gs.Stockpile,
		Enemies:   append([]Enemy(nil), gs.Enemies...),
		Drones:    append([]Drone(nil), gs.Drones...),
		Time:      gs.Time,
		Spawn:     gs.Spawn,
		Metrics:   gs.Metrics,
	}
	for cp := range gs.Fog {
		sf.Fog = append(sf.Fog, SavedPos{X: cp.X, Y: cp.Y})
	}
	sort.Slice(sf.Fog, func(i, j int) bool {
		if sf.Fog[i].X != sf.Fog[j].X {
			return sf.Fog[i].X < sf.Fog[j].X
		}
		return sf.Fog[i].Y < sf.Fog[j].Y
	})
	for cp, chunk := range gs.Chunks {
		sc := SavedChunk{X: cp.X, Y: cp.Y, Chunk: *chunk}
		for cy := 0; cy < CHUNK_SIZE; cy++ {
			for cx := 0; cx < CHUNK_SIZE; cx++ {
				if s := chunk.Cells[cy][cx]; s != nil {
					c := *s
					sc.Chunk.Cells[cy][cx] = &c
				}
			}
		}
		sf.Chunks = append(sf.Chunks, sc)
	}
	sort.Slice(sf.Chunks, func(i, j int) bool {
		if sf.Chunks[i].X != sf.Chunks[j].X {
			return sf.Chunks[i].X < sf.Chunks[j].X
		}
		return sf.Chunks[i].Y < sf.Chunks[j].Y
	})
	return sf
}

// Restore overwrites gs with the state in sf and gives it a fresh RNG (per
// ADR 0002, the RNG is never saved). Unknown versions are an error. The core
// anchor is re-derived by scanning the restored grid, and an empty fog set
// (a legacy save without fog) falls back to the core's chunk so the game
// stays playable.
func (gs *GameState) Restore(sf SaveFile) error {
	if sf.Version != SaveVersion {
		return fmt.Errorf("save version %d not supported (want %d)", sf.Version, SaveVersion)
	}
	gs.Chunks = make(map[ChunkPos]*Chunk, len(sf.Chunks))
	for _, sc := range sf.Chunks {
		gs.Chunks[ChunkPos{X: sc.X, Y: sc.Y}] = &sc.Chunk
	}
	gs.Stockpile = sf.Stockpile
	gs.Enemies = append([]Enemy(nil), sf.Enemies...)
	gs.Drones = append([]Drone(nil), sf.Drones...)
	gs.Time = sf.Time
	gs.Spawn = sf.Spawn
	gs.Metrics = sf.Metrics
	gs.Fog = make(map[ChunkPos]bool, len(sf.Fog))
	for _, p := range sf.Fog {
		gs.Fog[ChunkPos{X: p.X, Y: p.Y}] = true
	}
	// Re-derive the core anchor (never saved, grill Q2).
	gs.core, gs.coreX, gs.coreY = nil, 0, 0
	for cp, chunk := range gs.Chunks {
		for cy := 0; cy < CHUNK_SIZE; cy++ {
			for cx := 0; cx < CHUNK_SIZE; cx++ {
				if s := chunk.Cells[cy][cx]; s != nil && s.Kind == KindCore {
					gs.core = s
					gs.coreX = int64(cp.X)*CHUNK_SIZE + int64(cx)
					gs.coreY = int64(cp.Y)*CHUNK_SIZE + int64(cy)
				}
			}
		}
	}
	if len(gs.Fog) == 0 && gs.core != nil {
		cp, _, _ := cellInChunk(gs.coreX, gs.coreY)
		gs.Fog[cp] = true
	}
	gs.rng = rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0))
	return nil
}
