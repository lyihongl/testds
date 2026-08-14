package ui

import "math"

// Camera is the world→screen offset applied to every world-space element
// (grid cells, structures, enemies, drones, tracers) — the "infinite map"
// illusion (ticket #21): the world is an infinite grid of procedurally drawn
// cells, and the camera just decides which part of it lands on the panel.
//
// Ebiten has no Camera type: its transform primitive is the per-DrawImage
// GeoM, which vector/text draws don't take. So the camera is an explicit
// offset — screen = (world − cam) × CELL × Scale — applied when mapping
// world coordinates (continuous cell units; X.5 is the center of cell X) to
// panel pixels.
//
// The camera is continuous (fractional cell units) so panning is smooth, and
// the view is a window of (panelW / (CELL·Scale)) cells (ticket #22 zoom).
// It is presentation-only: never serialized, never touches the sim, and the
// fog region (gameplay) stays as-is — panning far out shows only void cells.
type Camera struct {
	X, Y  float64 // top-left of the view, in cell units
	Scale float64 // zoom factor: 1.0 = CELL pixels per cell
}

// zoomSteps is the discrete zoom ladder. Stepped rather than continuous
// keeps ebiten's per-size text glyph caches bounded and fits the ECAMS
// terminal's discrete feel.
var zoomSteps = []float64{0.5, 0.75, 1.0, 1.5, 2.0, 3.0}

// CellPx returns the current on-screen size of one cell in pixels.
func (c *Camera) CellPx() float64 { return CELL * c.Scale }

// Pan shifts the camera by dx, dy cell units (WASD panning).
func (c *Camera) Pan(dx, dy float64) {
	c.X += dx
	c.Y += dy
}

// Zoom steps the zoom ladder by dir (+1 in, −1 out), keeping the panel
// center's world point fixed under the view's center.
func (c *Camera) Zoom(dir int) {
	i := 0
	best := math.Abs(c.Scale - zoomSteps[0])
	for j, s := range zoomSteps {
		if d := math.Abs(c.Scale - s); d < best {
			best, i = d, j
		}
	}
	i += dir
	if i < 0 {
		i = 0
	}
	if i >= len(zoomSteps) {
		i = len(zoomSteps) - 1
	}
	c.ZoomTo(zoomSteps[i])
}

// ZoomTo sets the zoom to scale, keeping the panel center's world point
// fixed: the world point under the view's center before stays there after.
func (c *Camera) ZoomTo(scale float64) {
	if c.Scale <= 0 {
		c.Scale = 1
	}
	cx, cy := panelW/2.0, panelH/2.0
	wcx := c.X + cx/(CELL*c.Scale)
	wcy := c.Y + cy/(CELL*c.Scale)
	c.Scale = scale
	c.X = wcx - cx/(CELL*scale)
	c.Y = wcy - cy/(CELL*scale)
}

// WorldToScreen maps a world position (continuous cell units) to panel
// pixels (the grid panel's top-left is the origin), at the current zoom.
func (c *Camera) WorldToScreen(wx, wy float64) (float64, float64) {
	return (wx - c.X) * CELL * c.Scale, (wy - c.Y) * CELL * c.Scale
}

// CellTopLeft maps a grid cell to the panel pixel of its top-left corner.
func (c *Camera) CellTopLeft(gx, gy int64) (float64, float64) {
	return c.WorldToScreen(float64(gx), float64(gy))
}

// viewSize reports how many cells wide and tall the view shows (fractional).
func (c *Camera) viewSize() (float64, float64) {
	return panelW / (CELL * c.Scale), panelH / (CELL * c.Scale)
}

// VisibleCells returns the inclusive cell range the view shows. The far edge
// includes one extra row/column of (partially visible) cells; the grid panel
// clips them.
func (c *Camera) VisibleCells() (minX, minY, maxX, maxY int64) {
	x0, y0 := int64(math.Floor(c.X)), int64(math.Floor(c.Y))
	cols, rows := c.viewSize()
	return x0, y0, x0 + int64(math.Ceil(cols)), y0 + int64(math.Ceil(rows))
}

// Follow pans the camera only when the cursor leaves the viewport, so manual
// panning is never fought while the cursor stays visible. The viewport size
// tracks the current zoom.
func (c *Camera) Follow(curX, curY int64) {
	cols, rows := c.viewSize()
	colsInt, rowsInt := int64(math.Ceil(cols)), int64(math.Ceil(rows))
	if curX < int64(math.Floor(c.X)) {
		c.X = float64(curX)
	}
	if curX >= int64(math.Floor(c.X))+colsInt {
		c.X = float64(curX - colsInt + 1)
	}
	if curY < int64(math.Floor(c.Y)) {
		c.Y = float64(curY)
	}
	if curY >= int64(math.Floor(c.Y))+rowsInt {
		c.Y = float64(curY - rowsInt + 1)
	}
}
