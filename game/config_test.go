package game

import (
	"github.com/BurntSushi/toml"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultTuningSanity(t *testing.T) {
	tu := DefaultTuning()
	if tu.StartEl != 30 {
		t.Errorf("StartEl = %v, want 30", tu.StartEl)
	}
	core, ok := tu.Kinds[KindCore]
	if !ok || core.HP != 100 {
		t.Errorf("core stats missing or wrong: %+v", core)
	}
	turret := tu.Kinds[KindTurret]
	if turret.TurretRange != 4 || turret.TurretDmg != 10 || turret.TurretAmmoCost != 1 {
		t.Errorf("turret stats wrong: %+v", turret)
	}
}

func TestLoadTuningOverlaysPartialTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `# comments work in TOML
enemy_dps = 8.0

[Kinds.T]
turret_range = 6.0
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	tu, err := LoadTuning(path)
	if err != nil {
		t.Fatal(err)
	}
	if tu.EnemyDPS != 8.0 {
		t.Errorf("EnemyDPS = %v, want 8.0 (overridden)", tu.EnemyDPS)
	}
	if tu.EnemyHP != 30 {
		t.Errorf("EnemyHP = %v, want 30 (default kept)", tu.EnemyHP)
	}
	if got := tu.Kinds[KindTurret].TurretRange; got != 6.0 {
		t.Errorf("turret_range = %v, want 6.0 (overridden)", got)
	}
	if got := tu.Kinds[KindTurret].TurretDmg; got != 10.0 {
		t.Errorf("turret_dmg = %v, want 10.0 (default kept)", got)
	}
	// Other kinds untouched.
	if got := tu.Kinds[KindFactory].FactoryEl; got != 2.0 {
		t.Errorf("factory_el = %v, want 2.0 (untouched)", got)
	}
}

func TestLoadTuningMissingFileIsDefaults(t *testing.T) {
	tu, err := LoadTuning(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if tu.SpawnEvery != 2.5 {
		t.Errorf("SpawnEvery = %v, want 2.5 default", tu.SpawnEvery)
	}
}

func TestRegistrySharesTuningStats(t *testing.T) {
	gs := NewGameState()
	entry, ok := gs.Reg.Get(KindTurret)
	if !ok {
		t.Fatal("turret not in registry")
	}
	// Mutating the live tuning must be visible through the registry (the
	// debug sliders write through Tune, and the sim reads through Reg).
	gs.Tune.Kinds[KindTurret].TurretDmg = 42
	if entry.Stats.TurretDmg != 42 {
		t.Errorf("registry sees TurretDmg = %v, want 42 (shared pointer)", entry.Stats.TurretDmg)
	}
}

func TestUseTuningRebuildsRegistry(t *testing.T) {
	gs := NewGameState()
	tu := DefaultTuning()
	tu.Kinds[KindWall].HP = 77
	gs.UseTuning(tu)
	e, ok := gs.Reg.Get(KindWall)
	if !ok || e.Stats.HP != 77 {
		t.Errorf("registry not rebuilt from new tuning: %+v", e)
	}
}

func TestTOMLSaveRoundTrip(t *testing.T) {
	// The debug panel's save button encodes the live tuning; LoadTuning must
	// read it back to the same values.
	path := filepath.Join(t.TempDir(), "config.toml")
	src := DefaultTuning()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	enc := toml.NewEncoder(f)
	if err := enc.Encode(src); err != nil {
		t.Fatal(err)
	}
	f.Close()

	got, err := LoadTuning(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.StartEl != src.StartEl || got.EnemyDPS != src.EnemyDPS || got.SpawnEvery != src.SpawnEvery {
		t.Errorf("global fields did not round-trip: %+v", got)
	}
	for kind, want := range src.Kinds {
		g := got.Kinds[kind]
		if g == nil || g.HP != want.HP || g.CostEl != want.CostEl {
			t.Errorf("kind %q did not round-trip: %+v (want %+v)", kind, g, want)
		}
	}
}
