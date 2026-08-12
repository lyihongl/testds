"""PROTOTYPE - throwaway. ECAMS-terminal look for the sub-factory prototype.

Answers: does the "operating a complex industrial machine" feel work for the
grid? Letters in basic shapes + color legend + live status readout.

Run: .venv/bin/python prototype_ecams.py
Controls: ARROWS move cursor | E/ENTER descend into a SUB-FACTORY tile | ESC ascend
"""

import random
import sys

import pygame

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

FONT_MONO = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"

# letter -> (shape fill, letter color, name)   shapes are rounded rects
ENTITIES = {
    None: ((22, 30, 26), (40, 60, 50), "VOID"),
    "E": ((18, 42, 28), (70, 210, 100), "ENERGY PROD"),
    "M": ((16, 40, 50), (90, 190, 230), "ELEMENT MACH"),
    "A": ((48, 40, 14), (235, 190, 70), "AMMO"),
    "W": ((46, 24, 16), (235, 110, 75), "WEAPON"),
    "S": ((42, 24, 48), (215, 130, 235), "SUB-FACTORY"),
}


def make_grid(depth, rng):
    """One level of the world: mostly empty, some machines, a sub-factory or two."""
    g = [[None] * GRID_N for _ in range(GRID_N)]
    for _ in range(4 + depth):
        x, y = rng.randrange(GRID_N), rng.randrange(GRID_N)
        g[y][x] = rng.choice(["E", "M", "A", "W"])
    if depth < 3:  # keep the recursion visible but not overwhelming
        for _ in range(2):
            x, y = rng.randrange(GRID_N), rng.randrange(GRID_N)
            g[y][x] = "S"
    g[GRID_N // 2][GRID_N // 2] = "E"  # the home energy producer always exists
    return g


class Demo:
    def __init__(self):
        self.screen = pygame.display.set_mode((W, H))
        pygame.display.set_caption("FACTORY CONTROL — ECAMS PROTOTYPE")
        self.font = pygame.font.Font(FONT_MONO, 22)
        self.font_sm = pygame.font.Font(FONT_MONO, 16)
        self.rng = random.Random(7)
        self.stack = [make_grid(0, self.rng)]  # path of grids, index = depth
        self.cursor = [GRID_N // 2, GRID_N // 2]

    # ---- state -----------------------------------------------------------
    @property
    def depth(self):
        return len(self.stack) - 1

    @property
    def grid(self):
        return self.stack[-1]

    def tile(self):
        x, y = self.cursor
        return self.grid[y][x]

    # ---- actions ---------------------------------------------------------
    def enter_sub_factory(self):
        if self.tile() == "S":
            self.stack.append(make_grid(self.depth + 1, self.rng))
            self.cursor = [GRID_N // 2, GRID_N // 2]

    def exit_sub_factory(self):
        if self.depth > 0:
            self.stack.pop()
            self.cursor = [GRID_N // 2, GRID_N // 2]

    # ---- rendering -------------------------------------------------------
    def draw(self):
        self.screen.fill(BG)
        self._header()
        self._grid()
        self._legend()
        self._status()
        pygame.display.flip()

    def _header(self):
        self.screen.blit(self.font.render("FACTORY CONTROL", True, TEXT), (GRID_X, 34))
        self.screen.blit(
            self.font_sm.render("ELECTRONIC CENTRALIZED MONITORING — PROTOTYPE", True, TEXT_DIM),
            (GRID_X + 330, 42),
        )
        self.screen.blit(self.font_sm.render(f"DEPTH {self.depth}", True, AMBER), (W - 220, 42))

    def _cell_rect(self, gx, gy):
        return pygame.Rect(GRID_X + gx * CELL + 2, GRID_Y + gy * CELL + 2, CELL - 4, CELL - 4)

    def _grid(self):
        # level panel
        panel = pygame.Rect(GRID_X - 12, GRID_Y - 12, GRID_PX + 24, GRID_PX + 24)
        pygame.draw.rect(self.screen, PANEL, panel)
        pygame.draw.rect(self.screen, BORDER, panel, 1)

        for gy in range(GRID_N):
            for gx in range(GRID_N):
                fill, letter_color, _ = ENTITIES[self.grid[gy][gx]]
                r = self._cell_rect(gx, gy)
                pygame.draw.rect(self.screen, fill, r, border_radius=4)
                letter = ENTITIES[self.grid[gy][gx]][1] if self.grid[gy][gx] else None
                if self.grid[gy][gx]:
                    img = self.font.render(self.grid[gy][gx], True, letter_color)
                    self.screen.blit(img, img.get_rect(center=r.center))
        # cursor
        cx, cy = self.cursor
        pygame.draw.rect(self.screen, (255, 255, 255), self._cell_rect(cx, cy), 2, border_radius=4)

    def _legend(self):
        lx, ly = GRID_X + GRID_PX + 60, GRID_Y
        self.screen.blit(self.font_sm.render("LEGEND", True, TEXT), (lx, ly - 30))
        for i, (letter, (fill, lc, name)) in enumerate(ENTITIES.items()):
            y = ly + i * 34
            label = "·" if letter is None else letter
            box = pygame.Rect(lx, y, 26, 26)
            pygame.draw.rect(self.screen, fill, box, border_radius=4)
            pygame.draw.rect(self.screen, BORDER, box, 1)
            if letter:
                img = self.font_sm.render(label, True, lc)
                self.screen.blit(img, img.get_rect(center=box.center))
            self.screen.blit(self.font_sm.render(name, True, TEXT_DIM), (lx + 40, y + 5))

    def _status(self):
        bar = pygame.Rect(GRID_X - 12, H - 64, W - 2 * (GRID_X - 12), 44)
        pygame.draw.rect(self.screen, PANEL, bar)
        pygame.draw.rect(self.screen, BORDER, bar, 1)
        _, _, tname = ENTITIES[self.tile()]
        tile_str = f"{tname} <{self.tile()}>" if self.tile() else "VOID"
        line = (
            f"SYS NOMINAL   DEPTH {self.depth}   CURSOR {self.cursor[1]:02d},{self.cursor[0]:02d}   "
            f"TILE {tile_str}"
        )
        hint = "[E/ENTER] ENTER S   [ESC] EXIT" if self.tile() == "S" else "[E/ENTER] ENTER SUB-FACTORY   [ESC] EXIT"
        self.screen.blit(self.font_sm.render(line, True, TEXT), (bar.x + 14, bar.y + 6))
        self.screen.blit(self.font_sm.render(hint, True, TEXT_DIM), (bar.x + 14, bar.y + 24))

    # ---- loop ------------------------------------------------------------
    def run(self):
        clock = pygame.time.Clock()
        running = True
        while running:
            for e in pygame.event.get():
                if e.type == pygame.QUIT:
                    running = False
                elif e.type == pygame.KEYDOWN:
                    if e.key in (pygame.K_ESCAPE, pygame.K_BACKSPACE):
                        self.exit_sub_factory()
                    elif e.key in (pygame.K_e, pygame.K_RETURN):
                        self.enter_sub_factory()
                    elif e.key == pygame.K_LEFT:
                        self.cursor[0] = max(0, self.cursor[0] - 1)
                    elif e.key == pygame.K_RIGHT:
                        self.cursor[0] = min(GRID_N - 1, self.cursor[0] + 1)
                    elif e.key == pygame.K_UP:
                        self.cursor[1] = max(0, self.cursor[1] - 1)
                    elif e.key == pygame.K_DOWN:
                        self.cursor[1] = min(GRID_N - 1, self.cursor[1] + 1)
            self.draw()
            clock.tick(60)
        pygame.quit()


if __name__ == "__main__":
    Demo().run()
