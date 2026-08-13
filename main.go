// Command coredefense — minimal ebitengine window spike.
//
// Verifies the binding works on this machine (ADR 0001 consequence: the
// graphics binding is ebitengine). Real structure comes from the architecture
// tickets on "Chart the extensible game base".
package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenW = 640
	screenH = 360
)

// ebitenBG is the ECAMS near-black green background (#070B09).
var ebitenBG = color.RGBA{0x07, 0x0B, 0x09, 0xFF}

type Game struct{}

func (g *Game) Update() error { return nil }

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(ebitenBG)
}

func (g *Game) Layout(outsideW, outsideH int) (int, int) {
	return screenW, screenH
}

func main() {
	ebiten.SetWindowTitle("CORE DEFENSE — ebitengine base")
	ebiten.SetWindowSize(screenW, screenH)
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
