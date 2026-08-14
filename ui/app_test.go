package ui

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"coredefense/game"
)

// newTestApp builds an App without touching the window (NewApp configures
// ebiten; these tests only exercise cursor/camera math).
func newTestApp() *App {
	gs := game.NewGameState()
	return &App{gs: gs, curX: 8, curY: 8, cam: &Camera{Scale: 1}, sel: game.KindEnergyProducer}
}

func TestRegionIsTheDeFoggedChunk(t *testing.T) {
	a := newTestApp()
	minX, minY, maxX, maxY := a.region()
	if minX != 0 || minY != 0 || maxX != 15 || maxY != 15 {
		t.Fatalf("region = (%d,%d)-(%d,%d), want (0,0)-(15,15)", minX, minY, maxX, maxY)
	}
}

func TestMoveCursorClampsToRegion(t *testing.T) {
	a := newTestApp()
	// Move out of bounds in every direction; the cursor must stay inside.
	for i := 0; i < 30; i++ {
		a.moveCursor(ebiten.KeyArrowUp)
	}
	if a.curY != 0 {
		t.Fatalf("cursor left region at top: y=%d", a.curY)
	}
	for i := 0; i < 30; i++ {
		a.moveCursor(ebiten.KeyArrowDown)
	}
	if a.curY != 15 {
		t.Fatalf("cursor passed bottom edge: y=%d", a.curY)
	}
	for i := 0; i < 30; i++ {
		a.moveCursor(ebiten.KeyArrowRight)
	}
	if a.curX != 15 {
		t.Fatalf("cursor passed right edge: x=%d", a.curX)
	}
}

func TestCameraPanAndFollow(t *testing.T) {
	a := newTestApp()

	// Pan is continuous and free — the camera can leave the de-fogged region
	// (the infinite-map illusion; the region stays where it is).
	a.cam.Pan(40, -12.5)
	if a.cam.X != 40 || a.cam.Y != -12.5 {
		t.Fatalf("pan = (%v,%v), want (40,-12.5)", a.cam.X, a.cam.Y)
	}
	vx0, vy0, vx1, vy1 := a.cam.VisibleCells()
	if want := int64(math.Ceil(panelW / (CELL * 1.0))); vx0 != 40 || vy0 != -13 || vx1 != 40+want || vy1 != -13+want {
		t.Fatalf("visible = (%d,%d)-(%d,%d)", vx0, vy0, vx1, vy1)
	}

	// A cursor move while the camera is panned away snaps the view back to
	// the cursor (arrows = return to the action).
	a.moveCursor(ebiten.KeyArrowUp)
	if a.cam.X != 8 {
		t.Fatalf("cursor move didn't snap camera back: cam.X=%v", a.cam.X)
	}
}

func TestCameraFollowDoesNotFightVisibleCursor(t *testing.T) {
	a := newTestApp()
	// With the cursor in view, moving it pans gently — the camera follows the
	// cursor only at the view edges.
	a.cam.Pan(0.4, 0.3)
	a.moveCursor(ebiten.KeyArrowDown) // cursor 8,8 → 8,9, still in view
	if a.cam.X != 0.4 || a.cam.Y != 0.3 {
		t.Fatalf("follow fought a visible cursor: cam=(%v,%v)", a.cam.X, a.cam.Y)
	}

	// Follow in isolation: pans only when the cursor leaves the viewport.
	a.cam.Follow(20, 8)
	if want := float64(20 - int64(math.Ceil(panelW/(CELL*1.0))) + 1); a.cam.X != want {
		t.Fatalf("follow right: cam.X=%v, want %v", a.cam.X, want)
	}
	a.cam.Follow(-3, 8)
	if a.cam.X != -3 {
		t.Fatalf("follow left: cam.X=%v", a.cam.X)
	}
}

func TestCameraWorldMapping(t *testing.T) {
	cam := &Camera{X: 5, Y: 7, Scale: 1}
	// Cell (5,7) is the view's top-left → panel (0,0).
	x, y := cam.CellTopLeft(5, 7)
	if x != 0 || y != 0 {
		t.Fatalf("CellTopLeft(5,7) = (%v,%v), want (0,0)", x, y)
	}
	// The center of cell (8,8) sits CELL/2 inside that cell.
	sx, sy := cam.WorldToScreen(8.5, 8.5)
	if sx != 3.5*CELL || sy != 1.5*CELL {
		t.Fatalf("WorldToScreen(8.5,8.5) = (%v,%v), want (%v,%v)", sx, sy, 3.5*CELL, 1.5*CELL)
	}
}

