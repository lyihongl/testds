package ui

// The ECAMS render layer (ticket #18): GameState drawn as a dark monitoring
// terminal (docs/art-style.md). Structures are lettered rounded plaques with
// a legend; enemies and drones are blips; the world is the de-fogged region,
// rendered chunk-agnostically — the camera is a cell offset, so multi-chunk
// regions just scroll (the sim's own invariant, mirrored here).
//
// Layout (1280x720 = 2x the original 640x360): header / grid panel (16x16
// cells = one chunk's worth of view) / right column (stock, tile, legend,
// select) / bottom status bar.

import (
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"

	"coredefense/game"
)

// ---- palette (docs/art-style.md) ----

var (
	bgColor     = color.RGBA{0x07, 0x0B, 0x09, 0xFF} // near-black green terminal
	panelColor  = color.RGBA{0x0E, 0x16, 0x12, 0xFF}
	borderColor = color.RGBA{0x2D, 0x50, 0x3C, 0xFF}
	dimText     = color.RGBA{0x5A, 0x82, 0x6E, 0xFF}
	bright      = color.RGBA{0x96, 0xD2, 0xB4, 0xFF}
	amber       = color.RGBA{0xEB, 0xBE, 0x46, 0xFF}
	red         = color.RGBA{0xEB, 0x5F, 0x41, 0xFF}
	voidColor   = color.RGBA{0x16, 0x1E, 0x1A, 0xFF} // outside the de-fogged region
	emptyCell   = color.RGBA{0x1E, 0x28, 0x22, 0xFF} // de-fogged but empty
	cursorCol   = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	enemyFill   = color.RGBA{0x2E, 0x18, 0x10, 0xFF}
	hpGreen     = color.RGBA{0x46, 0xD2, 0x64, 0xFF}
	hpAmber     = amber
	hpRed       = red
)

// ---- layout (1280x720, all metrics at 2x) ----

const (
	gridX    = 16 // top-left of the first visible cell
	gridY    = 60 // the grid panel's top edge
	CELL     = 34
	viewCols = 16 // cells of view (one chunk's worth)
	viewRows = 16

	// The grid panel's screen rect. World space draws into it (clipped);
	// UI chrome (header, right column, status bar) draws on screen directly.
	panelX = gridX - 8
	panelY = gridY - 8
	panelW = CELL*viewCols + 16 // grid panel size
	panelH = CELL*viewRows + 16

	rightX = gridX + CELL*viewCols + 24 // right column's left edge
	barY   = 636                        // bottom status bar
	barH   = 80
)

// ---- fonts ----

var (
	monoSource *textv2.GoTextFaceSource
	basicFace  = textv2.NewGoXFace(basicfont.Face7x13)
)

func init() {
	// DejaVu Sans Mono is the documented terminal font (docs/art-style.md).
	// Fall back to the bitmap basicfont if it is not installed.
	for _, p := range []string{
		"/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
		"/usr/share/fonts/TTF/DejaVuSansMono.ttf",
		"/usr/share/fonts/dejavu/DejaVuSansMono.ttf",
	} {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		src, err := textv2.NewGoTextFaceSource(f)
		f.Close()
		if err == nil {
			monoSource = src
			break
		}
	}
}

// face returns a mono face at size px. Faces are cached by size: ebiten
// batches text through the face's source glyph cache, and reusing the face
// keeps per-frame text rendering allocation-free (sizes are few — the zoom
// ladder bounds them).
var faceCache = map[float64]textv2.Face{}

func face(size float64) textv2.Face {
	if monoSource == nil {
		return basicFace
	}
	if f, ok := faceCache[size]; ok {
		return f
	}
	f := &textv2.GoTextFace{Source: monoSource, Size: size}
	faceCache[size] = f
	return f
}

// ---- text helpers ----

func drawText(screen *ebiten.Image, s string, f textv2.Face, x, y float64, c color.Color) {
	opts := &textv2.DrawOptions{}
	opts.ColorScale.ScaleWithColor(c)
	opts.GeoM.Translate(x, y)
	textv2.Draw(screen, s, f, opts)
}

