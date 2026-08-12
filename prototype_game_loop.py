"""PROTOTYPE - throwaway. Basic game loop demo in the ECAMS terminal look.

Answers (ticket "Prototype the game loop demo"): does the loop feel playable,
and what breaks? Built per the spec in "Define the basic game loop" (#5):
a core under continuous enemy flow, five placeable structures, request-based
drones, global-stockpile economy. All numbers are demo values, tuning-pending.
Sub-factory mechanic is NOT part of this demo build (see spec).

Run:  .venv/bin/python prototype_game_loop.py
      .venv/bin/python prototype_game_loop.py --sim   # headless 30s smoke test
Controls: ARROWS move cursor | 1-5 select structure | ENTER place
          G toggle enemy spawn | R restart | ESC exit
"""

import math
import os
import random
import sys

import pygame

SIM = "--sim" in sys.argv
if SIM:
    os.environ.setdefault("SDL_VIDEODRIVER", "dummy")

pygame.init()

W, H = 1280, 720
GRID_N = 10
CELL = 44
GRID_PX = GRID_N * CELL
GRID_X, GRID_Y = 60, 110

BG = (7, 11, 9)
PANEL = (14, 22, 18)
BORDER = (45, 80, 60)
TEXT_DIM = (90, 130, 110)
TEXT = (150, 210, 180)
AMBER = (235, 190, 70)
RED = (235, 95, 65)
WHITE = (255, 255, 255)

FONT_MONO = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"

# ---- tuning (from the #5 spec; all tuning-pending) ------------------------
CORE_HP = 100
STRUCT_HP = 50
ENEMY_HP = 30
ENEMY_DPS = 5.0          # dmg/s while chewing a structure
ENEMY_SPEED = 1.0        # cells per second
SPAWN_EVERY = 2.5        # seconds between spawns, constant for the demo
GRACE = 6.0              # seconds before the first spawn
TURRET_RANGE = 4.0       # cells
TURRET_SHOTS = 1.0       # per second
TURRET_DMG = 10.0
TURRET_AMMO_COST = 1
ENERGY_PER_S = 10.0
ELEMENT_EVERY = 2.0      # seconds per element produced
FACTORY_EL = 2
FACTORY_AMMO = 1
FACTORY_EVERY = 1.0
DRONE_SPEED = 9.0        # cells per second (demo plumbing; raised so drones aren't the choke)
BUFFER = 3               # per-machine buffer capacity
START_EL = 30

# type -> (letter, letter color, fill, name, build cost, hp)
STRUCTS = {
    "C": ("C", WHITE, (86, 26, 22), "CORE", {}, CORE_HP),
    "E": ("E", (70, 210, 100), (18, 42, 28), "ENERGY PROD", {"el": 3}, STRUCT_HP),
    "M": ("M", (90, 190, 230), (16, 40, 50), "ELEMENT MACH", {"el": 3}, STRUCT_HP),
    "F": ("F", (180, 210, 90), (32, 42, 14), "FACTORY", {"el": 5}, STRUCT_HP),
    "T": ("T", (235, 110, 75), (46, 24, 16), "TURRET", {"el": 5, "en": 10}, STRUCT_HP),
    "W": ("W", (170, 140, 90), (38, 32, 16), "WALL", {"el": 2}, STRUCT_HP),
}
PLACEABLE = ["E", "M", "F", "T", "W"]
SELECT_KEYS = {pygame.K_1: "E", pygame.K_2: "M", pygame.K_3: "F", pygame.K_4: "T", pygame.K_5: "W"}

