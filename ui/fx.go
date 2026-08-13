package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

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

// Draw renders live effects. Tracer/warning visuals land with the render
// layer ticket; for now the layer is plumbing.
func (f *Fx) Draw(screen *ebiten.Image) {}
