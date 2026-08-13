// Package game holds the simulation state (and, later, the systems) of the
// Go build of the game.
//
// Provisional home: package boundaries are the "Define the package structure"
// ticket's call (#11). The state shape here follows ADR 0002 (the "Define the
// entity and state model" resolution): struct-per-kind, entities as data, one
// GameState container, chunked infinite grid, cells own structures, accessor
// boundary, cosmetics are events (not state).
package game

import (
	"math/rand/v2"
	"time"
)

// CHUNK_SIZE is the number of cells per chunk side. A structural constant for
// now; the config ticket (#12) may make it tunable.
const CHUNK_SIZE = 16

// Structure kinds — the type seam. Per-kind rules (stats, behavior) live in
// the config registry (ticket #12), not in the struct.
const (
	KindCore           = "C"
	KindEnergyProducer = "E"
	KindElementMachine = "M"
	KindFactory        = "F"
	KindTurret         = "T"
	KindWall           = "W"
)

// ChunkPos identifies a chunk in the infinite grid. JSON map keys cannot be
// structs, so SaveFile carries chunks as a sorted array instead (save.go).
type ChunkPos struct {
	X int32
	Y int32
}

// Structure is a placed structure. The six kinds are Kind values on one
// concrete struct so cells (and SaveFile) stay concrete — ADR 0002 forbids
// interface-valued fields in state.
type Structure struct {
	Kind     string  `json:"kind"`
	HP       float64 `json:"hp"`
	Timer    float64 `json:"timer"`    // M/F production timers (folded in per #10)
	Cooldown float64 `json:"cooldown"` // T fire cooldown (folded in per #10)
	Buffer   Buffer  `json:"buffer"`   // M/F/T item buffer
}

// Buffer is a machine's item buffer.
type Buffer struct {
	El int `json:"el"`
	Am int `json:"am"`
}

// Chunk is a fixed-size cell grid. A cell owns its structure; nil means empty.
type Chunk struct {
	Cells [CHUNK_SIZE][CHUNK_SIZE]*Structure `json:"cells"`
}

// Enemy walks toward the core; FX/FY are continuous cell coordinates.
type Enemy struct {
	FX float64 `json:"fx"`
	FY float64 `json:"fy"`
	HP float64 `json:"hp"`
}

// Drone carries one item on a trip. S/T are continuous cell coordinates of
// the trip endpoints; GX/GY is the target cell.
type Drone struct {
	Item    string  `json:"item"`
	ToDepot bool    `json:"to_depot"`
	SX      float64 `json:"sx"`
	SY      float64 `json:"sy"`
	TX      float64 `json:"tx"`
	TY      float64 `json:"ty"`
	T       float64 `json:"t"`
	Dur     float64 `json:"dur"`
	GX      int64   `json:"gx"`
	GY      int64   `json:"gy"`
}

// Stockpile is the global economy.
type Stockpile struct {
	El int `json:"el"`
	En int `json:"en"`
	Am int `json:"am"`
}

// SpawnState is the enemy spawn gate.
type SpawnState struct {
	Timer float64 `json:"timer"`
	On    bool    `json:"on"`
}

// Metrics are counters that cannot be re-derived; their saved status is the
// serializability ticket's call (ADR 0002) — kept in the save.
type Metrics struct {
	Killed  int `json:"killed"`
	Spawned int `json:"spawned"`
	Shots   int `json:"shots"`
}

// GameState is the sim container and serialization root (ADR 0002). The rng
// field is deliberately unexported: it is sim-only and never serialized
// (reseeded on restore). Tune and Reg are likewise never serialized — they
// are config, not state (ADR 0004).
type GameState struct {
	Chunks    map[ChunkPos]*Chunk
	Stockpile Stockpile
	Enemies   []Enemy
	Drones    []Drone
	Time      float64
	Spawn     SpawnState
	Metrics   Metrics
	Tune      *Tuning   // live tuning table (not serialized)
	Reg       *Registry // structure-kind registry (not serialized)
	rng       *rand.Rand
}

// NewGameState returns a GameState with default tuning and a fresh RNG.
func NewGameState() *GameState {
	gs := &GameState{
		Chunks: make(map[ChunkPos]*Chunk),
		rng:    rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)),
	}
	gs.UseTuning(DefaultTuning())
	return gs
}

// UseTuning swaps the live tuning table and rebuilds the registry. Tuning is
// never serialized (SaveFile excludes it).
func (gs *GameState) UseTuning(t *Tuning) {
	gs.Tune = t
	gs.Reg = NewRegistry(t)
}

// StructureAt returns the structure at global cell (gx, gy), or nil.
func (gs *GameState) StructureAt(gx, gy int64) *Structure {
	cp, cx, cy := cellInChunk(gx, gy)
	chunk := gs.Chunks[cp]
	if chunk == nil {
		return nil
	}
	return chunk.Cells[cy][cx]
}

// SetStructure places s at global cell (gx, gy), creating the chunk if needed.
func (gs *GameState) SetStructure(gx, gy int64, s *Structure) {
	cp, cx, cy := cellInChunk(gx, gy)
	chunk := gs.Chunks[cp]
	if chunk == nil {
		chunk = &Chunk{}
		gs.Chunks[cp] = chunk
	}
	chunk.Cells[cy][cx] = s
}

// Placed is a structure together with its grid position. Cells own
// structures (ADR 0002), so the position is not stored on the struct itself;
// AllStructures pairs them up for iteration.
type Placed struct {
	X, Y int64
	S    *Structure
}

// AllStructures returns every placed structure (loaded chunks only) with its
// position.
func (gs *GameState) AllStructures() []Placed {
	var out []Placed
	for cp, chunk := range gs.Chunks {
		for cy := 0; cy < CHUNK_SIZE; cy++ {
			for cx := 0; cx < CHUNK_SIZE; cx++ {
				if s := chunk.Cells[cy][cx]; s != nil {
					out = append(out, Placed{
						X: int64(cp.X)<<4 | int64(cx),
						Y: int64(cp.Y)<<4 | int64(cy),
						S: s,
					})
				}
			}
		}
	}
	return out
}

// RNG exposes the sim's random source (systems draw from it; it is never
// serialized).
func (gs *GameState) RNG() *rand.Rand { return gs.rng }

// cellInChunk maps global cell coordinates to a chunk position plus in-chunk
// offsets. Shifts are arithmetic, so negative coordinates land in the correct
// chunk (floor division) with correct offsets.
func cellInChunk(gx, gy int64) (ChunkPos, int, int) {
	return ChunkPos{X: int32(gx >> 4), Y: int32(gy >> 4)}, int(gx & 15), int(gy & 15)
}
