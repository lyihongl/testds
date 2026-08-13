# Build the real game in Go

The game loop was validated as a playable Python/pygame prototype (`prototype_game_loop.py`); the next step is an extensible codebase to grow features on. We will build that codebase in **Go**, not Python. Go buys a single static binary, headroom for the continuous real-time simulation, a type system that scales with the entity/state model, a native test runner (`go test`) for the smoke harness, and clean serialization (`encoding/json`/`gob`) for the settled serializable-state constraint.

**Status:** accepted

**Considered options:** staying in Python/pygame (fastest iteration, exact prototype parity) was rejected because the prototype's single-file `Demo` class is already at the limit of what that shape supports as features accumulate; other systems languages were not seriously considered for a hobby project.

**Consequences:** the graphics/audio binding is open — ebitengine is the likely choice and is resolved by the "Define the package structure" ticket. The Python prototype remains a *shape* reference, not an implementation reference — see `docs/prototype-role.md` for how to read it and the deliberate divergences; feature parity is measured against its general shape, and pygame stays for the prototype only. Toolchain is installed user-locally at `~/.local/go` (go1.26.5, no sudo); `go run .` from the repo root runs the stub entry point (`main.go`, module `coredefense`).
