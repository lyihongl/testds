package game

import "math"

// Logistics is the drone transport layer (CONTEXT.md: drone, logistics): the
// request side (one outstanding trip per machine cell+item), trip travel, and
// delivery between machine buffers and the stockpile point (the core's cell —
// the base has no separate depot, grill Q4).
//
// The module is stateless: trips live in GameState (Drone structs on
// gs.Drones), this file is pure behavior over it. Its interface is
// requestDrone + stepDrones — machines request with one call and never touch
// trip mechanics, dedup, or buffer rules.
//
// Edge semantics (pinned, not to change casually): a destroyed machine loses
// its in-flight export (or keeps the import's item in the stockpile);
// delivery clamps imports at the machine's buffer cap; one trip per
// cell+item.

// requestDrone starts one trip carrying item between the machine cell and the
// stockpile point. toStockpile true: machine → core; false: core → machine.
// It returns false when a trip for that cell+item is already in flight — the
// one-outstanding-request invariant — so callers never run a separate dedup
// check.
func requestDrone(gs *GameState, gx, gy int64, item string, toStockpile bool) bool {
	for _, d := range gs.Drones {
		if d.GX == gx && d.GY == gy && d.Item == item {
			return false
		}
	}
	var src, dst Vec
	if toStockpile {
		src = Vec{X: float64(gx) + 0.5, Y: float64(gy) + 0.5}
		dst = gs.coreCenter()
	} else {
		src = gs.coreCenter()
		dst = Vec{X: float64(gx) + 0.5, Y: float64(gy) + 0.5}
	}
	dist := math.Hypot(dst.X-src.X, dst.Y-src.Y)
	gs.Drones = append(gs.Drones, Drone{
		Item:    item,
		ToDepot: toStockpile,
		SX:      src.X, SY: src.Y,
		TX: dst.X, TY: dst.Y,
		Dur: dist / gs.Tune.DroneSpeed,
		GX:  gx, GY: gy,
	})
	return true
}

// stepDrones advances every in-flight trip by dt and applies deliveries for
// trips that reached their destination; completed drones leave the list.
func stepDrones(gs *GameState, dt float64) {
	alive := gs.Drones[:0]
	for _, d := range gs.Drones {
		d.T += dt
		if d.T < d.Dur {
			alive = append(alive, d)
			continue
		}
		deliver(gs, d)
	}
	gs.Drones = alive
}

// deliver applies a completed trip: one item moves between the machine cell's
// buffer and the stockpile. If the machine is gone, the item is lost (or, for
// imports, stays in the stockpile).
func deliver(gs *GameState, d Drone) {
	m := gs.StructureAt(d.GX, d.GY)
	if m == nil {
		return
	}
	if d.ToDepot { // machine → stockpile
		subBufferItem(&m.Buffer, d.Item)
		addStock(gs, d.Item)
	} else { // stockpile → machine
		addBufferItem(&m.Buffer, d.Item, bufferCap(gs, m.Kind))
		subStock(gs, d.Item)
	}
}

// bufferCap returns a kind's buffer capacity from the registry.
func bufferCap(gs *GameState, kind string) int {
	if e, ok := gs.Reg.Get(kind); ok {
		return int(e.Stats.Buffer)
	}
	return 0
}

func addBufferItem(b *Buffer, item string, cap int) {
	switch item {
	case "el":
		b.El = min(b.El+1, cap)
	case "am":
		b.Am = min(b.Am+1, cap)
	}
}

func subBufferItem(b *Buffer, item string) {
	switch item {
	case "el":
		b.El = max(b.El-1, 0)
	case "am":
		b.Am = max(b.Am-1, 0)
	}
}

func addStock(gs *GameState, item string) {
	switch item {
	case "el":
		gs.Stockpile.El++
	case "am":
		gs.Stockpile.Am++
	}
}

func subStock(gs *GameState, item string) {
	switch item {
	case "el":
		gs.Stockpile.El = max(gs.Stockpile.El-1, 0)
	case "am":
		gs.Stockpile.Am = max(gs.Stockpile.Am-1, 0)
	}
}
