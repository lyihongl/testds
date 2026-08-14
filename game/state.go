// Package game holds the simulation state and systems of the Go build of the
// game.
//
// Package boundaries are the "Define the package structure" ticket's call
// (#11). The state shape follows ADR 0002 (the "Define the entity and state
// model" resolution): struct-per-kind, entities as data, one GameState
// container, chunked infinite grid, cells own structures, accessor boundary,
// cosmetics are events (not state).
package game

import (
	"math/rand/v2"
	"time"
)

// CHUNK_SIZE is the number of cells per chunk side. Code-owned, not tunable
// (grill Q5): the chunk is the storage/serialization unit, and changing it is
// a save-format break, not a config option.
const CHUNK_SIZE = 16

// Spatial model: the sim thinks in cells — continuous cell coordinates for
// movement, integer cell coordinates for the grid. Chunks are storage and
// serialization only: StructureAt/SetStructure index by cell and create
// chunks on demand, so enemies or drones crossing a chunk boundary are just a
// different map lookup — multi-chunk regions work by construction (grill Q8).
// The only gameplay-facing chunk concept is the de-fogged set (Fog), which
// defines where enemy pressure enters (grill Q3).

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

// Stockpile is the global economy. El and Am are the item inventory at the
// stockpile point (the core's cell): elements earned by element machines,
// ammo made by factories, spent on builds and turret fire. En is the energy
// reservoir — a float, because energy accrues continuously. Items in transit
// live in machine buffers, not here.
type Stockpile struct {
	El int     `json:"el"`
	En float64 `json:"en"`
	Am int     `json:"am"`
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

// GameState is the sim container and serialization root (ADR 0002). Unexported
// fields are sim-only and never serialized: rng (reseeded on restore), Tune
// and Reg (config, ADR 0004), and the core anchor (grill Q2). Fog is state —
// the de-fogged territory that defines the spawn boundary (grill Q3).
type GameState struct {
	Chunks        map[ChunkPos]*Chunk
	Stockpile     Stockpile
	Enemies       []Enemy
	Drones        []Drone
	Time          float64
	Spawn         SpawnState
	Metrics       Metrics
	Fog           map[ChunkPos]bool // de-fogged chunks: territory, spawn boundary
	Tune          *Tuning           // live tuning table (not serialized)
	Reg           *Registry         // structure-kind registry (not serialized)
	core          *Structure        // anchor: the single core (never serialized)
	coreX         int64             // the core's cell (never serialized)
	coreY         int64
	placedScratch []Placed // AllStructures iteration buffer (never serialized)
	rng           *rand.Rand
}

// NewGameState returns a fresh, reset game: default tuning, core seeded at
// the chunk center, its chunk de-fogged. Callers that load config re-apply
// UseTuning then Reset.
func NewGameState() *GameState {
	gs := &GameState{
		Chunks: make(map[ChunkPos]*Chunk),
		rng:    rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)),
	}
	gs.UseTuning(DefaultTuning())
	gs.Reset()
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
// position. It fills a scratch slice owned by gs — zero allocation per call,
// so the frame's snapshot cost no longer grows with the structure count
// (callers must not retain the result beyond the frame).
func (gs *GameState) AllStructures() []Placed {
	gs.placedScratch = gs.placedScratch[:0]
	for cp, chunk := range gs.Chunks {
		for cy := 0; cy < CHUNK_SIZE; cy++ {
			for cx := 0; cx < CHUNK_SIZE; cx++ {
				if s := chunk.Cells[cy][cx]; s != nil {
					gs.placedScratch = append(gs.placedScratch, Placed{
						X: int64(cp.X)*CHUNK_SIZE + int64(cx),
						Y: int64(cp.Y)*CHUNK_SIZE + int64(cy),
						S: s,
					})
				}
			}
		}
	}
	return gs.placedScratch
}

// RNG exposes the sim's random source (systems draw from it; it is never
// serialized).
func (gs *GameState) RNG() *rand.Rand { return gs.rng }

// Core returns the core structure, or nil after its destruction. The core is
// the game's single anchor: enemies target it, its death is game over. It is
// a cached pointer (grill Q2), never serialized — Reset seeds it, destruction
// clears it, Restore re-derives it.
func (gs *GameState) Core() *Structure { return gs.core }

// CoreCell returns the core's grid cell.
func (gs *GameState) CoreCell() (int64, int64) { return gs.coreX, gs.coreY }

// coreCenter returns the core's cell center in continuous cell coordinates
// (what enemies walk toward).
func (gs *GameState) coreCenter() Vec {
	return Vec{X: float64(gs.coreX) + 0.5, Y: float64(gs.coreY) + 0.5}
}

// GameOver is derived, never stored (ADR 0002): true once the core is
// destroyed.
func (gs *GameState) GameOver() bool { return gs.core == nil || gs.core.HP <= 0 }

// cellInChunk maps global cell coordinates to a chunk position plus in-chunk
// offsets, using floor division so negative coordinates land in the correct
// chunk (grill Q6 — explicit math, no power-of-two assumption).
func cellInChunk(gx, gy int64) (ChunkPos, int, int) {
	return ChunkPos{X: int32(floorDiv(gx, CHUNK_SIZE)), Y: int32(floorDiv(gy, CHUNK_SIZE))},
		int(floorMod(gx, CHUNK_SIZE)), int(floorMod(gy, CHUNK_SIZE))
}

// floorDiv and floorMod implement mathematical floor division and modulo.
// Go's / and % truncate toward zero; floor semantics are required so that
// negative cells map into the correct chunk.
func floorDiv(a, n int64) int64 {
	if q := a / n; a%n < 0 {
		return q - 1
	} else {
		return q
	}
}

func floorMod(a, n int64) int64 {
	if m := a % n; m < 0 {
		return m + n
	} else {
		return m
	}
}
