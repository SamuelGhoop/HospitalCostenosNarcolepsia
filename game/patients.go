package game

import (
	"fmt"

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
	Collapsed      Stage = iota // se desplomó y nadie lo ha recogido (en el modelo, Awake)
	AwaitingReview              // recogido y en cama, sin revisar (AsleepInBed)
	InHallway                   // recogido pero sin cama (AsleepInHallway)
	InBed                       // revisado por un médico (AsleepInBed)
)

// String implementa fmt.Stringer para Stage.
func (s Stage) String() string {
	switch s {
	case Collapsed:
		return "desplomado"
	case AwaitingReview:
		return "esperando revisión"
	case InHallway:
		return "en el pasillo"
	case InBed:
		return "en cama"
	default:
		return fmt.Sprintf("Stage(%d)", int(s))
	}
}

// patient es un paciente del modo juego: el *hospital.Patient del modelo más
// lo que solo le importa al juego.
type patient struct {
	p          *hospital.Patient
	stage      Stage
	location   string       // dónde se desplomó: la ubicación que recibe RegisterEpisode
	reviewedBy *StaffMember // el médico que lo revisó; nil si nadie
}

// view copia los datos del paciente a una vista (solo valores).
func (pt *patient) view() GamePatientView {
	v := GamePatientView{PatientView: patientViewOf(pt.p), Stage: pt.stage}
	if pt.reviewedBy != nil {
		v.ReviewedBy = pt.reviewedBy.ID()
	}
	return v
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
