# Package structure: `game` (pure logic) + `ui` (ebiten presentation), root entry

The Go base (ADR 0001) is split into two packages: **`game`** — state, sim systems, config, `Intent`/`Event` types, serialization; pure Go with no ebitengine import, so the sim is headless-testable via `go test ./game` — and **`ui`** — ebitengine presentation (render, input, window/loop, Oto audio), importing `game` only. The entry point is a thin root `main.go`; packages meet once per frame through `SimulationStep(gs *game.GameState, in game.Intent, dt float64) []game.Event`.

**Status:** accepted

**Considered options:** a single `game` package with ebiten inside (simpler, but drags cgo/GL deps into the headless harness and grows unbounded), a deeper tree with separate state/sim/config packages (import discipline the scale doesn't need — ADR 0002 settled on one shared GameState), and a `cmd/` entry (convention for multi-binary modules; one binary doesn't justify it).

**Consequences:** the sim never imports ebiten — the smoke harness (Implement the smoke harness) runs headless; ADR 0002's "render reads state without owning it" is now a package boundary; cosmetic events and input intent are contract types owned by `game`, consumed/translated by `ui`. Config lives in `game` (the registry *approach* is the config ticket's call). The package-structure implementation ticket reorganizes the current root spike + provisional `game/` into this layout.
