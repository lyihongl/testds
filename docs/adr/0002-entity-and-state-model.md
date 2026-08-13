# Entity and state model: struct-per-type, one GameState, chunked grid

The Go base (ADR 0001) represents the game as **plain-data structs per entity kind** (structures, enemies, drones) with **behavior in system-shaped functions** (`SimulationStep(gs, dt)` → per-kind tick functions), a **single `GameState` container** as the sim root, a **chunked infinite grid** (`map[ChunkPos]*Chunk`, cells own structures), and **cosmetic effects as events** rather than state. Rationale: keeps "adding a new entity type" a localized addition, keeps the later component-system migration cheap, and keeps state save/load-ready — the serialization contract itself is deferred to the "Define the serializability model" ticket.

**Status:** accepted

**Considered options:** a component/ECS model now (overkill for a closed type set of 6 structures + enemy + drone; indirection while small), pointer-based references with methods on entities (simpler to write, but the component migration becomes a rewrite of the interaction layer), and cosmetics stored in state (simpler, but bloats the serialization root). Migration insurance instead: entities reference each other by id/grid-cell, never pointer; behavior lives in functions, not methods; systems reach structures only via GameState accessors (`StructureAt`/`SetStructure`/`AllStructures`), so a slice+index cache-layout swap is a GameState-internal change.

**Consequences:** GameState is the future serialization root (concrete types only, no interface-valued fields). Grid is chunked from day one (16×16, code-owned) even though the demo uses one chunk. Per-type structure rules dispatch through a type seam; the registry table is owned by the "Define the config and tuning approach" ticket. Sub-factory nesting/ports on the chunk model remains undecided. RNG lives in GameState but is reseeded on load; `game_over` is derived from core HP; metrics (killed/spawned/shots) are state but their saved status is the serializability ticket's call.

## Amendments (game-loop systems ticket, #17)

Spatial decisions from the chunking grill, recorded against the base:

- **Chunk size is code-owned at 16, not config-owned.** The chunk is the storage/serialization unit; changing it is a save-format break, never a runtime option.
- **The sim is chunk-agnostic.** Cells are the only logical coordinate; chunks are storage and serialization only. Enemies/drones crossing a chunk boundary are just a different map lookup — multi-chunk regions work by construction.
- **The de-fogged chunk set (`Fog`) is saved state** and the only gameplay-facing chunk concept: enemy pressure enters at the boundary of the de-fogged region. The demo world de-fogs one chunk; growth rules (defense coverage) belong to the escalation ticket.
- **The core is a cached pointer on GameState** — the one exception to "never pointer": it is a GameState-internal anchor (identity + cell), never serialized, re-derived on restore, cleared on destruction, not an entity-to-entity reference.
- **There is no depot point.** Drones route cell-to-cell to supply; the core's cell stands in as the stockpile point. Storage will be a future structure kind; the routing shape (drones target structure cells) makes that a registry change.
- **Iteration discipline:** systems consume one `AllStructures()` snapshot per frame, taken after the enemy system. In-place iteration (no materialized slice) is the future optimization, protected by the accessor boundary.
