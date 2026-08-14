# Art Style — ECAMS

The whole visual language collapses into one leading word: **ECAMS** — the Airbus Electronic Centralized Aircraft Monitoring System. Every screen should feel like operating a complex industrial machine: a dark monitoring terminal that reports the state of systems, not a "game view".

Reference implementation: `prototype_ecams.py` (throwaway). This doc is canonical; the prototype is the living example.

## The terminal

- Near-black green-tinted background `#070B09`.
- Everything in a monospace font — DejaVu Sans Mono on this system.
- Panels `#0E1612` with thin `#2D503C` borders.
- Chrome text: dim green `#5A826E`; readouts: bright green `#96D2B4`; amber `#EBBE46` for caution values (e.g. DEPTH); red `#EB5F41` for warnings.
- A status bar reports system state — `SYS NOMINAL`, `DEPTH`, `CURSOR`, `TILE` — updating on every action, with a context hint for the tile under the cursor.
- Header style: title in bright green, subtitle in dim green, DEPTH readout in amber.

## Entities

- Every entity is a basic shape — a rounded rect — with a single letter inside denoting what it is.
- A color legend always accompanies the grid: letter → color → name.
- New entities follow the same rule: a letter, a color, a legend entry, and a meaning in the palette below.

## Palette

The letters are the game's structure kinds (the prototype's letters were its own — this is canonical). Enemy and drone are blips, not kinds.

| Letter | Letter color | Fill | Entity | ECAMS meaning |
| ------ | ------------ | ---- | ------ | ------------- |
| C | `#96D2B4` bright green | `#561A16` | Core | the anchor; loss = system failure |
| E | `#46D264` green | `#122A1C` | Energy producer | nominal, normal ops |
| M | `#5ABEE6` cyan | `#102832` | Element machine | info / blue systems |
| F | `#B4D25A` yellow-green | `#202A0E` | Factory | conversion |
| T | `#EB6E4B` red-orange | `#2E1810` | Turret | weapon / warning |
| W | `#AA8C5A` tan | `#262010` | Wall | barrier |
| X | `#EB5F41` red | `#2E1810` | Enemy | threat |
| · | `#283C32` dim | `#161E1A` | Void / empty | — |
