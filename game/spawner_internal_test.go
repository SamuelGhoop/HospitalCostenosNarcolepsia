package game

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// §5.3: el intervalo del día 1 va de 18 a 25 s; cada día es el 90 % del
// anterior, con un mínimo de 8 s. Se prueba al mediodía (12:00, franja
// "normal" de rushBands, factor 1) para que las horas pico (§12.8) no
// cambien estos rangos.
func TestArrivalInterval_ShrinksEachDayDownTo8s(t *testing.T) {
	const noon = 12 * 60
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 1000; i++ {
		if d := arrivalInterval(rng, 1, noon); d < 18*time.Second || d > 25*time.Second {
			t.Fatalf("día 1: intervalo %v; debe ir de 18 a 25 s", d)
		}
		if d := arrivalInterval(rng, 2, noon); d < 16200*time.Millisecond || d > 22500*time.Millisecond {
			t.Fatalf("día 2: intervalo %v; debe ir de 16,2 a 22,5 s (×0,9)", d)
		}
		if d := arrivalInterval(rng, 30, noon); d != 8*time.Second {
			t.Fatalf("día 30: intervalo %v; el mínimo es 8 s", d)
		}
	}
}

// §5.3 y §12.8: en hora pico (factor 0,5) el intervalo del día 1 se reduce a
// la mitad; fuera de pico (tranquilo, factor 1,3) se alarga. El piso de
// arrivalFloor sigue mandando aunque el factor lo empuje más abajo.
func TestArrivalInterval_RushHourScalesTheInterval(t *testing.T) {
	const (
		peakMinute  = 11 * 60 // 11:00, dentro de la franja pico 10:00–12:00
		quietMinute = 9 * 60  // 09:00, dentro de la franja tranquila 08:00–10:00
	)
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 1000; i++ {
		if d := arrivalInterval(rng, 1, peakMinute); d < 9*time.Second || d > 12500*time.Millisecond {
			t.Fatalf("día 1 en pico: intervalo %v; debe ir de 9 a 12,5 s (18–25 s × 0,5)", d)
		}
		if d := arrivalInterval(rng, 1, quietMinute); d < 23400*time.Millisecond || d > 32500*time.Millisecond {
			t.Fatalf("día 1 tranquilo: intervalo %v; debe ir de 23,4 a 32,5 s (18–25 s × 1,3)", d)
		}
		// Día 30 ya está en el piso (8 s); en pico (×0,5) seguiría dando 4 s,
		// pero arrivalFloor no se puede bajar.
		if d := arrivalInterval(rng, 30, peakMinute); d != arrivalFloor {
			t.Fatalf("día 30 en pico: intervalo %v; arrivalFloor (%v) manda aunque el factor empuje más abajo", d, arrivalFloor)
		}
	}
}

// §5.3 y §12.8: las franjas de rushBands cubren el turno completo sin
// huecos, con las horas pico en 10:00–12:00 y 15:00–17:30.
func TestRushAt_MatchesTheScheduledBands(t *testing.T) {
	cases := []struct {
		minute     int
		wantFactor float64
		wantPeak   bool
	}{
		{8 * 60, 1.3, false},     // 08:00 tranquilo
		{9*60 + 59, 1.3, false},  // 09:59 todavía tranquilo
		{10 * 60, 0.5, true},     // 10:00 entra el pico
		{11*60 + 59, 0.5, true},  // 11:59 sigue el pico
		{12 * 60, 1.0, false},    // 12:00 vuelve a normal
		{14*60 + 59, 1.0, false}, // 14:59 todavía normal
		{15 * 60, 0.5, true},     // 15:00 entra el segundo pico
		{17*60 + 29, 0.5, true},  // 17:29 sigue el pico
		{17*60 + 30, 0.9, false}, // 17:30 bajando el ritmo
		{19*60 + 59, 0.9, false}, // 19:59 sigue bajando, hasta cerrar
	}
	for _, c := range cases {
		factor, peak := rushAt(c.minute)
		if factor != c.wantFactor || peak != c.wantPeak {
			t.Errorf("rushAt(%d) = (%v, %v); se esperaba (%v, %v)", c.minute, factor, peak, c.wantFactor, c.wantPeak)
		}
	}
}

// §5.3: Mild 40 %, Moderate 35 %, Severe 25 %.
func TestRandomLevel_FollowsTheShares(t *testing.T) {
	const n = 20000
	rng := rand.New(rand.NewSource(1))
	counts := map[hospital.NarcolepsyLevel]int{}
	for i := 0; i < n; i++ {
		counts[randomLevel(rng)]++
	}
	for level, percent := range map[hospital.NarcolepsyLevel]int{hospital.Mild: 40, hospital.Moderate: 35, hospital.Severe: 25} {
		got := counts[level] * 100 / n
		if got < percent-2 || got > percent+2 {
			t.Errorf("%v salió el %d %% de las veces; se esperaba el %d %%", level, got, percent)
		}
	}
}

// §5.3: AdmitPatient se llama al cruzar la puerta, no al aparecer en la calle.
func TestArrival_AdmitPatientIsCalledAtTheDoor(t *testing.T) {
	g := newInternalGame(t)
	g.ArriveForTest("Yeimy Padilla", hospital.Mild)

	for i := 0; i < 39; i++ { // 3,9 s
		g.Tick(tickInterval)
	}
	if n := len(g.h.Patients()); n != 0 {
		t.Fatalf("a los 3,9 s el hospital ya tiene %d pacientes admitidos; todavía va por la acera", n)
	}
	g.Tick(tickInterval) // 4 s: cruza la puerta
	if n := len(g.h.Patients()); n != 1 {
		t.Errorf("al cruzar la puerta el hospital tiene %d pacientes admitidos; se esperaba 1", n)
	}
}