// TestPanWiring guards the WASD pan path: Update's pan calls must actually
// move the camera (this caught the block silently missing from Update once).
// TestCursorRepeatSchedule pins the held-arrow auto-repeat cadence: quiet
// through the initial delay, then one move every repeatInterval frames.
func TestCursorRepeatSchedule(t *testing.T) {
	for d := 1; d <= repeatDelay; d++ {
		if cursorRepeat(d) {
			t.Fatalf("cursorRepeat(%d) fired during the hold delay", d)
		}
	}
	for d := repeatDelay + 1; d <= repeatDelay+3*repeatInterval; d++ {
		want := (d-repeatDelay)%repeatInterval == 0
		if got := cursorRepeat(d); got != want {
			t.Fatalf("cursorRepeat(%d) = %v, want %v", d, got, want)
		}
	}
}

// TestIsArrowKey pins that exactly the four cursor keys auto-repeat.
func TestIsArrowKey(t *testing.T) {
	arrows := map[ebiten.Key]bool{
		ebiten.KeyArrowUp: true, ebiten.KeyArrowDown: true,
		ebiten.KeyArrowLeft: true, ebiten.KeyArrowRight: true,
	}
	for _, k := range []ebiten.Key{ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyArrowRight, ebiten.KeyEnter, ebiten.KeyG, ebiten.KeyR, ebiten.KeyDigit1, ebiten.KeyMinus, ebiten.KeyW} {
		if got := isArrowKey(k); got != arrows[k] {
			t.Fatalf("isArrowKey(%v) = %v, want %v", k, got, arrows[k])
		}
	}
}

func TestPanWiring(t *testing.T) {
	a := newTestApp()
	pressed := func(k ebiten.Key) bool { return k == ebiten.KeyD }
	a.pan(1.0, pressed) // dt=1 → 12 cells right
	if a.cam.X != 12 {
		t.Fatalf("pan(D) didn't move camera: cam.X=%v", a.cam.X)
	}
	a.cam.Pan(-12, 0) // reset
	pressed = func(k ebiten.Key) bool { return k == ebiten.KeyW }
	a.pan(0.5, pressed) // dt=0.5 → 6 cells up
	if a.cam.Y != -6 {
		t.Fatalf("pan(W) didn't move camera: cam.Y=%v", a.cam.Y)
	}
}

func TestZoomLadder(t *testing.T) {
	a := newTestApp()
	a.cam.Zoom(1)
	if a.cam.Scale != 1.5 {
		t.Fatalf("zoom in: Scale=%v, want 1.5", a.cam.Scale)
	}
	a.cam.Zoom(-1)
	if a.cam.Scale != 1 {
		t.Fatalf("zoom out back: Scale=%v, want 1", a.cam.Scale)
	}
	// Clamp at both ends.
	for i := 0; i < 10; i++ {
		a.cam.Zoom(1)
	}
	if a.cam.Scale != 3 {
		t.Fatalf("zoom clamp top: Scale=%v, want 3", a.cam.Scale)
	}
	for i := 0; i < 10; i++ {
		a.cam.Zoom(-1)
	}
	if a.cam.Scale != 0.5 {
		t.Fatalf("zoom clamp bottom: Scale=%v, want 0.5", a.cam.Scale)
	}
}

func TestZoomKeepsPanelCenterFixed(t *testing.T) {
	cam := &Camera{X: 5, Y: 7, Scale: 1}
	cx, cy := float64(panelW)/2, float64(panelH)/2
	// World point under the panel center before zooming.
	wx := cam.X + cx/(CELL*cam.Scale)
	wy := cam.Y + cy/(CELL*cam.Scale)
	cam.ZoomTo(2)
	// ...must still sit at the panel center after.
	sx, sy := cam.WorldToScreen(wx, wy)
	if math.Abs(sx-cx) > 1e-6 || math.Abs(sy-cy) > 1e-6 {
		t.Fatalf("center drifted: (%v,%v), want (%v,%v)", sx, sy, cx, cy)
	}
}

func TestZoomVisibleAndFollow(t *testing.T) {
	a := newTestApp()
	a.cam.ZoomTo(3)
	vx0, _, vx1, _ := a.cam.VisibleCells()
	if want := int64(math.Ceil(panelW / (CELL * 3.0))); vx1-vx0 != want {
		t.Fatalf("visible cols at 3x: %d, want %d", vx1-vx0, want)
	}
	// Follow tracks the zoomed viewport width.
	a.cam.ZoomTo(2) // cols = ceil(560/68) = 9
	a.cam.Follow(20, 8)
	if a.cam.X != 20-9+1 {
		t.Fatalf("follow at 2x: cam.X=%v, want %d", a.cam.X, 20-9+1)
	}
}

func TestPanSpeedScalesWithZoom(t *testing.T) {
	a := newTestApp()
	pressD := func(k ebiten.Key) bool { return k == ebiten.KeyD }
	a.cam.ZoomTo(2) // 12/2 = 6 cells per second (ZoomTo also re-anchors X)
	before := a.cam.X
	a.pan(1.0, pressD)
	if a.cam.X-before != 6 {
		t.Fatalf("pan at 2x moved %v cells, want 6", a.cam.X-before)
	}
}