func drawTextRight(screen *ebiten.Image, s string, f textv2.Face, right, y float64, c color.Color) {
	w, _ := textv2.Measure(s, f, 20)
	drawText(screen, s, f, right-w, y, c)
}

// drawTextCentered draws s centered on (cx, cy) with the given line height.
// Centering is manual (AlignStart + measured width); ebiten's AlignCenter
// shifts by half the advance internally, which would double the offset.
func drawTextCentered(screen *ebiten.Image, s string, f textv2.Face, cx, cy, lineH float64, c color.Color) {
	w, _ := textv2.Measure(s, f, lineH)
	opts := &textv2.DrawOptions{}
	opts.ColorScale.ScaleWithColor(c)
	opts.GeoM.Translate(cx-w/2, cy-lineH/2)
	opts.LineSpacing = lineH
	textv2.Draw(screen, s, f, opts)
}

// ---- shapes ----

func roundedRectPath(x, y, w, h, r float32) *vector.Path {
	p := &vector.Path{}
	p.MoveTo(x+r, y)
	p.ArcTo(x+w, y, x+w, y+r, r) // top-right corner
	p.LineTo(x+w, y+h-r)
	p.ArcTo(x+w, y+h, x+w-r, y+h, r) // bottom-right corner
	p.LineTo(x+r, y+h)
	p.ArcTo(x, y+h, x, y+h-r, r) // bottom-left corner
	p.LineTo(x, y+r)
	p.ArcTo(x, y, x+r, y, r) // top-left corner
	p.Close()
	return p
}

func fillRounded(screen *ebiten.Image, x, y, w, h, r float32, c color.Color) {
	opts := &vector.DrawPathOptions{}
	opts.ColorScale.ScaleWithColor(c)
	vector.FillPath(screen, roundedRectPath(x, y, w, h, r), &vector.FillOptions{}, opts)
}

func strokeRounded(screen *ebiten.Image, x, y, w, h, r, width float32, c color.Color) {
	opts := &vector.DrawPathOptions{}
	opts.ColorScale.ScaleWithColor(c)
	vector.StrokePath(screen, roundedRectPath(x, y, w, h, r), &vector.StrokeOptions{Width: width}, opts)
}

func fillRect(screen *ebiten.Image, x, y, w, h float32, c color.Color) {
	vector.DrawFilledRect(screen, x, y, w, h, c, false)
}

// panel draws a standard panel: dark fill, thin border.
func panel(screen *ebiten.Image, x, y, w, h int) {
	fillRect(screen, float32(x), float32(y), float32(w), float32(h), panelColor)
	vector.StrokeRect(screen, float32(x)+1, float32(y)+1, float32(w)-2, float32(h)-2, 2, borderColor, false)
}

// ---- entity legend ----

// kindInfo carries a kind's ECAMS identity: letter, letter color, plaque
// fill, and name. Mirrors the palette in docs/art-style.md for the game's
// six structure kinds.
type kindInfo struct {
	name   string
	letter string
	lc     color.RGBA // letter color
	fill   color.RGBA // plaque fill
}

var kinds = map[string]kindInfo{
	game.KindCore:           {"CORE", "C", bright, color.RGBA{0x56, 0x1A, 0x16, 0xFF}},
	game.KindEnergyProducer: {"ENERGY PROD", "E", color.RGBA{0x46, 0xD2, 0x64, 0xFF}, color.RGBA{0x12, 0x2A, 0x1C, 0xFF}},
	game.KindElementMachine: {"ELEMENT MACH", "M", color.RGBA{0x5A, 0xBE, 0xE6, 0xFF}, color.RGBA{0x10, 0x28, 0x32, 0xFF}},
	game.KindFactory:        {"FACTORY", "F", color.RGBA{0xB4, 0xD2, 0x5A, 0xFF}, color.RGBA{0x20, 0x2A, 0x0E, 0xFF}},
	game.KindTurret:         {"TURRET", "T", color.RGBA{0xEB, 0x6E, 0x4B, 0xFF}, color.RGBA{0x2E, 0x18, 0x10, 0xFF}},
	game.KindWall:           {"WALL", "W", color.RGBA{0xAA, 0x8C, 0x5A, 0xFF}, color.RGBA{0x26, 0x20, 0x10, 0xFF}},
}

