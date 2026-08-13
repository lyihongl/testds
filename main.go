// Command coredefense wires the game (sim) and ui (presentation) packages
// (ADR 0003) and runs the window. Thin by design: build the state, build the
// app, run.
package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"coredefense/game"
	"coredefense/ui"
)

func main() {
	t, err := game.LoadTuning("config.toml")
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	gs := game.NewGameState()
	gs.UseTuning(t)
	gs.Reset()
	if err := ebiten.RunGame(ui.NewApp(gs)); err != nil {
		log.Fatal(err)
	}
}