ENEMY_LC, ENEMY_FILL = (255, 90, 60), (70, 16, 10)
CORE_CELL = (GRID_N // 2, GRID_N // 2)


def cell_center(gx, gy):
    return (GRID_X + gx * CELL + CELL / 2, GRID_Y + gy * CELL + CELL / 2)


class Demo:
    def __init__(self):
        self.screen = pygame.display.set_mode((W, H))
        pygame.display.set_caption("CORE DEFENSE — ECAMS GAME LOOP PROTOTYPE")
        self.font = pygame.font.Font(FONT_MONO, 22)
        self.font_sm = pygame.font.Font(FONT_MONO, 16)
        self.clock = pygame.time.Clock()
        self.rng = random.Random(11)
        self.reset()

    # ---- state -----------------------------------------------------------
    def reset(self):
        self.grid = [[None] * GRID_N for _ in range(GRID_N)]
        self.stock = {"el": START_EL, "am": 0, "en": 0}
        self.core = {"type": "C", "hp": CORE_HP}
        cy, cx = CORE_CELL
        self.grid[cy][cx] = self.core
        self.enemies = []          # {fx, fy, hp}
        self.drones = []           # {item, sx, sy, tx, ty, t, dur}
        self.tracers = []          # {a, b, t}
        self.cursor = [GRID_N // 2, GRID_N // 2]
        self.sel = "T"
        self.time = 0.0
        self.spawn_timer = GRACE
        self.turrets_ready = {}    # (gx,gy) -> cooldown left
        self.prod = {}             # (gx,gy) -> timer
        self.warning = None
        self.warn_t = 0.0
        self.game_over = False
        self.killed = 0
        self.spawned = 0
        self.shots = 0
        self.spawn_on = False  # default off: familiarization/sandbox; G re-enables

    def struct_at(self, gx, gy):
        if 0 <= gx < GRID_N and 0 <= gy < GRID_N:
            return self.grid[gy][gx]
        return None

    # ---- player actions --------------------------------------------------
    def place(self, t, gx, gy):
        if self.game_over:
            return
        s = self.struct_at(gx, gy)
        if s is not None:
            self._warn("CELL OCCUPIED")
            return
        cost = STRUCTS[t][4]
        if any(self.stock[k] < v for k, v in cost.items()):
            self._warn("INSUFFICIENT STOCKPILE")
            return
        for k, v in cost.items():
            self.stock[k] -= v
        cell = {"type": t, "hp": STRUCT_HP, "buf": {"el": 0, "am": 0}}
        self.grid[gy][gx] = cell
        self.prod[(gx, gy)] = 0.0

    def _warn(self, msg):
        self.warning = msg
        self.warn_t = 1.0

    # ---- simulation ------------------------------------------------------
    def update(self, dt):
        if self.game_over:
            return
        self.time += dt
        self.warn_t = max(0.0, self.warn_t - dt)

        # enemy spawn (constant rate; G toggles it off for familiarization)
        if self.spawn_on:
            self.spawn_timer -= dt
            if self.spawn_timer <= 0:
                self.spawn_timer = SPAWN_EVERY
                side = self.rng.randrange(4)
                if side == 0:
                    gx, gy = self.rng.randrange(GRID_N), 0
                elif side == 1:
                    gx, gy = self.rng.randrange(GRID_N), GRID_N - 1
                elif side == 2:
                    gx, gy = 0, self.rng.randrange(GRID_N)
                else:
                    gx, gy = GRID_N - 1, self.rng.randrange(GRID_N)
                px, py = cell_center(gx, gy)
                self.enemies.append({"fx": (px - GRID_X) / CELL, "fy": (py - GRID_Y) / CELL, "hp": ENEMY_HP})
                self.spawned += 1

        # enemies: walk toward the core, chew whatever blocks them
        tcx, tcy = CORE_CELL
        for e in self.enemies:
            dx = tcx + 0.5 - e["fx"]
            dy = tcy + 0.5 - e["fy"]
            dist = math.hypot(dx, dy)
            if dist < 0.05:
                e["fx"], e["fy"] = tcx + 0.5, tcy + 0.5
                continue
            step = ENEMY_SPEED * dt / dist
            nx, ny = e["fx"] + dx * step, e["fy"] + dy * step
            s = self.struct_at(int(nx), int(ny))
            if s is not None:
                s["hp"] -= ENEMY_DPS * dt
                if s["hp"] <= 0:
                    gx, gy = int(nx), int(ny)
                    self.grid[gy][gx] = None
                    self.prod.pop((gx, gy), None)
                    self.turrets_ready.pop((gx, gy), None)
                    if s is self.core:
                        self.game_over = True
                        return
            else:
                e["fx"], e["fy"] = nx, ny
        self.enemies = [e for e in self.enemies if e["hp"] > 0]

        # energy: flat global production
        for gy in range(GRID_N):
            for gx in range(GRID_N):
                if self.grid[gy][gx] and self.grid[gy][gx]["type"] == "E":
                    self.stock["en"] += ENERGY_PER_S * dt

        # machines: production + drone requests
        for gy in range(GRID_N):
            for gx in range(GRID_N):
                c = self.grid[gy][gx]
                if c is None or c is self.core:
                    continue
                key = (gx, gy)
                buf = c["buf"]
                if c["type"] == "M":
                    self.prod[key] = self.prod.get(key, 0.0) + dt
                    if self.prod[key] >= ELEMENT_EVERY:
                        self.prod[key] -= ELEMENT_EVERY
                        buf["el"] = min(BUFFER, buf["el"] + 1)
                    if buf["el"] > 0 and not self._drone_pending(key, "el"):
                        self._request(key, "el", to_depot=True)
                elif c["type"] == "F":
                    self.prod[key] = self.prod.get(key, 0.0) + dt
                    if self.prod[key] >= FACTORY_EVERY and buf["el"] >= FACTORY_EL:
                        self.prod[key] -= FACTORY_EVERY
                        buf["el"] -= FACTORY_EL
                        buf["am"] = min(BUFFER, buf["am"] + FACTORY_AMMO)
                    if buf["am"] > 0 and not self._drone_pending(key, "am"):
                        self._request(key, "am", to_depot=True)
                    if buf["el"] < BUFFER and self.stock["el"] > 0 and not self._drone_pending(key, "el"):
                        self._request(key, "el", to_depot=False)
                elif c["type"] == "T":
                    if buf["am"] < BUFFER and self.stock["am"] > 0 and not self._drone_pending(key, "am"):
                        self._request(key, "am", to_depot=False)

        # drones
        for d in self.drones:
            d["t"] += dt
            if d["t"] >= d["dur"]:
                d["done"] = True
                gx, gy = d["gx"], d["gy"]
                c = self.struct_at(gx, gy)
                if d["to_depot"]:
                    self.stock[d["item"]] += 1
                    if c is not None:
                        c["buf"][d["item"]] = max(0, c["buf"][d["item"]] - 1)
                elif c is not None:
                    c["buf"][d["item"]] = min(BUFFER, c["buf"][d["item"]] + 1)
        self.drones = [d for d in self.drones if not d.get("done")]

        # turrets: nearest enemy in range, 1 shot/s, 1 ammo/shot
        for gy in range(GRID_N):
            for gx in range(GRID_N):
                c = self.grid[gy][gx]
                if c is None or c["type"] != "T":
                    continue
                key = (gx, gy)
                cd = self.turrets_ready.get(key, 0.0) - dt
                if cd > 0:
                    self.turrets_ready[key] = cd
                    continue
                if c["buf"]["am"] < TURRET_AMMO_COST:
                    self.turrets_ready[key] = 0.0
                    continue
                tx, ty = cell_center(gx, gy)
                best, bd = None, None
                for e in self.enemies:
                    ex = GRID_X + e["fx"] * CELL
                    ey = GRID_Y + e["fy"] * CELL
                    d = math.hypot(ex - tx, ey - ty) / CELL
                    if d <= TURRET_RANGE and (bd is None or d < bd):
                        best, bd = e, d
                if best is not None:
                    c["buf"]["am"] -= TURRET_AMMO_COST
                    best["hp"] -= TURRET_DMG
                    self.shots += 1
                    self.tracers.append({"a": (tx, ty), "b": (GRID_X + best["fx"] * CELL, GRID_Y + best["fy"] * CELL), "t": 0.12})
                    self.turrets_ready[key] = 1.0 / TURRET_SHOTS
        before = len(self.enemies)
        self.enemies = [e for e in self.enemies if e["hp"] > 0]
        self.killed += before - len(self.enemies)

        for t in self.tracers:
            t["t"] -= dt
        self.tracers = [t for t in self.tracers if t["t"] > 0]

    def _drone_pending(self, key, item=None):
        for d in self.drones:
            if d["gx"] == key[0] and d["gy"] == key[1] and (item is None or d["item"] == item):
                return True
        return False

    def _request(self, key, item, to_depot):
        gx, gy = key
        sx, sy = cell_center(gx, gy)
        if to_depot:
            tx, ty = self.depot_center()
        else:
            tx, ty = cell_center(gx, gy)
            sx, sy = self.depot_center()
        dist = math.hypot(tx - sx, ty - sy)
        self.drones.append({
            "item": item, "sx": sx, "sy": sy, "tx": tx, "ty": ty,
            "gx": gx, "gy": gy, "t": 0.0, "dur": dist / (DRONE_SPEED * CELL), "to_depot": to_depot,
        })

    def depot_center(self):
        return (GRID_X + GRID_PX + 60 + 100, GRID_Y + 42)

    # ---- rendering -------------------------------------------------------
    def draw(self):
        self.screen.fill(BG)
        self._header()
        self._stock_panel()
        self._grid()
        self._legend()
        self._status()
        if self.game_over:
            self._game_over()
        pygame.display.flip()

    def _header(self):
        self.screen.blit(self.font.render("CORE DEFENSE", True, TEXT), (GRID_X, 34))
        self.screen.blit(
            self.font_sm.render("ELECTRONIC CENTRALIZED MONITORING — GAME LOOP PROTOTYPE", True, TEXT_DIM),
            (GRID_X + 250, 42),
        )
        t = f"T {int(self.time // 60):02d}:{int(self.time % 60):02d}"
        self.screen.blit(self.font_sm.render(t, True, AMBER), (W - 180, 42))

    def _panel(self, x, y, w, h):
        pygame.draw.rect(self.screen, PANEL, (x, y, w, h))
        pygame.draw.rect(self.screen, BORDER, (x, y, w, h), 1)

    def _stock_panel(self):
        x, y = GRID_X + GRID_PX + 60, GRID_Y
        self._panel(x, y, 200, 96)
        self.screen.blit(self.font_sm.render("STOCKPILE", True, TEXT), (x + 12, y + 8))
        rows = [("EL", self.stock["el"], (90, 190, 230)), ("AM", self.stock["am"], AMBER), ("EN", self.stock["en"], (70, 210, 100))]
        for i, (k, v, col) in enumerate(rows):
            self.screen.blit(self.font_sm.render(f"{k} {int(v):04d}", True, col), (x + 12, y + 32 + i * 22))

    def _grid(self):
        self._panel(GRID_X - 12, GRID_Y - 12, GRID_PX + 24, GRID_PX + 24)
        for gy in range(GRID_N):
            for gx in range(GRID_N):
                r = pygame.Rect(GRID_X + gx * CELL + 2, GRID_Y + gy * CELL + 2, CELL - 4, CELL - 4)
                pygame.draw.rect(self.screen, (14, 20, 16), r, border_radius=4)
                c = self.grid[gy][gx]
                if c is not None:
                    letter, lc, fill, _, _, _ = STRUCTS[c["type"]]
                    pygame.draw.rect(self.screen, fill, r, border_radius=4)
                    img = self.font.render(letter, True, lc)
                    self.screen.blit(img, img.get_rect(center=r.center))
                    frac = max(0.0, c["hp"] / STRUCT_HP if c["type"] != "C" else c["hp"] / CORE_HP)
                    bar = pygame.Rect(r.x + 4, r.bottom - 8, r.w - 8, 4)
                    col = (70, 210, 100) if frac > 0.5 else AMBER if frac > 0.25 else RED
                    pygame.draw.rect(self.screen, col, (bar.x, bar.y, int(bar.w * frac), bar.h))
                    pygame.draw.rect(self.screen, (30, 40, 34), bar, 1)
        # enemies
        for e in self.enemies:
            x = GRID_X + e["fx"] * CELL
            y = GRID_Y + e["fy"] * CELL
            r = pygame.Rect(int(x) - 14, int(y) - 14, 28, 28)
            pygame.draw.rect(self.screen, ENEMY_FILL, r, border_radius=5)
            pygame.draw.rect(self.screen, ENEMY_LC, r, 1, border_radius=5)
            img = self.font_sm.render("X", True, ENEMY_LC)
            self.screen.blit(img, img.get_rect(center=r.center))
            pygame.draw.rect(self.screen, RED, (r.x, r.bottom + 2, int(24 * max(0.0, e["hp"] / ENEMY_HP)), 3))
        # tracers
        for t in self.tracers:
            pygame.draw.line(self.screen, AMBER, t["a"], t["b"], 1)
        # drones
        for d in self.drones:
            p = min(1.0, d["t"] / d["dur"])
            x = d["sx"] + (d["tx"] - d["sx"]) * p
            y = d["sy"] + (d["ty"] - d["sy"]) * p
            pts = [(x, y - 8), (x + 8, y), (x, y + 8), (x - 8, y)]
            pygame.draw.polygon(self.screen, TEXT_DIM, pts)
            img = self.font_sm.render("E" if d["item"] == "el" else "A", True, TEXT)
            self.screen.blit(img, img.get_rect(center=(int(x), int(y))))
        # cursor + placement ghost
        cx, cy = self.cursor
        r = pygame.Rect(GRID_X + cx * CELL + 2, GRID_Y + cy * CELL + 2, CELL - 4, CELL - 4)
        pygame.draw.rect(self.screen, WHITE, r, 2, border_radius=4)
        if not self.game_over and self.struct_at(cx, cy) is None:
            letter, lc, fill, _, _, _ = STRUCTS[self.sel]
            img = self.font.render(letter, True, lc)
            img.set_alpha(120)
            self.screen.blit(img, img.get_rect(center=r.center))

    def _legend(self):
        lx, ly = GRID_X + GRID_PX + 60, GRID_Y + 120
        self.screen.blit(self.font_sm.render("LEGEND", True, TEXT), (lx, ly - 24))
        items = [(*STRUCTS[t], ) for t in "CEMFTW"] + [("X", ENEMY_LC, ENEMY_FILL, "ENEMY")]
        for i, item in enumerate(items):
            y = ly + i * 26
            letter, lc, fill, name = item[:4]
            box = pygame.Rect(lx, y, 22, 22)
            pygame.draw.rect(self.screen, fill, box, border_radius=4)
            pygame.draw.rect(self.screen, BORDER, box, 1)
            img = self.font_sm.render(letter, True, lc)
            self.screen.blit(img, img.get_rect(center=box.center))
            self.screen.blit(self.font_sm.render(name, True, TEXT_DIM), (lx + 32, y + 3))

    def _status(self):
        bar = pygame.Rect(GRID_X - 12, H - 64, W - 2 * (GRID_X - 12), 44)
        pygame.draw.rect(self.screen, PANEL, bar)
        pygame.draw.rect(self.screen, BORDER, bar, 1)
        s = self.struct_at(*self.cursor)
        tile = "VOID" if s is None else STRUCTS[s["type"]][3]
        line = (
            f"CORE {int(self.core['hp'])}/{CORE_HP}   EL {int(self.stock['el']):03d}   AM {int(self.stock['am']):03d}   "
            f"EN {int(self.stock['en']):04d}   ENEMIES {len(self.enemies):02d}   KILLED {self.killed:02d}   "
            f"CURSOR {self.cursor[1]:02d},{self.cursor[0]:02d}   TILE {tile}   SPAWN {'ON' if self.spawn_on else 'OFF'}"
        )
        self.screen.blit(self.font_sm.render(line, True, TEXT), (bar.x + 14, bar.y + 4))
        _, lc, _, name, _, _ = STRUCTS[self.sel]
        hint = f"SELECT [{self.sel}] {name}   [1-5] SELECT   [ENTER] PLACE   [G] SPAWN {'OFF' if self.spawn_on else 'ON'}   [R] RESTART   [ESC] EXIT"
        self.screen.blit(self.font_sm.render(hint, True, lc), (bar.x + 14, bar.y + 24))
        if self.warning and self.warn_t > 0:
            self.screen.blit(self.font_sm.render(self.warning, True, RED), (bar.x + 420, bar.y + 24))

    def _game_over(self):
        veil = pygame.Surface((W, H), pygame.SRCALPHA)
        veil.fill((0, 0, 0, 200))
        self.screen.blit(veil, (0, 0))
        t = f"SURVIVED {int(self.time // 60):02d}:{int(self.time % 60):02d}"
        self.screen.blit(self.font.render("GAME OVER — CORE DESTROYED", True, RED), (W // 2 - 210, H // 2 - 30))
        self.screen.blit(self.font_sm.render(t, True, TEXT), (W // 2 - 70, H // 2 + 10))
        self.screen.blit(self.font_sm.render("[R] RESTART", True, TEXT_DIM), (W // 2 - 50, H // 2 + 40))

    # ---- loop ------------------------------------------------------------
    def run(self):
        running = True
        while running:
            dt = min(self.clock.tick(60) / 1000.0, 0.05)
            for e in pygame.event.get():
                if e.type == pygame.QUIT:
                    running = False
                elif e.type == pygame.KEYDOWN:
                    if e.key == pygame.K_ESCAPE:
                        running = False
                    elif e.key == pygame.K_r:
                        self.reset()
                    elif e.key == pygame.K_g:
                        self.spawn_on = not self.spawn_on
                    elif e.key in SELECT_KEYS:
                        self.sel = SELECT_KEYS[e.key]
                    elif e.key == pygame.K_RETURN:
                        self.place(self.sel, self.cursor[0], self.cursor[1])
                    elif e.key == pygame.K_LEFT:
                        self.cursor[0] = max(0, self.cursor[0] - 1)
                    elif e.key == pygame.K_RIGHT:
                        self.cursor[0] = min(GRID_N - 1, self.cursor[0] + 1)
                    elif e.key == pygame.K_UP:
                        self.cursor[1] = max(0, self.cursor[1] - 1)
                    elif e.key == pygame.K_DOWN:
                        self.cursor[1] = min(GRID_N - 1, self.cursor[1] + 1)
            self.update(dt)
            self.draw()
        pygame.quit()

    # ---- headless smoke test ----------------------------------------------
    def sim(self, seconds=30.0):
        # auto-build a starter base around the core, then let the loop run.
        # Turret needs 10 banked energy, so it goes in at t=3s after producers run.
        self.spawn_on = True  # the headless test exercises the enemy flow
        for t, gx, gy in [("E", 5, 2), ("E", 2, 5), ("M", 7, 2), ("M", 2, 7), ("F", 8, 5)]:
            self.place(t, gx, gy)
        frames = int(seconds / 0.016)
        for i in range(frames):
            self.update(0.016)
            if i == 180:  # ~3s
                self.place("T", 5, 8)
                self.place("W", 6, 6)
            self.draw()
        print("=== SIM SUMMARY (30s) ===")
        print(f"core hp    : {int(self.core['hp'])}/{CORE_HP}")
        print(f"stockpile  : el={self.stock['el']} am={self.stock['am']} en={int(self.stock['en'])}")
        print(f"enemies    : spawned={self.spawned} killed={self.killed} alive={len(self.enemies)}")
        print(f"drones     : active={len(self.drones)}")
        print(f"survived   : {self.time:.1f}s  game_over={self.game_over}")
        return 0


if __name__ == "__main__":
    d = Demo()
    if SIM:
        sys.exit(d.sim())
    d.run()