// kindOrder is the legend's canonical order.
var kindOrder = []string{
	game.KindCore,
	game.KindEnergyProducer,
	game.KindElementMachine,
	game.KindFactory,
	game.KindTurret,
	game.KindWall,
}

// enemyInfo is the enemy's legend entry: a red "X" blip.
var enemyInfo = kindInfo{name: "ENEMY", letter: "X", lc: red, fill: enemyFill}

// ---- coordinates ----

// regionBounds returns the de-fogged region's cell bounds (inclusive), or the
// core's cell when the fog set is empty (restored saves).
func regionBounds(gs *game.GameState) (minX, minY, maxX, maxY int64) {
	if len(gs.Fog) == 0 {
		cx, cy := gs.CoreCell()
		return cx, cy, cx, cy
	}
	first := true
	for cp := range gs.Fog {
		x0, y0 := int64(cp.X)*game.CHUNK_SIZE, int64(cp.Y)*game.CHUNK_SIZE
		x1, y1 := x0+game.CHUNK_SIZE-1, y0+game.CHUNK_SIZE-1
		if first {
			minX, minY, maxX, maxY = x0, y0, x1, y1
			first = false
			continue
		}
		minX, minY = min(minX, x0), min(minY, y0)
		maxX, maxY = max(maxX, x1), max(maxY, y1)
	}
	return minX, minY, maxX, maxY
}

// ---- render ----

// Draw renders the whole terminal. App calls this after the fx layer update;
// the fx layer draws on top of the grid, then the game-over overlay, then the
// debug panel (debug builds).
func (a *App) Draw(screen *ebiten.Image) {
	screen.Fill(bgColor)
	drawHeader(screen, a.gs)
	a.lastVisibleEnemies = drawGrid(screen, a.worldBuf, a.gs, a.cam, a.curX, a.curY, a.sel, a.gs.GameOver())
	drawStock(screen, a.gs)
	drawTile(screen, a.gs, a.curX, a.curY)
	drawLegend(screen)
	drawSelect(screen, a.gs, a.sel)
	drawStatus(screen, a.gs, a.curX, a.curY, a.sel, a.lastVisibleEnemies)
	a.fx.Draw(screen, a.worldBuf, a.cam)
	if a.gs.GameOver() {
		drawGameOver(screen, a.gs)
	}
	if debugDraw != nil {
		debugDraw(screen)
	}
}

func drawHeader(screen *ebiten.Image, gs *game.GameState) {
	drawText(screen, "CORE DEFENSE", face(30), 16, 16, bright)
	drawText(screen, "ELECTRONIC CENTRALIZED MONITORING", face(20), 356, 28, dimText)
	t := fmt.Sprintf("T %02d:%02d", int(gs.Time/60), int(gs.Time)%60)
	drawTextRight(screen, t, face(22), 1272, 28, amber)
}

