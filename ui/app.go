// Package ui is the ebitengine presentation layer: window, render, input, and
// audio (ADR 0003). It imports game and reads GameState read-only; it never
// mutates the sim directly — input travels as game.Intent, cosmetics arrive
// as game.Event.
package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"coredefense/game"
)

// Logical screen size (1280x720 at 2x of the original 640x360 — the terminal
// is readable on modern displays). CELL lives in render.go with the layout.
const (
	ScreenW = 1280
	ScreenH = 720
)

// App implements ebiten.Game: it owns the GameState reference and the fx
// layer, and drives one SimulationStep per frame (ADR 0003 contract).
//
// Presentation-only state (never serialized, never touches the sim): the
// cursor cell, the camera (top-left visible cell), and the selected build
// kind. Input travels to the sim as Intent.
type App struct {
	gs   *game.GameState
	fx   *Fx
	prev map[ebiten.Key]bool // key states from the previous frame (edge detection)

	curX, curY int64   // cursor cell
	cam        *Camera // world→panel offset; pans freely over the infinite grid
	sel        string  // selected structure kind (keys 1-5)

	// worldBuf is the offscreen the world renders into each frame (grid,
	// entities, tracers), then blits at the panel rect. Drawing world space
	// into a screen SubImage is unreliable (ebiten's vector fills ignore the
	// subimage offset), so the world gets its own unambiguous 0,0 surface.
	worldBuf *ebiten.Image
}

// selectKeys maps the digit keys to build kinds, mirroring the game loop's
// controls (prototype shape): 1 E, 2 M, 3 F, 4 T, 5 W.
var selectKeys = map[ebiten.Key]string{
	ebiten.KeyDigit1: game.KindEnergyProducer,
	ebiten.KeyDigit2: game.KindElementMachine,
	ebiten.KeyDigit3: game.KindFactory,
	ebiten.KeyDigit4: game.KindTurret,
	ebiten.KeyDigit5: game.KindWall,
}

// Debug hooks (ADR 0004): installDebug is a no-op in production builds and is
// replaced by the debug-tagged panel; debugUpdate/debugDraw stay nil unless
// the panel installed them. The base App never references debug types.
var (
	installDebug = func(gs *game.GameState) {}
	debugUpdate  func() error
	debugDraw    func(screen *ebiten.Image)
)

// NewApp builds the application and configures the window. The cursor starts
// on the core's cell and the camera is centered on it (the demo world's
// middle); WASD pans the view freely from there.
func NewApp(gs *game.GameState) *App {
	ebiten.SetWindowTitle("CORE DEFENSE")
	ebiten.SetWindowSize(ScreenW, ScreenH)
	ebiten.SetWindowResizable(true)
	cx, cy := gs.CoreCell()
	a := &App{
		gs:   gs,
		fx:   NewFx(),
		prev: make(map[ebiten.Key]bool),
		curX: cx, curY: cy,
		cam:      &Camera{X: float64(cx) - viewCols/2, Y: float64(cy) - viewRows/2, Scale: 1},
		sel:      game.KindEnergyProducer,
		worldBuf: ebiten.NewImage(panelW, panelH),
	}
	installDebug(gs)
	return a
}

// region returns the de-fogged region's cell bounds (inclusive).
func (a *App) region() (minX, minY, maxX, maxY int64) {
	return regionBounds(a.gs)
}

// Update reads key edges, moves the cursor/camera, steps the sim once, and
// feeds its events to fx. dt is fixed at 1/60 (frame-rate independent
// systems; the uncapped-dt tunneling caveat is ticket #20).
func (a *App) Update() error {
	const dt = 1.0 / 60.0

	var in game.Intent
	for _, k := range []ebiten.Key{
		ebiten.KeyG, ebiten.KeyR, ebiten.KeyEnter,
		ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyArrowRight,
		ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5,
		ebiten.KeyEqual, ebiten.KeyNumpadAdd,
		ebiten.KeyMinus,
	} {
		pressed := ebiten.IsKeyPressed(k)
		if pressed && !a.prev[k] { // rising edge: pressed this frame
			switch k {
			case ebiten.KeyG:
				in.ToggleSpawn = true
			case ebiten.KeyR:
				in.Restart = true
			case ebiten.KeyEnter:
				in.Place = true
				in.Kind = a.sel
				in.CellX, in.CellY = a.curX, a.curY
			case ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyArrowRight:
				a.moveCursor(k)
			case ebiten.KeyEqual, ebiten.KeyNumpadAdd:
				a.cam.Zoom(1)
			case ebiten.KeyMinus:
				a.cam.Zoom(-1)
			default:
				if kind, ok := selectKeys[k]; ok {
					a.sel = kind
				}
			}
		}
		a.prev[k] = pressed
	}

	// WASD pans the camera freely over the infinite grid (held keys, no edge
	// detection). The cursor and its placement stay in the de-fogged region.
	a.pan(dt, ebiten.IsKeyPressed)

	// Mouse wheel zooms (one ladder step per tick).
	if _, wy := ebiten.Wheel(); wy != 0 {
		if wy > 0 {
			a.cam.Zoom(1)
		} else {
			a.cam.Zoom(-1)
		}
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

// pan pans the camera by WASD held-state. pressed is a key-state function
// (ebiten.IsKeyPressed in Update; a fake in tests), which keeps the wiring
// one line and the pan math testable.
func (a *App) pan(dt float64, pressed func(ebiten.Key) bool) {
	const panSpeed = 12.0 // cells per second at 1x zoom
	dt *= 1 / a.cam.Scale // zoomed in = slower pan (constant screen speed)
	if pressed(ebiten.KeyW) {
		a.cam.Pan(0, -panSpeed*dt)
	}
	if pressed(ebiten.KeyS) {
		a.cam.Pan(0, panSpeed*dt)
	}
	if pressed(ebiten.KeyA) {
		a.cam.Pan(-panSpeed*dt, 0)
	}
	if pressed(ebiten.KeyD) {
		a.cam.Pan(panSpeed*dt, 0)
	}
}

// moveCursor moves the cursor within the de-fogged region and pans the camera
// to keep the cursor in view. Cursor-follow engages only when the cursor
// moves, so manual WASD panning is never fought until the player acts again.
func (a *App) moveCursor(k ebiten.Key) {
	minX, minY, maxX, maxY := a.region()
	switch k {
	case ebiten.KeyArrowUp:
		if a.curY > minY {
			a.curY--
		}
	case ebiten.KeyArrowDown:
		if a.curY < maxY {
			a.curY++
		}
	case ebiten.KeyArrowLeft:
		if a.curX > minX {
			a.curX--
		}
	case ebiten.KeyArrowRight:
		if a.curX < maxX {
			a.curX++
		}
	}
	a.cam.Follow(a.curX, a.curY)
}

// Layout reports the fixed logical size.
func (a *App) Layout(outsideW, outsideH int) (int, int) {
	return ScreenW, ScreenH
}
