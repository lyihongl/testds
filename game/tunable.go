//go:build debug

package game

// Tunable describes one live-tunable value for the debug panel (ADR 0004).
// V points into the live Tuning/Stats struct, so writing through it is
// immediately visible to the sim. This file compiles only with `-tags debug`.
type Tunable struct {
	Name string
	V    *float64
	Min  float64
	Max  float64
}

// RegisterTunables appends every tunable value in t to out — the single
// registration the debug panel's sliders are built from.
func RegisterTunables(t *Tuning, out *[]Tunable) {
	add := func(name string, v *float64, min, max float64) {
		*out = append(*out, Tunable{Name: name, V: v, Min: min, Max: max})
	}
	add("start_el", &t.StartEl, 0, 500)
	add("enemy.hp", &t.EnemyHP, 1, 500)
	add("enemy.dps", &t.EnemyDPS, 0, 50)
	add("enemy.speed", &t.EnemySpeed, 0, 10)
	add("spawn.every", &t.SpawnEvery, 0.1, 10)
	add("spawn.grace", &t.Grace, 0, 30)
	add("drone.speed", &t.DroneSpeed, 1, 30)
	for name, st := range t.Kinds {
		p := "kind." + name + "."
		add(p+"hp", &st.HP, 1, 500)
		add(p+"cost_el", &st.CostEl, 0, 50)
		add(p+"cost_en", &st.CostEn, 0, 50)
		add(p+"buffer", &st.Buffer, 0, 20)
		add(p+"energy_per_s", &st.EnergyPerSec, 0, 50)
		add(p+"produce_every", &st.ProduceEvery, 0.1, 10)
		add(p+"factory_el", &st.FactoryEl, 0, 10)
		add(p+"factory_ammo", &st.FactoryAmmo, 0, 10)
		add(p+"turret_range", &st.TurretRange, 0, 20)
		add(p+"turret_dmg", &st.TurretDmg, 0, 100)
		add(p+"turret_shots", &st.TurretShots, 0.1, 10)
		add(p+"turret_ammo_cost", &st.TurretAmmoCost, 0, 10)
	}
}