// drawGrid renders the visible world into buf (the App's offscreen, whose
// 0,0 is the panel's top-left) and blits it at the panel rect. Cells are
// procedurally drawn every frame — nothing is stored in memory for them;
// per-chunk image caching is the optimization to revisit when the de-fogged
// region grows (ticket #21). Drawing into a screen SubImage is avoided:
// ebiten's vector fills ignore the subimage offset.
func drawGrid(screen, buf *ebiten.Image, gs *game.GameState, cam *Camera, curX, curY int64, sel string, gameOver bool) int {
	panel(screen, panelX, panelY, panelW, panelH)
	buf.Clear()
	minX, minY, maxX, maxY := regionBounds(gs)

	// Cells: the visible range at the camera, void beyond the de-fogged
	// region — the infinite grid is always there, dark where unexplored.
	vx0, vy0, vx1, vy1 := cam.VisibleCells()
	for gy := vy0; gy <= vy1; gy++ {
		for gx := vx0; gx <= vx1; gx++ {
			x, y := cam.CellTopLeft(gx, gy)
			sz := float32(cam.Scale)
			if gx < minX || gx > maxX || gy < minY || gy > maxY {
				fillRounded(buf, float32(x+2*float64(sz)), float32(y+2*float64(sz)), CELL*sz-4*sz, CELL*sz-4*sz, 6*sz, voidColor)
				continue
			}
			fillRounded(buf, float32(x+2*float64(sz)), float32(y+2*float64(sz)), CELL*sz-4*sz, CELL*sz-4*sz, 6*sz, emptyCell)
			if s := gs.StructureAt(gx, gy); s != nil {
				drawStructure(buf, gs, cam, s, x, y)
			}
		}
	}

	// Chunk boundary lines inside the view (the de-fogged region's edges read
	// as the panel border; interior edges show when multiple chunks are lit).
	for e := vx0 + 1; e <= vx1; e++ {
		if e%game.CHUNK_SIZE == 0 {
			x := float32((float64(e) - cam.X) * CELL * cam.Scale)
			vector.StrokeLine(buf, x, 0, x, float32(panelH), float32(2*cam.Scale), borderColor, false)
		}
	}
	for e := vy0 + 1; e <= vy1; e++ {
		if e%game.CHUNK_SIZE == 0 {
			y := float32((float64(e) - cam.Y) * CELL * cam.Scale)
			vector.StrokeLine(buf, 0, y, float32(panelW), y, float32(2*cam.Scale), borderColor, false)
		}
	}

	visible := drawEnemies(buf, gs, cam)
	drawDrones(buf, gs, cam)

	// Placement ghost: the selected kind's letter at low alpha on the cursor
	// cell when it is free (skip while game over — the sim is frozen).
	if !gameOver && gs.StructureAt(curX, curY) == nil {
		ki, ok := kinds[sel]
		if ok {
			cx, cy := cam.CellTopLeft(curX, curY)
			c := color.RGBA{ki.lc.R, ki.lc.G, ki.lc.B, 90}
			drawTextCentered(buf, ki.letter, face(24*cam.Scale), cx+CELL*cam.Scale/2, cy+CELL*cam.Scale/2, CELL*cam.Scale*0.9, c)
		}
	}

	// Cursor.
	cx, cy := cam.CellTopLeft(curX, curY)
	sz := float32(cam.Scale)
	strokeRounded(buf, float32(cx+2*float64(sz)), float32(cy+2*float64(sz)), CELL*sz-4*sz, CELL*sz-4*sz, 6*sz, 3*sz, cursorCol)

	// Blit the world onto the panel.
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(panelX, panelY)
	screen.DrawImage(buf, opts)
	return visible
}

// drawStructure renders one plaque: kind-colored letter on its fill, plus a
// thin HP strip when damaged.
func drawStructure(screen *ebiten.Image, gs *game.GameState, cam *Camera, s *game.Structure, x, y float64) {
	ki, ok := kinds[s.Kind]
	if !ok {
		ki = kindInfo{letter: s.Kind, lc: bright, fill: panelColor}
	}
	sz := float32(cam.Scale)
	cp := CELL * sz
	fillRounded(screen, float32(x+2*float64(sz)), float32(y+2*float64(sz)), cp-4*sz, cp-4*sz, 6*sz, ki.fill)
	drawTextCentered(screen, ki.letter, face(22*cam.Scale), x+float64(cp)/2, y+float64(cp)/2, float64(cp)*0.9, ki.lc)

	if e, ok := gs.Reg.Get(s.Kind); ok && e.Stats.HP > 0 && s.HP < e.Stats.HP {
		frac := float32(max(0, s.HP)) / float32(e.Stats.HP)
		c := hpGreen
		if frac <= 0.25 {
			c = hpRed
		} else if frac <= 0.5 {
			c = hpAmber
		}
		fillRect(screen, float32(x+6*float64(sz)), float32(y+float64(cp-10*sz)), float32(cp-12*sz)*frac, 4*sz, c)
		fillRect(screen, float32(x+6*float64(sz))+float32(cp-12*sz)*frac, float32(y+float64(cp-10*sz)), float32(cp-12*sz)*(1-frac), 4*sz, emptyCell)
	}
}

