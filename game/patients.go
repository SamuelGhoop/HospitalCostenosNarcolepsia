package game

import (
	"fmt"
	"math/rand"
	"slices"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// Stage es el estado de un paciente en el modo juego (GAME_DESIGN §3.4).
//
// Es una capa ENCIMA del estado del modelo (hospital.PatientState), que solo
// distingue despierto, dormido en el pasillo y dormido en cama. El juego
// necesita más detalle: por ejemplo, el desplomado que nadie ha recogido
// sigue Awake en el modelo, porque el hospital todavía no se ha enterado.
type Stage int

const (
	Arriving       Stage = iota // camina por la acera hacia la puerta (todavía no está admitido)
	Wandering                   // despierto, deambula por el hospital
	Drowsy                      // cabecea: en 2 s se desploma
	Collapsed                   // se desplomó y nadie lo ha recogido (en el modelo, Awake)
	AwaitingReview              // recogido y en cama, sin revisar (AsleepInBed)
	InHallway                   // recogido pero sin cama (AsleepInHallway)
	InBed                       // revisado por un médico (AsleepInBed)
	Leaving                     // camina hacia la puerta para irse
)

// String implementa fmt.Stringer para Stage.
func (s Stage) String() string {
	switch s {
	case Arriving:
		return "llegando"
	case Wandering:
		return "deambulando"
	case Drowsy:
		return "cabeceando"
	case Collapsed:
		return "desplomado"
	case AwaitingReview:
		return "esperando revisión"
	case InHallway:
		return "en el pasillo"
	case InBed:
		return "en cama"
	case Leaving:
		return "saliendo"
	default:
		return fmt.Sprintf("Stage(%d)", int(s))
	}
}

// patient es un paciente del modo juego: el *hospital.Patient del modelo más
// lo que solo le importa al juego.
type patient struct {
	p          *hospital.Patient
	look       Appearance
	stage      Stage
	timer      time.Duration // lo que le falta a la etapa actual (llegar a la puerta, el ataque, el cabeceo…)
	zone       Zone          // dónde está (al desplomarse, la ubicación que recibe RegisterEpisode)
	wanderLeft time.Duration // deambulando: cuánto falta para cambiar de zona
	waited     time.Duration // cronómetro de espera del episodio: del desplome a la revisión
	sleeps     int           // sueños completos (revisado y despertado solo), para el alta
	penalties  int           // penalizaciones por espera ya cobradas en este episodio
	reviewedBy *StaffMember  // el médico que lo revisó; nil si nadie
	gone       bool          // ya cruzó la puerta de salida: se quita del mapa
}

// view copia los datos del paciente a una vista (solo valores).
func (pt *patient) view() GamePatientView {
	v := GamePatientView{PatientView: patientViewOf(pt.p), Stage: pt.stage, Zone: pt.zone, Look: pt.look, Waited: pt.waited}
	if pt.reviewedBy != nil {
		v.ReviewedBy = pt.reviewedBy.ID()
	}
	return v
}

// countdown descuenta dt del cronómetro de la etapa actual y dice si ya se
// cumplió.
func (pt *patient) countdown(dt time.Duration) bool {
	pt.timer -= dt
	return pt.timer <= 0
}

// nearTheDoor dice si el paciente está a doorWindow o menos de cruzar la
// puerta, entrando o saliendo: ahí la puerta se abre sola (§5.3).
func (pt *patient) nearTheDoor() bool {
	return (pt.stage == Arriving || pt.stage == Leaving) && pt.timer <= doorWindow
}

// tickPatientsLocked avanza dt la máquina de estados de cada paciente
// (§5.4). Los que ya salieron por la puerta se quitan del mapa.
func (g *Game) tickPatientsLocked(dt time.Duration) {
	for _, pt := range g.patients {
		switch pt.stage {
		case Arriving:
			if pt.countdown(dt) {
				g.admitLocked(pt)
			}
		case Wandering: // timer = tiempo despierto que le queda
			pt.wanderLeft -= dt
			if pt.wanderLeft <= 0 {
				pt.zone = nextZone(g.rng, pt.zone)
				pt.wanderLeft = randomDuration(g.rng, wanderMin, wanderMax)
			}
			if pt.countdown(dt) {
				pt.stage, pt.timer = Drowsy, drowsyDuration
			}
		case Drowsy:
			if pt.countdown(dt) {
				pt.stage, pt.waited, pt.penalties = Collapsed, 0, 0 // arranca el cronómetro de espera
			}
		case Collapsed, InHallway, AwaitingReview:
			pt.waited += dt // corre hasta la revisión, sin volver a empezar
			g.applyWaitPenaltiesLocked(pt)
			if pt.waited >= angryAfter {
				g.leaveAngryLocked(pt)
			}
		case InBed: // timer = sueño que le queda
			if pt.countdown(dt) {
				g.wakeUpLocked(pt)
			}
		case Leaving:
			if pt.countdown(dt) {
				pt.gone = true // en el modelo sigue, como Awake
			}
		}
	}
	// slices.DeleteFunc (Go 1.21) quita del slice los que cumplen la condición.
	g.patients = slices.DeleteFunc(g.patients, func(pt *patient) bool { return pt.gone })
}

// admitLocked: el paciente cruza la puerta. Recién aquí el hospital lo
// admite y empieza a correr su tiempo despierto, nunca en la acera.
func (g *Game) admitLocked(pt *patient) {
	if err := g.h.AdmitPatient(pt.p); err != nil {
		// No debería pasar (los IDs son consecutivos); si pasa, se avisa y
		// el paciente se devuelve por donde vino.
		g.noticeLocked(fmt.Sprintf("%s no pudo entrar: %v", pt.p, err))
		g.leaveLocked(pt, "") // sin nota: nunca entró al modelo, no sale en el reporte
		return
	}
	pt.stage, pt.zone = Wandering, Lobby
	pt.timer = awakeDuration(g.rng, pt.p.Level())
	pt.wanderLeft = randomDuration(g.rng, wanderMin, wanderMax)
}

// wakeUpLocked: se cumplió el sueño. El juego llama WakePatient, que
// despierta al paciente y libera la cama. Si ya completó los sueños de su
// nivel, recibe el alta y sale; si no, vuelve a deambular.
func (g *Game) wakeUpLocked(pt *patient) {
	if err := g.h.WakePatient(pt.p); err != nil {
		// No debería pasar (en el modelo está dormido en cama); si pasa,
		// se avisa y el juego lo trata como despierto igual.
		g.noticeLocked(fmt.Sprintf("%s no se pudo despertar: %v", pt.p, err))
	}
	pt.sleeps++
	pt.reviewedBy = nil
	if pt.sleeps >= sleepsBeforeDischarge(pt.p.Level()) {
		g.noticeLocked(fmt.Sprintf("%s se fue de alta", pt.p))
		g.money += dischargeFee
		g.score += dischargePoints
		g.changeReputationLocked(+dischargeBonus)
		g.leaveLocked(pt, "(alta)")
		return
	}
	g.wanderAgainLocked(pt, awakeDuration(g.rng, pt.p.Level()))
}

// wanderAgainLocked: el paciente se levantó de la cama y vuelve a deambular,
// desde una zona al azar, con awake de tiempo despierto antes del próximo
// ataque.
func (g *Game) wanderAgainLocked(pt *patient, awake time.Duration) {
	pt.reviewedBy = nil // el que lo revisó ya no cuenta: el próximo episodio es otro
	pt.stage, pt.zone = Wandering, wanderZones[g.rng.Intn(len(wanderZones))]
	pt.timer = awake
	pt.wanderLeft = randomDuration(g.rng, wanderMin, wanderMax)
}

// WakeEarly es la acción DESPERTAR (§5.7): despierta antes de tiempo a un
// paciente revisado que duerme en cama, para liberar su cama al instante.
// Cuesta −0,25 ★. No cuenta como sueño completo para el alta, y el paciente
// vuelve a dormirse en la mitad del tiempo normal.
func (g *Game) WakeEarly(patientID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.over != NotOver {
		return fmt.Errorf("%w: %v", ErrGameOver, g.over)
	}
	pt, err := g.findPatientLocked(patientID)
	if err != nil {
		return err
	}
	if pt.stage != InBed { // el sueño solo corre después de la revisión
		return fmt.Errorf("%w: %s está %v", ErrNotInBed, pt.p, pt.stage)
	}
	if err := g.h.WakePatient(pt.p); err != nil { // lo despierta y libera la cama
		return err
	}
	g.changeReputationLocked(-wakeEarlyPenalty)
	g.wanderAgainLocked(pt, awakeDuration(g.rng, pt.p.Level())/2)
	return nil
}

// leaveAngryLocked: pasaron 45 s desde el desplome y nadie lo revisó. Se
// despierta solo y se va enojado (§5.4).
func (g *Game) leaveAngryLocked(pt *patient) {
	// Si estaba desplomado, el modelo nunca se enteró: sigue Awake y no hay
	// episodio, así que no hay nada que decirle. Si estaba en el pasillo o
	// esperando revisión, WakePatient lo despierta y libera la cama si tenía.
	if pt.stage != Collapsed {
		if err := g.h.WakePatient(pt.p); err != nil {
			g.noticeLocked(fmt.Sprintf("%s no se pudo despertar: %v", pt.p, err))
		}
	}
	g.cancelJobsForLocked(pt)
	g.changeReputationLocked(-angryPenalty)
	g.noticeLocked(fmt.Sprintf("%s se fue enojado", pt.p))
	g.leaveLocked(pt, "(se fue enojado)")
}

// leaveLocked manda al paciente hacia la puerta para irse del mapa. note es
// cómo sale en el Shift Report, por ejemplo "(alta)". Mientras sale no
// tiene tiempo despierto que correr, así que no vuelve a tener ataque.
func (g *Game) leaveLocked(pt *patient, note string) {
	pt.stage, pt.timer = Leaving, exitWalk
	g.notes[pt.p.ID()] = note
}

// sleepDuration es cuánto duerme en cama: 15/22/30 s según el nivel, con
// −8 % por punto de pericia del médico que lo revisó. Aritmética entera de
// time.Duration, como el reloj.
func sleepDuration(level hospital.NarcolepsyLevel, skill int) time.Duration {
	base := severeSleep
	switch level {
	case hospital.Mild:
		base = mildSleep
	case hospital.Moderate:
		base = moderateSleep
	}
	return base * time.Duration(100-sleepSkillPercent*skill) / 100
}

// sleepsBeforeDischarge es cuántos sueños completos necesita para el alta.
func sleepsBeforeDischarge(level hospital.NarcolepsyLevel) int {
	switch level {
	case hospital.Mild:
		return mildSleepsToDischarge
	case hospital.Moderate:
		return moderateSleepsToDischarge
	default:
		return severeSleepsToDischarge
	}
}

// nextZone sortea la zona siguiente para deambular, distinta de la actual.
func nextZone(rng *rand.Rand, current Zone) Zone {
	var others []Zone
	for _, z := range wanderZones {
		if z != current {
			others = append(others, z)
		}
	}
	return others[rng.Intn(len(others))]
}

// awakeDuration sortea cuánto aguanta despierto antes del ataque, según el
// nivel: Mild 60–90 s, Moderate 35–55 s, Severe 15–30 s.
func awakeDuration(rng *rand.Rand, level hospital.NarcolepsyLevel) time.Duration {
	switch level {
	case hospital.Mild:
		return randomDuration(rng, mildAwakeMin, mildAwakeMax)
	case hospital.Moderate:
		return randomDuration(rng, moderateAwakeMin, moderateAwakeMax)
	default:
		return randomDuration(rng, severeAwakeMin, severeAwakeMax)
	}
}

// nextPatientIDLocked devuelve el ID del paciente siguiente: P-001, P-002…
// Son consecutivos porque AdmitPatient rechaza IDs repetidos.
func (g *Game) nextPatientIDLocked() string {
	g.patientCount++
	return fmt.Sprintf("P-%03d", g.patientCount)
}

// findPatientLocked busca un paciente del juego por su ID.
func (g *Game) findPatientLocked(id string) (*patient, error) {
	for _, pt := range g.patients {
		if pt.p.ID() == id {
			return pt, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrUnknownPatient, id)
}
