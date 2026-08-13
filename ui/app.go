// Package ui is the ebitengine presentation layer: window, render, input, and
// audio (ADR 0003). It imports game and reads GameState read-only; it never
// mutates the sim directly — input travels as game.Intent, cosmetics arrive
// as game.Event.
package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"coredefense/game"
)

// Logical screen size and presentation cell size in pixels.
const (
	ScreenW = 640
	ScreenH = 360
	CELL    = 44
)

// App implements ebiten.Game: it owns the GameState reference and the fx
// layer, and drives one SimulationStep per frame (ADR 0003 contract).
type App struct {
	gs   *game.GameState
	fx   *Fx
	prev map[ebiten.Key]bool // key states from the previous frame (edge detection)
}

// Debug hooks (ADR 0004): installDebug is a no-op in production builds and is
// replaced by the debug-tagged panel; debugUpdate/debugDraw stay nil unless
// the panel installed them. The base App never references debug types.
var (
	installDebug = func(gs *game.GameState) {}
	debugUpdate  func() error
	debugDraw    func(screen *ebiten.Image)
)

// NewApp builds the application and configures the window.
func NewApp(gs *game.GameState) *App {
	ebiten.SetWindowTitle("CORE DEFENSE")
	ebiten.SetWindowSize(ScreenW, ScreenH)
	a := &App{gs: gs, fx: NewFx(), prev: make(map[ebiten.Key]bool)}
	installDebug(gs)
	return a
}

// Update reads key edges, steps the sim once, and feeds its events to fx.
// A fixed 1/60 dt for the skeleton; the sim tickets own the real timestep.
func (a *App) Update() error {
	const dt = 1.0 / 60.0

	var in game.Intent
	for _, k := range []ebiten.Key{ebiten.KeyEscape, ebiten.KeyG} {
		pressed := ebiten.IsKeyPressed(k)
		if pressed && !a.prev[k] { // rising edge: pressed this frame
			switch k {
			case ebiten.KeyEscape:
				return ebiten.Termination
			case ebiten.KeyG:
				in.ToggleSpawn = true
			}
		}
		a.prev[k] = pressed
	}

	a.fx.Add(game.SimulationStep(a.gs, in, dt))
	a.fx.Update(dt)
	if debugUpdate != nil {
		if err := debugUpdate(); err != nil {
			return err
		}
	}
	return nil
}

// Draw renders the world, the fx layer, and (in debug builds) the tuning
// panel on top.
func (a *App) Draw(screen *ebiten.Image) {
	screen.Fill(bg)
	drawStructures(screen, a.gs)
	drawHUD(screen, a.gs)
	a.fx.Draw(screen)
	if debugDraw != nil {
		debugDraw(screen)
	}
}

// Layout reports the fixed logical size.
func (a *App) Layout(outsideW, outsideH int) (int, int) {
	return ScreenW, ScreenH
}