// enemyPlaque returns the pre-rendered enemy blip (fill + stroke + X) for a
// zoom scale, built once per ladder step. Drawing the static plaque is one
// blit per enemy instead of two rounded-path fills plus a text draw — the
// per-frame render cost no longer scales with the enemy population (the
// deferred vector fill of thousands of paths was the frame killer).
var enemyPlaqueCache = map[float64]*ebiten.Image{}

func enemyPlaque(scale float64) *ebiten.Image {
	if img, ok := enemyPlaqueCache[scale]; ok {
		return img
	}
	sz := float32(scale)
	m := 2 * sz // stroke overhang + anti-aliasing margin
	size := int(32*sz + 2*m)
	img := ebiten.NewImage(size, size)
	fillRounded(img, m, m, 32*sz, 32*sz, 8*sz, enemyFill)
	strokeRounded(img, m, m, 32*sz, 32*sz, 8*sz, 2*sz, red)
	drawTextCentered(img, "X", face(20*scale), float64(m+16*sz), float64(m+16*sz+1), 24*scale, red)
	enemyPlaqueCache[scale] = img
	return img
}

// drawEnemies draws the enemy blips in the panel and returns how many were
// visible (passed the viewport cull) — the count on screen this frame, for
// the status bar's VISIBLE readout.
func drawEnemies(view *ebiten.Image, gs *game.GameState, cam *Camera) int {
	sz := float32(cam.Scale)
	plaque := enemyPlaque(cam.Scale)
	m := float64(32*sz + 4)          // cull margin: plaque half-width + stroke + slack
	var opts ebiten.DrawImageOptions // reused per enemy (blit transform only)
	visible := 0
	for i := range gs.Enemies {
		e := &gs.Enemies[i]
		sx, sy := cam.WorldToScreen(e.FX, e.FY)
		// Viewport cull: enemies outside the panel skip all draw work. The
		// population may be large while only a windowful is ever visible.
		if sx < -m || sx > panelW+m || sy < -m || sy > panelH+m {
			continue
		}
		visible++
		opts.GeoM.Reset()
		opts.GeoM.Translate(sx-16*float64(sz)-2*float64(sz), sy-16*float64(sz)-2*float64(sz))
		view.DrawImage(plaque, &opts)
		// HP strip only when damaged — full-HP enemies skip two fills.
		if e.HP < gs.Tune.EnemyHP {
			frac := float32(max(0, e.HP)) / float32(gs.Tune.EnemyHP)
			fillRect(view, float32(sx-16*float64(sz)), float32(sy+18*float64(sz)), 32*sz*frac, 4*sz, red)
			fillRect(view, float32(sx-16*float64(sz))+32*sz*frac, float32(sy+18*float64(sz)), 32*sz*(1-frac), 4*sz, emptyCell)
		}
	}
	return visible
}

