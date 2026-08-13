package ui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"

	"coredefense/game"
)

// ECAMS palette (docs/art-style.md).
var (
	bg      = color.RGBA{0x07, 0x0B, 0x09, 0xFF}
	dimText = color.RGBA{0x5A, 0x82, 0x6E, 0xFF}
	bright  = color.RGBA{0x96, 0xD2, 0xB4, 0xFF}
)

// Grid origin in pixels, leaving room for the HUD line.
const (
	gridX = 10
	gridY = 20
)

// kindColors maps each structure kind to its (letter, plaque) colors, mirroring
// the prototype's legend. The full ECAMS look — rounded shapes, legend panel,
// stock/status panels, overlay — is the render layer ticket's work.
var kindColors = map[string][2]color.RGBA{
	game.KindCore:           {{0xFF, 0xFF, 0xFF, 0xFF}, {0x56, 0x1A, 0x16, 0xFF}},
	game.KindEnergyProducer: {{0x46, 0xD2, 0x64, 0xFF}, {0x12, 0x2A, 0x1C, 0xFF}},
	game.KindElementMachine: {{0x5A, 0xBE, 0xE6, 0xFF}, {0x10, 0x28, 0x32, 0xFF}},
	game.KindFactory:        {{0xB4, 0xD2, 0x5A, 0xFF}, {0x20, 0x2A, 0x0E, 0xFF}},
	game.KindTurret:         {{0xEB, 0x6E, 0x4B, 0xFF}, {0x2E, 0x18, 0x10, 0xFF}},
	game.KindWall:           {{0xAA, 0x8C, 0x5A, 0xFF}, {0x26, 0x20, 0x10, 0xFF}},
}

// drawStructures renders each placed structure as its kind letter on a plaque
// at its cell. Minimal proof that state renders; the render layer ticket owns
// the real look.
func drawStructures(screen *ebiten.Image, gs *game.GameState) {
	for _, p := range gs.AllStructures() {
		cols := kindColors[p.S.Kind]
		if cols == [2]color.RGBA{} {
			cols = [2]color.RGBA{bright, bg}
		}
		x, y := gridX+int(p.X)*CELL, gridY+int(p.Y)*CELL
		rect := screen.SubImage(image.Rect(x+2, y+2, x+CELL-2, y+CELL-2)).(*ebiten.Image)
		rect.Fill(cols[1])
		text.Draw(screen, p.S.Kind, basicfont.Face7x13, x+18, y+26, cols[0])
	}
}

// drawHUD draws the live readout line (ECAMS style).
func drawHUD(screen *ebiten.Image, gs *game.GameState) {
	line := fmt.Sprintf("T+%.1fs  EL %d  EN %d  AM %d  spawn %v  killed %d",
		gs.Time, gs.Stockpile.El, gs.Stockpile.En, gs.Stockpile.Am, gs.Spawn.On, gs.Metrics.Killed)
	text.Draw(screen, line, basicfont.Face7x13, 6, 10, bright)
	text.Draw(screen, "[G] spawn  [ESC] quit", basicfont.Face7x13, ScreenW-130, ScreenH-6, dimText)
}
