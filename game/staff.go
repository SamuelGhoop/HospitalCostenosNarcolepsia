package game

import (
	"fmt"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// doctorSpecialty es la especialidad de todos los médicos del modo juego.
const doctorSpecialty = "Medicina del sueño"

// StaffMember es un miembro del personal del modo juego (GAME_DESIGN §3.1).
//
// Es un DECORADOR: cumple la interfaz hospital.Attender envolviendo al
// Doctor u Orderly real (inner) y le cambia una sola cosa: IsAvailable. Como
// HireStaff acepta cualquier Attender, el hospital lo contrata sin saber que
// va envuelto, y el paquete hospital no se modifica.
//
// ⚠️ El hospital llama sus métodos desde adentro de RegisterEpisode, con su
// propio candado tomado y mientras Game ya tiene g.mu. Por eso NUNCA toman
// g.mu: sería un deadlock (§11).
type StaffMember struct {
	inner  hospital.Attender // el *hospital.Doctor o *hospital.Orderly real
	doctor *hospital.Doctor  // el mismo médico sin envolver (para MyEpisodes); nil si es camillero
	speed  int               // rapidez, de 1 a 5
	skill  int               // pericia, de 1 a 5; 0 en los camilleros
	job    *job              // lo que está haciendo; nil = libre
	onCall bool              // "de guardia": true solo durante el despacho que eligió el jugador
}

// Verificación en tiempo de compilación, como en hospital/attender.go: si a
// StaffMember le faltara un método de Attender, el paquete no compilaría.
var _ hospital.Attender = (*StaffMember)(nil)

// newDoctorMember envuelve a un médico real.
func newDoctorMember(d *hospital.Doctor, speed, skill int) *StaffMember {
	return &StaffMember{inner: d, doctor: d, speed: speed, skill: skill}
}

// newOrderlyMember envuelve a un camillero real.
func newOrderlyMember(o *hospital.Orderly, speed int) *StaffMember {
	return &StaffMember{inner: o, speed: speed}
}

// ID y Name se delegan al Doctor u Orderly real, que los tiene promovidos de
// Person.
func (s *StaffMember) ID() string   { return s.inner.ID() }
func (s *StaffMember) Name() string { return s.inner.Name() }

// Attend también se delega: el registro del episodio lo crea el Doctor u
// Orderly real. Así el historial muestra a la persona real y, si es médico,
// el episodio queda en su MyEpisodes (consulta 5.2).
func (s *StaffMember) Attend(p *hospital.Patient, location string) (hospital.EpisodeRecord, error) {
	return s.inner.Attend(p, location)
}

// IsAvailable es lo ÚNICO que cambia el decorador: solo está disponible
// mientras está "de guardia" (§3.2), así que el round-robin de
// RegisterEpisode solo puede elegir al que mandó el jugador. Tampoco mira el
// cupo de 4 pacientes del Doctor real.
func (s *StaffMember) IsAvailable() bool { return s.onCall }

// isDoctor dice si es médico (false = camillero).
func (s *StaffMember) isDoctor() bool { return s.doctor != nil }

// walkTime es lo que tarda en llegar a donde lo mandan: (8 − rapidez) s,
// de 3 a 7 s (§5.5).
func (s *StaffMember) walkTime() time.Duration {
	return time.Duration(walkBase-s.speed) * time.Second
}

// view copia los datos del miembro del personal a una vista (solo valores).
func (s *StaffMember) view() StaffMemberView {
	v := StaffMemberView{
		StaffView: StaffView{ID: s.ID(), Name: s.Name(), IsDoctor: s.isDoctor()},
		Speed:     s.speed,
		Skill:     s.skill,
	}
	if s.job != nil { // sin trabajo, Activity queda en su valor cero: Free
		v.Activity = s.job.activity
		v.PatientID = s.job.patient.p.ID()
	}
	return v
}

// Activity es lo que está haciendo un miembro del personal.
type Activity int

const (
	Free            Activity = iota // libre: se le puede despachar
	GoingToPickUp                   // camina hasta el paciente desplomado
	Carrying                        // lo lleva a la cama
	GoingToReview                   // camina hasta la habitación a revisarlo
	GoingToTransfer                 // camina hasta el paciente del pasillo para llevarlo a una cama
)

// String implementa fmt.Stringer para Activity.
func (a Activity) String() string {
	switch a {
	case Free:
		return "libre"
	case GoingToPickUp:
		return "va a recogerlo"
	case Carrying:
		return "lo lleva a la cama"
	case GoingToReview:
		return "va a revisarlo"
	case GoingToTransfer:
		return "va a llevarlo a una cama"
	default:
		return fmt.Sprintf("Activity(%d)", int(a))
	}
}

// findStaffLocked busca a un miembro del personal por su ID.
func (g *Game) findStaffLocked(id string) (*StaffMember, error) {
	for _, s := range g.staff {
		if s.ID() == id {
			return s, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrUnknownStaff, id)
}

// ─── Contratación ──────────────────────────────────────────────────────

// newDoctorLocked crea un médico al azar, envuelto en StaffMember, con el ID
// siguiente (D-01, D-02…). Todavía no lo contrata.
func (g *Game) newDoctorLocked(speed, skill int) *StaffMember {
	g.doctorCount++
	d := hospital.NewDoctor(fmt.Sprintf("D-%02d", g.doctorCount), doctorName(g.rng),
		randomAge(g.rng, staffMinAge, staffMaxAge), doctorSpecialty)
	return newDoctorMember(d, speed, skill)
}

// newOrderlyLocked crea un camillero al azar (C-01, C-02…), envuelto.
func (g *Game) newOrderlyLocked(speed int) *StaffMember {
	g.orderlyCount++
	name, _ := randomName(g.rng) // _ : al camillero no se le pone título
	o := hospital.NewOrderly(fmt.Sprintf("C-%02d", g.orderlyCount), name,
		randomAge(g.rng, staffMinAge, staffMaxAge))
	return newOrderlyMember(o, speed)
}

// hireLocked contrata a s en el hospital y lo agrega a la lista de Game.
//
// Todo el personal del juego entra por aquí y siempre envuelto (invariante
// de §3.1). Game guarda su propia lista porque HireStaff no registra a los
// médicos envueltos como médicos: no salen en h.Doctors().
func (g *Game) hireLocked(s *StaffMember) error {
	if err := g.h.HireStaff(s); err != nil {
		return err
	}
	g.staff = append(g.staff, s)
	return nil
}
