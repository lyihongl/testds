//go:build debug

package ui

import (
	"bytes"
	"image/color"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/ebitenui/ebitenui"
	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"

	"coredefense/game"
)

// This file compiles only with `-tags debug` (ADR 0004): the live-tuning
// slider panel. A normal `go build` produces a binary that does not contain
// it at all — player-facing UI stays ECAMS hand-drawn; this is a dev tool.

func init() {
	installDebug = func(gs *game.GameState) {
		p := newDebugPanel(gs)
		debugUpdate = p.update
		debugDraw = p.draw
	}
}

// debugPanel is the dev-only live-tuning panel: an ebitenui widget tree with
// one slider per tunable, built from game.RegisterTunables — the single
// registration source.
type debugPanel struct {
	ui   *ebitenui.UI
	gs   *game.GameState
	save func()
}

func newDebugPanel(gs *game.GameState) *debugPanel {
	src, err := textv2.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic(err) // debug-only; a broken panel should not silently vanish
	}
	var face textv2.Face = &textv2.GoTextFace{Source: src, Size: 16}

	textCol := color.RGBA{0x96, 0xD2, 0xB4, 0xFF}
	track := &widget.SliderTrackImage{Idle: eimage.NewNineSliceColor(color.RGBA{0x2D, 0x50, 0x3C, 0xFF})}
	handle := &widget.ButtonImage{
		Idle:    eimage.NewNineSliceColor(textCol),
		Hover:   eimage.NewNineSliceColor(color.RGBA{0xEB, 0xBE, 0x46, 0xFF}),
		Pressed: eimage.NewNineSliceColor(color.RGBA{0x5A, 0x82, 0x6E, 0xFF}),
	}
	btnImg := &widget.ButtonImage{
		Idle:    eimage.NewNineSliceColor(color.RGBA{0x2D, 0x50, 0x3C, 0xFF}),
		Hover:   eimage.NewNineSliceColor(color.RGBA{0x46, 0x7A, 0x5C, 0xFF}),
		Pressed: eimage.NewNineSliceColor(color.RGBA{0x1C, 0x32, 0x26, 0xFF}),
	}

	p := &debugPanel{gs: gs}

	// Outer column: one row per tunable, then the save button.
	outer := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewGridLayout(
		widget.GridLayoutOpts.Columns(1),
		widget.GridLayoutOpts.Spacing(2, 2),
	)))

	var tunables []game.Tunable
	game.RegisterTunables(gs.Tune, &tunables)
	for _, tn := range tunables {
		scale := 100.0 // two decimal places
		minV, maxV := int(tn.Min*scale), int(tn.Max*scale)
		cur := int(*tn.V * scale)
		if cur < minV {
			cur = minV
		}
		if cur > maxV {
			cur = maxV
		}

		slider := widget.NewSlider(
			widget.SliderOpts.MinMax(minV, maxV),
			widget.SliderOpts.InitialCurrent(cur),
			widget.SliderOpts.FixedHandleSize(12),
			widget.SliderOpts.PageSizeFunc(func() int { return 1 }),
			widget.SliderOpts.TrackImage(track),
			widget.SliderOpts.HandleImage(handle),
			widget.SliderOpts.ChangedHandler(func(args *widget.SliderChangedEventArgs) {
				*tn.V = float64(args.Current) / scale
			}),
			widget.SliderOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.GridLayoutData{MaxWidth: 240})),
		)

		label := widget.NewLabel(
			widget.LabelOpts.Text(tn.Name, &face, &widget.LabelColor{Idle: textCol}),
			widget.LabelOpts.TextOpts(widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.GridLayoutData{MaxWidth: 180}))),
		)

		row := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Spacing(6, 0),
		)))
		row.AddChild(label, slider)
		outer.AddChild(row)
	}

	saveBtn := widget.NewButton(
		widget.ButtonOpts.Text("Save to config.toml", &face, &widget.ButtonTextColor{Idle: textCol}),
		widget.ButtonOpts.Image(btnImg),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			p.save()
		}),
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.GridLayoutData{MaxWidth: 240})),
	)
	outer.AddChild(saveBtn)

	p.save = p.writeTOML
	p.ui = &ebitenui.UI{Container: outer}
	return p
}

func (p *debugPanel) update() error {
	p.ui.Update()
	return nil
}

func (p *debugPanel) draw(screen *ebiten.Image) {
	p.ui.Draw(screen)
}

// writeTOML persists the merged tuning back to config.toml in the working
// directory, so slider changes survive restarts and are reviewable as diff.
func (p *debugPanel) writeTOML() {
	f, err := os.Create("config.toml")
	if err != nil {
		return
	}
	defer f.Close()
	_ = toml.NewEncoder(f).Encode(p.gs.Tune)
}