func drawDrones(view *ebiten.Image, gs *game.GameState, cam *Camera) {
	sz := float32(cam.Scale)
	for i := range gs.Drones {
		d := &gs.Drones[i]
		p := d.T / d.Dur
		if p > 1 {
			p = 1
		}
		dx, dy := d.SX+(d.TX-d.SX)*p, d.SY+(d.TY-d.SY)*p
		sx, sy := cam.WorldToScreen(dx, dy)
		x0, y0 := cam.WorldToScreen(d.SX, d.SY)
		x1, y1 := cam.WorldToScreen(d.TX, d.TY)
		vector.StrokeLine(view, float32(x0), float32(y0), float32(x1), float32(y1), 2*sz, color.RGBA{0x2D, 0x50, 0x3C, 0x60}, false)

		// Diamond blip + item letter (E element, A ammo).
		pth := &vector.Path{}
		pth.MoveTo(float32(sx), float32(sy-12*float64(sz)))
		pth.LineTo(float32(sx+12*float64(sz)), float32(sy))
		pth.LineTo(float32(sx), float32(sy+12*float64(sz)))
		pth.LineTo(float32(sx-12*float64(sz)), float32(sy))
		pth.Close()
		opts := &vector.DrawPathOptions{}
		opts.ColorScale.ScaleWithColor(color.RGBA{0x16, 0x1E, 0x1A, 0xFF})
		vector.FillPath(view, pth, &vector.FillOptions{}, opts)
		letter := "E"
		if d.Item == "am" {
			letter = "A"
		}
		drawTextCentered(view, letter, face(16*cam.Scale), sx, sy+1, 20*cam.Scale, bright)
	}
}

// ---- right column ----

func drawStock(screen *ebiten.Image, gs *game.GameState) {
	x, y, w := rightX, 60, 708
	panel(screen, x, y, w, 108)
	drawText(screen, "STOCKPILE", face(20), float64(x+20), float64(y+24), bright)
	rows := []struct {
		k string
		v string
		c color.RGBA
	}{
		{"EL", fmt.Sprintf("%04d", gs.Stockpile.El), color.RGBA{0x5A, 0xBE, 0xE6, 0xFF}},
		{"AM", fmt.Sprintf("%04d", gs.Stockpile.Am), amber},
		{"EN", fmt.Sprintf("%04.0f", gs.Stockpile.En), hpGreen},
	}
	for i, r := range rows {
		drawText(screen, r.k, face(20), float64(x+20), float64(y+52+i*28), dimText)
		drawTextRight(screen, r.v, face(20), float64(x+w-20), float64(y+52+i*28), r.c)
	}
}

func drawTile(screen *ebiten.Image, gs *game.GameState, curX, curY int64) {
	x, y, w := rightX, 180, 708
	panel(screen, x, y, w, 124)
	drawText(screen, "TILE", face(20), float64(x+20), float64(y+24), bright)

	s := gs.StructureAt(curX, curY)
	if s == nil {
		drawText(screen, "VOID", face(26), float64(x+20), float64(y+56), dimText)
		drawText(screen, "EMPTY — PLACE SELECTED", face(20), float64(x+20), float64(y+92), dimText)
		return
	}
	ki, ok := kinds[s.Kind]
	if !ok {
		ki = kindInfo{name: s.Kind, lc: bright}
	}
	drawText(screen, ki.name, face(24), float64(x+20), float64(y+54), ki.lc)
	if e, ok := gs.Reg.Get(s.Kind); ok {
		hp := fmt.Sprintf("HP %3.0f/%.0f", s.HP, e.Stats.HP)
		drawText(screen, hp, face(20), float64(x+20), float64(y+86), dimText)
		frac := float32(max(0, s.HP)) / float32(e.Stats.HP)
		c := hpGreen
		if frac <= 0.25 {
			c = hpRed
		} else if frac <= 0.5 {
			c = hpAmber
		}
		fillRect(screen, float32(x+160), float32(y+86), 120*frac, 6, c)
		if s.Buffer.El > 0 || s.Buffer.Am > 0 {
			buf := fmt.Sprintf("BUF EL %d  AM %d", s.Buffer.El, s.Buffer.Am)
			drawText(screen, buf, face(20), float64(x+20), float64(y+110), dimText)
		}
	}
}

