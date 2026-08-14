# Context

> How to read the Python prototypes: `docs/prototype-role.md` — they mock out
> the basic shape of the game; the Go build is the implementation.

## Glossary

- **sub-factory** — a tile that opens into a contained nested factory; encapsulation/functions applied to factory automation. The core mechanic of this project.
- **port** — a sub-factory tile's input/output interface; the only thing a parent grid sees of the inside.
- **drive** — the energy component that opens a sub-factory tile. Cost model: simple tree sum across nesting levels; balanced, tuning deferred.
- **saved layout** — a reusable sub-factory definition (a "function"), placeable as an instance.
- **element machine** — a sci-fi box that compresses/decomposes air into any element at a fixed rate; the raw-materials source.
- **energy producer** — a machine generating energy for weapons and factories.
- **wave** — the recurring survival-pressure event; attacks the base and destroys machines outside defended territory. No discrete waves in the current loop — pressure is continuous enemy flow.
- **defense coverage** — the expansion gate; territory is only usable while defended.
- **core** — the single building whose destruction ends the game.
- **wall** — a cheap blocking structure; enemies chew through it.
- **turret** — the single weapon type; consumes ammo, targets the nearest enemy in range.
- **drone** — the transport layer; ferries items between machines on request.
- **logistics** — the drone transport module (game/logistics.go): dedups trips per machine cell+item, moves drones, and delivers between machine buffers and the stockpile point. Its interface is requestDrone + stepDrones; machines never touch trip mechanics.
