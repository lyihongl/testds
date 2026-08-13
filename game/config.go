package game

import (
	"errors"
	"os"

	"github.com/BurntSushi/toml"
)

// Tuning holds every live-tunable number in the game (ADR 0004). All numerics
// are float64 so the debug slider panel can reach every value uniformly;
// counts (costs, buffer sizes) are cast to int at use sites. The Go source
// table is the default; an optional config.toml overlays present fields only.
type Tuning struct {
	StartEl    float64 `toml:"start_el"`
	EnemyHP    float64 `toml:"enemy_hp"`
	EnemyDPS   float64 `toml:"enemy_dps"`
	EnemySpeed float64 `toml:"enemy_speed"`
	SpawnEvery float64 `toml:"spawn_every"`
	Grace      float64 `toml:"grace"`
	DroneSpeed float64 `toml:"drone_speed"`

	// Kinds is the per-kind stats table (registry source).
	Kinds map[string]*Stats `toml:"kinds"`
}

// Stats is the per-kind stats table. Fields are zero when a kind doesn't use
// them (one concrete struct so the table stays a plain map — ADR 0004).
type Stats struct {
	HP     float64 `toml:"hp"`
	CostEl float64 `toml:"cost_el"`
	CostEn float64 `toml:"cost_en"`
	Buffer float64 `toml:"buffer"`

	// Energy producer (E).
	EnergyPerSec float64 `toml:"energy_per_s"`
	// Element machine and factory (M, F): production cadence.
	ProduceEvery float64 `toml:"produce_every"`
	// Factory (F): conversion.
	FactoryEl   float64 `toml:"factory_el"`
	FactoryAmmo float64 `toml:"factory_ammo"`
	// Turret (T).
	TurretRange    float64 `toml:"turret_range"`
	TurretDmg      float64 `toml:"turret_dmg"`
	TurretShots    float64 `toml:"turret_shots"`
	TurretAmmoCost float64 `toml:"turret_ammo_cost"`
}

// DefaultTuning returns the base tuning table — the numbers from "Define the
// basic game loop" (all tuning-pending).
func DefaultTuning() *Tuning {
	return &Tuning{
		StartEl:    30,
		EnemyHP:    30,
		EnemyDPS:   5,
		EnemySpeed: 1,
		SpawnEvery: 2.5,
		Grace:      6,
		DroneSpeed: 9,
		Kinds: map[string]*Stats{
			KindCore:           {HP: 100},
			KindEnergyProducer: {HP: 50, CostEl: 3, EnergyPerSec: 10},
			KindElementMachine: {HP: 50, CostEl: 3, Buffer: 3, ProduceEvery: 2},
			KindFactory:        {HP: 50, CostEl: 5, Buffer: 3, ProduceEvery: 1, FactoryEl: 2, FactoryAmmo: 1},
			KindTurret:         {HP: 50, CostEl: 5, CostEn: 10, Buffer: 3, TurretRange: 4, TurretDmg: 10, TurretShots: 1, TurretAmmoCost: 1},
			KindWall:           {HP: 50, CostEl: 2},
		},
	}
}

// LoadTuning returns DefaultTuning overlaid with the TOML file at path, if it
// exists. Present fields override defaults; absent ones keep them. A missing
// file is not an error — it means pure defaults.
//
// The file decodes into a fresh Tuning and is merged field-by-field (a
// non-zero value means "present") because TOML decoding replaces map entries
// rather than overlaying them.
func LoadTuning(path string) (*Tuning, error) {
	t := DefaultTuning()
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return t, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var overlay Tuning
	if _, err := toml.NewDecoder(f).Decode(&overlay); err != nil {
		return nil, err
	}
	mergeTuning(t, &overlay)
	return t, nil
}

// mergeTuning overlays present (non-zero) fields of src onto dst.
func mergeTuning(dst, src *Tuning) {
	if src.StartEl != 0 {
		dst.StartEl = src.StartEl
	}
	if src.EnemyHP != 0 {
		dst.EnemyHP = src.EnemyHP
	}
	if src.EnemyDPS != 0 {
		dst.EnemyDPS = src.EnemyDPS
	}
	if src.EnemySpeed != 0 {
		dst.EnemySpeed = src.EnemySpeed
	}
	if src.SpawnEvery != 0 {
		dst.SpawnEvery = src.SpawnEvery
	}
	if src.Grace != 0 {
		dst.Grace = src.Grace
	}
	if src.DroneSpeed != 0 {
		dst.DroneSpeed = src.DroneSpeed
	}
	for kind, ov := range src.Kinds {
		d, ok := dst.Kinds[kind]
		if !ok {
			dst.Kinds[kind] = ov
			continue
		}
		mergeStats(d, ov)
	}
}

// mergeStats overlays present (non-zero) fields of src onto dst.
func mergeStats(dst, src *Stats) {
	if src.HP != 0 {
		dst.HP = src.HP
	}
	if src.CostEl != 0 {
		dst.CostEl = src.CostEl
	}
	if src.CostEn != 0 {
		dst.CostEn = src.CostEn
	}
	if src.Buffer != 0 {
		dst.Buffer = src.Buffer
	}
	if src.EnergyPerSec != 0 {
		dst.EnergyPerSec = src.EnergyPerSec
	}
	if src.ProduceEvery != 0 {
		dst.ProduceEvery = src.ProduceEvery
	}
	if src.FactoryEl != 0 {
		dst.FactoryEl = src.FactoryEl
	}
	if src.FactoryAmmo != 0 {
		dst.FactoryAmmo = src.FactoryAmmo
	}
	if src.TurretRange != 0 {
		dst.TurretRange = src.TurretRange
	}
	if src.TurretDmg != 0 {
		dst.TurretDmg = src.TurretDmg
	}
	if src.TurretShots != 0 {
		dst.TurretShots = src.TurretShots
	}
	if src.TurretAmmoCost != 0 {
		dst.TurretAmmoCost = src.TurretAmmoCost
	}
}
