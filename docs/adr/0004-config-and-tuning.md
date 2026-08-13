# Config and tuning: Go table defaults + TOML override, debug-only slider panel

Tuning lives in the Go base (ADR 0001) as a typed **source table** (`Tuning` + `DefaultTuning()` in `game`) layered with an optional **TOML** override file (`config.toml`, `BurntSushi/toml` — present fields only). A **registry** (`map[Kind]Entry{Stats, Tick func}`) is the type seam: one registration per structure kind bundles its stats with its per-tick behavior. A dev-only **ebitenui slider panel** live-tunes values, gated by `//go:build debug` so a production `go build` physically omits it; a save button writes the merged tuning back to `config.toml`. Player-facing UI is unaffected — it stays ECAMS hand-drawn.

**Status:** accepted

**Considered options:** embedded JSON (no comments, rejected for the override file), a runtime `--debug` flag (ships the code, weaker guarantee than build tags), hand-rolled ECAMS sliders (full control, but the user preferred less own widget code — ebitenui is pure Go, no cgo), and a C imgui binding (cgo + non-ECAMS, rejected).

**Consequences:** `GameState` carries a `Tune *Tuning` pointer that is never serialized (SaveFile excludes it), keeping `SimulationStep(gs, in, dt)` unchanged. The base `App` exposes nil `drawDebug`/`updateDebug` hooks; debug-tagged files install the panel, so production never references ebitenui. Live values are reached by `Tunable` descriptors (`{Name, V *float64, Min, Max}`) pointing into the shared structs — single-threaded, no races. The tuning numbers from "Define the basic game loop" are the defaults (all tuning-pending).
