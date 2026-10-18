package game

import (
	"math/rand"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// tickArrivalsLocked cuenta el intervalo de llegadas y, cuando se cumple,
// aparece un paciente en la calle (GAME_DESIGN §5.3). Si el mapa está
// lleno, esa llegada se salta y se sortea otro intervalo.
func (g *Game) tickArrivalsLocked(dt time.Duration) {
	g.nextArrival -= dt
	if g.nextArrival > 0 {
		return
	}
	g.nextArrival = arrivalInterval(g.rng, g.clock.day, g.clock.minute())
	if len(g.patients) >= maxPatientsOnMap {
		return
	}
	g.spawnLocked()
}

// tickRushHourLocked revisa si la hora de juego actual está en una franja
// pico (§5.3) y, si recién entró a una, deja el aviso "¡Hora pico! Llegan
// más pacientes" (como los demás Notices). Snapshot.RushHour muestra el
// estado actual para que la interfaz lo dibuje.
func (g *Game) tickRushHourLocked() {
	_, peak := rushAt(g.clock.minute())
	if peak && !g.rushHour {
		g.noticeLocked("¡Hora pico! Llegan más pacientes")
	}
	g.rushHour = peak
}

// spawnLocked crea un paciente al azar en la calle. Todavía NO está admitido:
// AdmitPatient se llama cuando cruza la puerta.
func (g *Game) spawnLocked() {
	name, _ := randomName(g.rng) // _ : a los pacientes no se les pone título
	g.addArrivalLocked(name, randomAge(g.rng, patientMinAge, patientMaxAge), randomLevel(g.rng))
}

// addArrivalLocked pone en la calle a un paciente nuevo, con el ID siguiente
// y una apariencia al azar, a streetWalk de la puerta.
func (g *Game) addArrivalLocked(name string, age int, level hospital.NarcolepsyLevel) *patient {
	pt := &patient{
		p:     hospital.NewPatient(g.nextPatientIDLocked(), name, age, level),
		look:  RandomAppearance(g.rng),
		stage: Arriving,
		timer: streetWalk,
	}
	g.patients = append(g.patients, pt)
	return pt
}

// arrivalInterval sortea cuánto falta para la llegada siguiente: de 18 a
// 25 s el día 1; cada día, el 90 % del anterior, con un mínimo de 8 s. Por
// último, la franja horaria de minute (§5.3, rushBands en config.go) aprieta
// o afloja ese intervalo: el piso de arrivalFloor se respeta también después
// de aplicar el factor.
func arrivalInterval(rng *rand.Rand, day, minute int) time.Duration {
	d := randomDuration(rng, arrivalMin, arrivalMax)
	for i := 1; i < day; i++ {
		d = d * arrivalDailyPercent / 100
	}
	if d < arrivalFloor {
		d = arrivalFloor
	}
	factor, _ := rushAt(minute)
	d = time.Duration(float64(d) * factor)
	if d < arrivalFloor {
		d = arrivalFloor
	}
	return d
}

// randomLevel sortea el nivel de narcolepsia: Mild 40 %, Moderate 35 % y
// Severe 25 %.
func randomLevel(rng *rand.Rand) hospital.NarcolepsyLevel {
	switch n := rng.Intn(100); {
	case n < mildPercent:
		return hospital.Mild
	case n < mildPercent+moderatePercent:
		return hospital.Moderate
	default:
		return hospital.Severe
	}
}

// randomDuration sortea una duración entre lo y hi (incluidos), al
// milisegundo.
func randomDuration(rng *rand.Rand, lo, hi time.Duration) time.Duration {
	ms := rng.Int63n(int64((hi-lo)/time.Millisecond) + 1)
	return lo + time.Duration(ms)*time.Millisecond
}
