package game

import (
	"fmt"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// job es el trabajo que el jugador le encargó a alguien del personal.
type job struct {
	activity Activity      // qué está haciendo ahora
	patient  *patient      // a quién va o a quién lleva
	left     time.Duration // cuánto le falta a lo que está haciendo
}

// Dispatch es la decisión del jugador (GAME_DESIGN §5.5): manda a staffID a
// atender a patientID. El estado del paciente decide para qué:
//
//   - Collapsed: a recogerlo. Le toca a un camillero; un médico solo si no
//     hay ningún camillero libre.
//   - AwaitingReview: a revisarlo en la habitación. Solo un médico.
//
// La persona sale caminando; lo demás pasa en Tick, cuando llega.
func (g *Game) Dispatch(patientID, staffID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	pt, err := g.findPatientLocked(patientID)
	if err != nil {
		return err
	}
	s, err := g.findStaffLocked(staffID)
	if err != nil {
		return err
	}
	if s.job != nil {
		return fmt.Errorf("%w: %s %v", ErrStaffBusy, s.Name(), s.job.activity)
	}
	if g.someoneOnTheWayLocked(pt) {
		return fmt.Errorf("%w: %s", ErrAlreadyDispatched, pt.p)
	}

	switch pt.stage {
	case Collapsed:
		if s.isDoctor() && g.anyOrderlyFreeLocked() {
			return fmt.Errorf("%w (%s no puede recoger a %s)", ErrOrderlyAvailable, s.Name(), pt.p.Name())
		}
		s.job = &job{activity: GoingToPickUp, patient: pt, left: s.walkTime()}
	case AwaitingReview:
		if !s.isDoctor() {
			return fmt.Errorf("%w (%s es camillero)", ErrNotADoctor, s.Name())
		}
		s.job = &job{activity: GoingToReview, patient: pt, left: s.walkTime()}
	default:
		return fmt.Errorf("%w: %s está %v", ErrNothingToDispatch, pt.p, pt.stage)
	}
	return nil
}

// someoneOnTheWayLocked dice si alguien del personal ya va caminando hacia
// pt. El que lo lleva a la cama no cuenta: ya llegó, y mientras lo lleva se
// puede mandar al médico a revisarlo.
func (g *Game) someoneOnTheWayLocked(pt *patient) bool {
	for _, s := range g.staff {
		if s.job != nil && s.job.patient == pt && s.job.activity != Carrying {
			return true
		}
	}
	return false
}

// anyOrderlyFreeLocked dice si hay al menos un camillero sin trabajo.
func (g *Game) anyOrderlyFreeLocked() bool {
	for _, s := range g.staff {
		if !s.isDoctor() && s.job == nil {
			return true
		}
	}
	return false
}

// tickStaffLocked avanza dt el trabajo de cada miembro del personal. Al que
// termina lo que estaba haciendo le toca el paso siguiente.
func (g *Game) tickStaffLocked(dt time.Duration) {
	for _, s := range g.staff {
		if s.job == nil {
			continue
		}
		s.job.left -= dt
		if s.job.left > 0 {
			continue
		}
		switch s.job.activity {
		case GoingToPickUp:
			g.pickUpLocked(s, s.job.patient)
		case Carrying:
			s.job = nil // lo dejó en la cama: queda libre
		case GoingToReview:
			g.reviewLocked(s, s.job.patient)
		}
	}
}

// pickUpLocked: s llegó donde el paciente desplomado y lo carga. Aquí se
// hace el despacho "de guardia" y el modelo se entera del episodio.
func (g *Game) pickUpLocked(s *StaffMember, pt *patient) {
	s.job = nil
	if err := g.registerOnCallLocked(s, pt); err != nil {
		// No debería pasar (el paciente está admitido y despierto en el
		// modelo), pero si pasa no se pierde: el paciente sigue desplomado,
		// s queda libre y el error sale como aviso.
		g.noticeLocked(fmt.Sprintf("%s no pudo recoger a %s: %v", s.Name(), pt.p, err))
		return
	}

	// El episodio que acaba de crear RegisterEpisode es el último del
	// historial: con g.mu tomado nadie más puede registrar otro en medio.
	// Se guarda la hora del JUEGO para la bitácora y el Shift Report (§3.2).
	history := g.h.History()
	g.episodeTimes[history[len(history)-1].ID()] = g.clock.String()

	// Sin cama, RegisterEpisode no da error: el modelo lo deja dormido en el
	// pasillo. No hay a dónde llevarlo, así que s queda libre de una vez.
	if pt.p.Room() == nil {
		pt.stage = InHallway
		g.noticeLocked(fmt.Sprintf("%s se quedó en el pasillo: %v", pt.p, hospital.ErrNoRoomAvailable))
		return
	}
	pt.stage = AwaitingReview
	s.job = &job{activity: Carrying, patient: pt, left: carryDuration}
}

// reviewLocked: el médico s llegó a la habitación y revisa al paciente.
//
// Es SOLO lógica del juego: no llama nada del modelo. Tampoco
// DiagnosePatient, porque llena el cupo de 4 pacientes del médico y el
// modelo no lo libera nunca. Aquí arranca el sueño, que dura según la
// pericia de s. (F1.4 hace aquí el pago.)
func (g *Game) reviewLocked(s *StaffMember, pt *patient) {
	s.job = nil
	pt.stage = InBed
	pt.reviewedBy = s
	pt.timer = sleepDuration(pt.p.Level(), s.skill)
}

// registerOnCallLocked es el mecanismo "de guardia" (§3.2). Por un momento s
// es el ÚNICO disponible del hospital, así que el round-robin de
// RegisterEpisode solo lo puede elegir a él. El hospital sigue decidiendo
// "entre los disponibles"; el juego decide quién está disponible.
func (g *Game) registerOnCallLocked(s *StaffMember, pt *patient) error {
	s.onCall = true
	defer func() { s.onCall = false }() // se apaga aunque RegisterEpisode falle
	return g.h.RegisterEpisode(pt.p, pt.zone.String())
}
