package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"coredefense/game"
)

// fxItem is a live cosmetic effect with its remaining lifetime.
type fxItem struct {
	ev   game.Event
	life float64
}

// Fx is the particle-system-style layer that owns cosmetic events (ADR 0002):
// the sim emits events, this layer animates them, render draws them. Events
// are transient — never state, never serialized.
type Fx struct {
	items []fxItem
}

func NewFx() *Fx { return &Fx{} }

// Add ingests the events a sim step produced.
func (f *Fx) Add(events []game.Event) {
	for _, ev := range events {
		life := ev.Dur
		if life <= 0 {
			life = 0.12
		}
		f.items = append(f.items, fxItem{ev: ev, life: life})
	}
}

// Update ages the effects and drops expired ones.
func (f *Fx) Update(dt float64) {
	alive := f.items[:0]
	for _, it := range f.items {
		it.life -= dt
		if it.life > 0 {
			alive = append(alive, it)
		}
	}
	f.items = alive
}

// Draw renders live effects: tracers as fading amber shot lines (world-space,
// drawn into the world buffer after drawGrid blits it, then blitted again —
// they ride the camera), warnings as fading red lines at the panel's top
// edge (screen-space chrome, drawn on screen directly).
func (f *Fx) Draw(screen, buf *ebiten.Image, cam *Camera) {
	var warns []fxItem
	for _, it := range f.items {
		switch it.ev.Kind {
		case game.EventTracer:
			x0, y0 := cam.WorldToScreen(it.ev.A.X, it.ev.A.Y)
			x1, y1 := cam.WorldToScreen(it.ev.B.X, it.ev.B.Y)
			alpha := uint8(255 * min(1, it.life/0.12))
			c := color.RGBA{amber.R, amber.G, amber.B, alpha}
			vector.StrokeLine(buf, float32(x0), float32(y0), float32(x1), float32(y1), float32(3*cam.Scale), c, true)
		case game.EventWarning:
			warns = append(warns, it)
		}
	}
	// Tracers landed in buf after its first blit; blit again to show them.
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(panelX, panelY)
	screen.DrawImage(buf, opts)
	// Warnings stack from the grid panel's top edge, red, fading with life.
	for i, it := range warns {
		alpha := uint8(255 * min(1, it.life/0.5))
		c := color.RGBA{red.R, red.G, red.B, alpha}
		drawTextCentered(screen, it.ev.Text, face(24), float64(gridX+viewCols*CELL/2), float64(gridY-16+i*32), 32, c)
	}
}