// §5.4: tiempo despierto antes del ataque, según el nivel.
func TestAwakeDuration_DependsOnTheLevel(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	ranges := map[hospital.NarcolepsyLevel][2]time.Duration{
		hospital.Mild:     {60 * time.Second, 90 * time.Second},
		hospital.Moderate: {35 * time.Second, 55 * time.Second},
		hospital.Severe:   {15 * time.Second, 30 * time.Second},
	}
	for level, r := range ranges {
		for i := 0; i < 1000; i++ {
			if d := awakeDuration(rng, level); d < r[0] || d > r[1] {
				t.Fatalf("%v: tiempo despierto %v; debe ir de %v a %v", level, d, r[0], r[1])
			}
		}
	}
}

// El cronómetro de espera también corre en el pasillo (§5.4).
func TestWaiting_KeepsRunningInTheHallway(t *testing.T) {
	g := newInternalGame(t)
	pt := &patient{p: hospital.NewPatient("P-404", "Fantasma", 30, hospital.Mild), stage: InHallway, waited: 30 * time.Second}
	g.patients = append(g.patients, pt)

	for i := 0; i < 10; i++ { // 1 s
		g.Tick(tickInterval)
	}
	if pt.waited != 31*time.Second {
		t.Errorf("en el pasillo, después de 1 s más: waited = %v; se esperaba 31 s", pt.waited)
	}
}

// §5.4: 15/22/30 s según el nivel, −8 % por punto de pericia.
func TestSleepDuration_ByLevelAndSkill(t *testing.T) {
	cases := []struct {
		level hospital.NarcolepsyLevel
		skill int
		want  time.Duration
	}{
		{hospital.Mild, 1, 13800 * time.Millisecond},     // 15 × 0,92
		{hospital.Moderate, 3, 16720 * time.Millisecond}, // 22 × 0,76
		{hospital.Severe, 5, 18 * time.Second},           // 30 × 0,60
	}
	for _, c := range cases {
		if got := sleepDuration(c.level, c.skill); got != c.want {
			t.Errorf("sleepDuration(%v, pericia %d) = %v; se esperaba %v", c.level, c.skill, got, c.want)
		}
	}
}

// §5.4: alta después de 1 / 2 / 3 sueños completos.
func TestSleepsBeforeDischarge(t *testing.T) {
	want := map[hospital.NarcolepsyLevel]int{hospital.Mild: 1, hospital.Moderate: 2, hospital.Severe: 3}
	for level, n := range want {
		if got := sleepsBeforeDischarge(level); got != n {
			t.Errorf("%v: alta después de %d sueños; se esperaban %d", level, got, n)
		}
	}
}

// Las notas del Shift Report también salen en la consulta 5.4.
func TestReport_SevereLinesCarryTheNote(t *testing.T) {
	g := newInternalGame(t)
	id, err := g.CollapseForTest("Kevin Mercado", hospital.Severe, Hallway2)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Dispatch(id, "C-01"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ { // C-01 lo recoge: ya tiene un episodio
		g.Tick(tickInterval)
	}
	g.notes[id] = "(alta)"

	if sev := g.Report().Severe; len(sev) != 1 || sev[0].Note != "(alta)" {
		t.Errorf("consulta 5.4 = %+v; se esperaba %s marcado \"(alta)\"", sev, id)
	}
}

// Si AdmitPatient falla en la puerta (no debería: los IDs son consecutivos),
// el error sale como aviso y el paciente se devuelve: nunca queda adentro
// sin estar admitido.
func TestArrival_IfAdmitPatientFailsThePatientTurnsBack(t *testing.T) {
	g := newInternalGame(t)
	id := g.ArriveForTest("Yeimy Padilla", hospital.Mild)
	pt, _ := g.findPatientLocked(id)
	if err := g.h.AdmitPatient(pt.p); err != nil { // lo admite antes: en la puerta fallará
		t.Fatal(err)
	}

	for i := 0; i < 40; i++ { // 4 s: llega a la puerta
		g.Tick(tickInterval)
	}
	if pt.stage != Leaving {
		t.Errorf("con AdmitPatient fallando, el paciente quedó %v; debía devolverse", pt.stage)
	}
	if len(g.notices) != 1 || !strings.Contains(g.notices[0], hospital.ErrAlreadyAdmitted.Error()) {
		t.Errorf("avisos = %q; se esperaba uno con el error del modelo", g.notices)
	}
}

// nextZone nunca repite la zona actual y puede llevar a cualquiera de las otras.
func TestNextZone_IsAlwaysADifferentZone(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, current := range wanderZones {
		seen := map[Zone]bool{}
		for i := 0; i < 400; i++ {
			next := nextZone(rng, current)
			if next == current {
				t.Fatalf("desde %v, nextZone devolvió la misma zona", current)
			}
			seen[next] = true
		}
		if len(seen) != len(wanderZones)-1 {
			t.Errorf("desde %v solo salieron %d zonas; se esperaban las otras %d", current, len(seen), len(wanderZones)-1)
		}
	}
}
