# The prototype's role: shape, not implementation

**`prototype_game_loop.py` (and `prototype_ecams.py`) exist to mock out the basic
shape of the game** — whether the loop feels playable, what the economy looks
like in motion, what the ECAMS presentation reads like. That is their only job.

**They are NOT an implementation reference.** Do not port constants, do not
copy logic structure, do not treat a prototype behavior as the intended
behavior. The Go codebase is the implementation; the prototype is a rough
sketch of the general shape it implements.

## Why

Porting the prototype faithfully has repeatedly produced the wrong thing.
Copying its details transplanted throwaway scaffolding into the real code:

- its fixed **10×10 board** (a rendering convenience — the game's world is
  chunk-based),
- its off-board **depot point** (a visual anchor — the real game has no depot;
  drones route cell-to-cell, and the core's cell is the stockpile point),
- its **never-debit stockpile** (elements/ammo were never consumed, a demo
  hack — the Go economy debits on conversion and turret fire),
- its **pixel-space positions** (the Go sim is cell-space continuous).

## Rules for reading the prototype

1. **Numbers are tuning-pending.** Every value in the prototype is a demo
   number (the tuning spec says so explicitly). The `Tuning` table in Go is
   the real balance source.
2. **The general shape is the reference**: core under continuous enemy flow,
   five placeable structures, request-based drones, a global energy economy,
   the ECAMS look. Copy the *shape*, rebuild the *details* from first
   principles.
3. **Check the ADRs and the ticket records for deliberate divergences.**
   ADR 0002's amendments and the resolutions of the chunking grill (#17),
   the config registry (#16), and the serializability model (#13) record
   where the Go build intentionally departs from the prototype and why.
4. **When in doubt, the Go code + its tests are the source of truth**, not
   the prototype.

## How to flag a prototype behavior

If a prototype behavior looks wrong, suspect the prototype first. Ask: is
this a real gameplay choice, or an artifact of the sketch? (Examples already
found this way: enemies parking ~0.5 cells from the core and chewing forever
— faithful to the prototype's *logic*, but a suspicious shape; the demo
balance being harsh once the economy actually debits.) Record what you find
in the relevant ticket/ADR, then decide on the Go side.
