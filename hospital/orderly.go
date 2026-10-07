package hospital

import "fmt"

// Orderly es un camillero: el segundo tipo que cumple Attender.
//
// Puede atender a un paciente dormido y llevarlo a su cama, pero NO puede
// diagnosticar: no tiene método DiagnosePatient, así que el compilador
// rechaza cualquier intento de llamarlo.
type Orderly struct {
	Person

	transfers int // cuántos pacientes ha llevado a una cama
}

// NewOrderly crea un camillero sin traslados.
func NewOrderly(id, name string, age int) *Orderly {
	return &Orderly{Person: NewPerson(id, name, age)}
}

// Transfers devuelve cuántos pacientes ha llevado a una cama.
func (o *Orderly) Transfers() int { return o.transfers }

// Attend cumple la interfaz Attender.
//
//   - Si el paciente ya tiene cama asignada, el camillero lo lleva: cuenta
//     un traslado y el registro queda con esa habitación.
//   - Si no hay cama, no hay a dónde llevarlo: lo deja vigilado en el pasillo,
//     el registro queda sin habitación (room nil) y no suma traslado.
func (o *Orderly) Attend(p *Patient, location string) (EpisodeRecord, error) {
	if p == nil {
		return EpisodeRecord{}, ErrNilPatient
	}
	if p.state == Awake {
		return EpisodeRecord{}, fmt.Errorf("%w: %s está despierto, no hay a quién trasladar", ErrNotAsleep, p)
	}
	if p.room != nil {
		o.transfers++
	}
	return newEpisodeRecord(p, o, location), nil
}

// IsAvailable cumple la interfaz Attender. El camillero siempre está de
// turno: un traslado es rápido y no le deja pacientes a cargo.
func (o *Orderly) IsAvailable() bool { return true }