func drawLegend(screen *ebiten.Image) {
	x, y, w := rightX, 316, 708
	panel(screen, x, y, w, 200)
	drawText(screen, "LEGEND", face(20), float64(x+20), float64(y+24), bright)
	items := make([]kindInfo, 0, len(kindOrder)+1)
	for _, k := range kindOrder {
		items = append(items, kinds[k])
	}
	items = append(items, enemyInfo)
	for i, it := range items {
		cx := x + 20 + (i/4)*330
		cy := y + 56 + (i%4)*40
		fillRounded(screen, float32(cx), float32(cy), 26, 26, 6, it.fill)
		strokeRounded(screen, float32(cx), float32(cy), 26, 26, 6, 2, borderColor)
		drawTextCentered(screen, it.letter, face(18), float64(cx)+13, float64(cy)+13, 24, it.lc)
		drawText(screen, it.name, face(18), float64(cx+36), float64(cy+6), dimText)
	}
}

func drawSelect(screen *ebiten.Image, gs *game.GameState, sel string) {
	x, y, w := rightX, 528, 708
	panel(screen, x, y, w, 100)
	drawText(screen, "SELECT", face(20), float64(x+20), float64(y+24), bright)

	ki, ok := kinds[sel]
	if !ok {
		return
	}
	drawText(screen, fmt.Sprintf("[%s] %s", ki.letter, ki.name), face(24), float64(x+20), float64(y+56), ki.lc)
	if e, ok := gs.Reg.Get(sel); ok {
		cost := fmt.Sprintf("COST %g EL", e.Stats.CostEl)
		if e.Stats.CostEn > 0 {
			cost += fmt.Sprintf(" + %g EN", e.Stats.CostEn)
		}
		drawTextRight(screen, cost, face(20), float64(x+w-20), float64(y+58), dimText)
	}
}

// ---- bottom bar ----

func drawStatus(screen *ebiten.Image, gs *game.GameState, curX, curY int64, sel string, visibleEnemies int) {
	panel(screen, 8, barY, 1264, barH)

	core := gs.Core()
	var coreHP string
	if core == nil {
		coreHP = "—"
	} else {
		coreHP = fmt.Sprintf("%3.0f/%.0f", core.HP, gs.Tune.Kinds[game.KindCore].HP)
	}
	tile := "VOID"
	if s := gs.StructureAt(curX, curY); s != nil {
		if ki, ok := kinds[s.Kind]; ok {
			tile = ki.name
		}
	}
	spawn := "OFF"
	if gs.Spawn.On {
		spawn = "ON"
	}
	line := fmt.Sprintf("CORE %-8s ENEMIES %02d  VISIBLE %02d  KILLED %02d  CURSOR %02d,%02d  TILE %-11s SPAWN %s",
		coreHP, len(gs.Enemies), visibleEnemies, gs.Metrics.Killed, curY, curX, tile, spawn)
	drawText(screen, line, face(20), 28, float64(barY+20), bright)

	ki, ok := kinds[sel]
	if !ok {
		return
	}
	hint := fmt.Sprintf("SELECT [%s] %s  [ENTER] PLACE  [G] SPAWN %s  [R] RESTART  [WASD] PAN  [+/-] ZOOM  [WINDOW X] EXIT",
		ki.letter, ki.name, map[bool]string{true: "OFF", false: "ON"}[gs.Spawn.On])
	drawText(screen, hint, face(20), 28, float64(barY+54), ki.lc)
}

// ---- game over ----

func drawGameOver(screen *ebiten.Image, gs *game.GameState) {
	veil := screen.SubImage(image.Rect(0, 0, ScreenW, ScreenH)).(*ebiten.Image)
	veil.Fill(color.RGBA{0x00, 0x00, 0x00, 0xC8})
	drawTextCentered(screen, "GAME OVER — CORE DESTROYED", face(40), ScreenW/2, 320, 52, red)
	survived := fmt.Sprintf("SURVIVED %02d:%02d", int(gs.Time/60), int(gs.Time)%60)
	drawTextCentered(screen, survived, face(24), ScreenW/2, 400, 32, bright)
	drawTextCentered(screen, "[R] RESTART   [WINDOW X] EXIT", face(22), ScreenW/2, 456, 28, dimText)
}
