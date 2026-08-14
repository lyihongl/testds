package game

import "testing"

// The logistics tests pin the module's interface directly (requestDrone +
// stepDrones); the delivery economics also stay pinned end-to-end through
// SimulationStep by TestElementMachineShipsElementsToStockpile.

func TestRequestDroneDedupsPerCellAndItem(t *testing.T) {
	gs := NewGameState()
	if !requestDrone(gs, 2, 2, "el", true) {
		t.Fatal("first request for a cell+item should be accepted")
	}
	if requestDrone(gs, 2, 2, "el", true) {
		t.Fatal("second request for the same cell+item should be refused")
	}
	if !requestDrone(gs, 2, 2, "am", true) {
		t.Fatal("a different item at the same cell should be accepted")
	}
	if !requestDrone(gs, 3, 3, "el", true) {
		t.Fatal("the same item at a different cell should be accepted")
	}
	if len(gs.Drones) != 3 {
		t.Fatalf("want 3 drones in flight, got %d", len(gs.Drones))
	}
}

func TestExportTripDeliversToStockpile(t *testing.T) {
	gs := NewGameState()
	gs.SetStructure(2, 2, &Structure{Kind: KindElementMachine, HP: 50, Buffer: Buffer{El: 1}})
	if !requestDrone(gs, 2, 2, "el", true) {
		t.Fatal("request refused")
	}
	d := gs.Drones[0]
	stepDrones(gs, d.Dur+0.01) // complete the trip
	if m := gs.StructureAt(2, 2); m.Buffer.El != 0 {
		t.Fatalf("machine buffer not drained: el=%d", m.Buffer.El)
	}
	if gs.Stockpile.El != int(gs.Tune.StartEl)+1 {
		t.Fatalf("stockpile el=%d, want %d", gs.Stockpile.El, int(gs.Tune.StartEl)+1)
	}
	if len(gs.Drones) != 0 {
		t.Fatalf("trip should have drained, %d drones left", len(gs.Drones))
	}
}

func TestImportTripDeliversToMachineBuffer(t *testing.T) {
	gs := NewGameState()
	gs.SetStructure(2, 2, &Structure{Kind: KindFactory, HP: 50, Buffer: Buffer{}})
	if !requestDrone(gs, 2, 2, "el", false) {
		t.Fatal("request refused")
	}
	d := gs.Drones[0]
	stepDrones(gs, d.Dur+0.01)
	if m := gs.StructureAt(2, 2); m.Buffer.El != 1 {
		t.Fatalf("machine buffer not filled: el=%d", m.Buffer.El)
	}
	if gs.Stockpile.El != int(gs.Tune.StartEl)-1 {
		t.Fatalf("stockpile el=%d, want %d", gs.Stockpile.El, int(gs.Tune.StartEl)-1)
	}
}

func TestImportDeliveryClampsAtBufferCap(t *testing.T) {
	gs := NewGameState()
	gs.SetStructure(2, 2, &Structure{Kind: KindFactory, HP: 50, Buffer: Buffer{El: 3}})
	requestDrone(gs, 2, 2, "el", false)
	d := gs.Drones[0]
	stepDrones(gs, d.Dur+0.01)
	if m := gs.StructureAt(2, 2); m.Buffer.El != 3 {
		t.Fatalf("buffer overflowed the cap: el=%d", m.Buffer.El)
	}
	// Pinned as-is: the import is debited from the stockpile regardless of the
	// clamp (the machine-side request guard normally prevents this state).
	if gs.Stockpile.El != int(gs.Tune.StartEl)-1 {
		t.Fatalf("stockpile el=%d, want %d", gs.Stockpile.El, int(gs.Tune.StartEl)-1)
	}
}

func TestDestroyedMachineLosesTrip(t *testing.T) {
	gs := NewGameState()
	gs.SetStructure(2, 2, &Structure{Kind: KindElementMachine, HP: 50, Buffer: Buffer{El: 1}})
	requestDrone(gs, 2, 2, "el", true)
	d := gs.Drones[0]
	gs.SetStructure(2, 2, nil) // machine destroyed mid-trip
	stepDrones(gs, d.Dur+0.01)
	if gs.Stockpile.El != int(gs.Tune.StartEl) {
		t.Fatalf("item should be lost with the machine: el=%d", gs.Stockpile.El)
	}
	if len(gs.Drones) != 0 {
		t.Fatalf("trip should still drain, %d drones left", len(gs.Drones))
	}
}

func TestStepDrainsOnlyCompletedTrips(t *testing.T) {
	gs := NewGameState()
	gs.SetStructure(2, 2, &Structure{Kind: KindElementMachine, HP: 50, Buffer: Buffer{El: 1}})
	requestDrone(gs, 2, 2, "el", true)
	d := gs.Drones[0]
	stepDrones(gs, d.Dur/2)
	if len(gs.Drones) != 1 {
		t.Fatalf("trip should still be in flight at half duration")
	}
	stepDrones(gs, d.Dur/2+0.01)
	if len(gs.Drones) != 0 {
		t.Fatalf("trip should drain after its duration, %d drones left", len(gs.Drones))
	}
}
