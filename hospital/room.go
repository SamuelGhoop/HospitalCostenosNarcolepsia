package hospital

import "fmt"

// RoomState indica si una habitación todavía tiene camas libres.
type RoomState int

// Estados de una habitación. Available es el valor cero: una habitación
// nueva arranca disponible.
const (
	Available RoomState = iota // le queda al menos una cama libre
	Occupied                   // todas sus camas están ocupadas
)

// String implementa fmt.Stringer para RoomState.
func (s RoomState) String() string {
	switch s {
	case Available:
		return "disponible"
	case Occupied:
		return "ocupada"
	default:
		return fmt.Sprintf("RoomState(%d)", int(s))
	}
}

// Room es una habitación del hospital con una o varias camas.
//
// state se podría calcular contando occupants, pero el enunciado lo pide como
// campo. Para que nunca contradiga a occupants, solo lo escribe updateState.
type Room struct {
	number    int
	capacity  int
	state     RoomState
	occupants []*Patient
}

// NewRoom crea una habitación vacía. Devuelve error si la capacidad no es
// válida: una habitación sin camas no tiene sentido.
func NewRoom(number, capacity int) (*Room, error) {
	if capacity < 1 {
		return nil, fmt.Errorf("%w: habitación %d con capacidad %d", ErrInvalidCapacity, number, capacity)
	}
	return &Room{
		number:    number,
		capacity:  capacity,
		occupants: make([]*Patient, 0, capacity), // slice vacío con espacio reservado
	}, nil
}

// Number devuelve el número de la habitación (101, 102…).
func (r *Room) Number() int { return r.number }

// Capacity devuelve cuántas camas tiene.
func (r *Room) Capacity() int { return r.capacity }

// State devuelve si está disponible u ocupada.
func (r *Room) State() RoomState { return r.state }

// IsAvailable dice si le queda al menos una cama libre.
func (r *Room) IsAvailable() bool { return r.state == Available }

// Occupants devuelve los pacientes que duermen aquí.
//
// Devuelve una COPIA del slice: si devolviera r.occupants, quien lo recibe
// podría escribir lista[0] = nil y cambiar la habitación por dentro,
// saltándose la encapsulación.
func (r *Room) Occupants() []*Patient {
	out := make([]*Patient, len(r.occupants))
	copy(out, r.occupants)
	return out
}

// Occupy acuesta en esta habitación a un paciente dormido en el pasillo.
//
// Room y Patient están en el mismo paquete, así que Room puede escribir los
// campos privados del paciente: se actualizan los dos lados en el mismo
// método para que la invariante cama/estado se cumpla siempre.
func (r *Room) Occupy(p *Patient) error {
	// switch sin variable = cadena de if/else if: entra al primer case verdadero.
	switch {
	case p == nil:
		return ErrNilPatient
	case p.state == Awake:
		return fmt.Errorf("%w: %s no se acuesta despierto", ErrNotAsleep, p)
	case p.state == AsleepInBed:
		return fmt.Errorf("%w: %s ya está en la habitación %d", ErrAlreadyInBed, p, p.room.number)
	case !r.IsAvailable():
		return fmt.Errorf("%w: habitación %d", ErrRoomFull, r.number)
	}

	r.occupants = append(r.occupants, p)
	r.updateState()

	p.room = r
	p.state = AsleepInBed
	p.currentLocation = fmt.Sprintf("habitación %d", r.number)
	return nil
}

// Release saca a un paciente de esta habitación y libera su cama.
// Si el paciente seguía dormido, vuelve a AsleepInHallway: lo sacaron de la
// cama pero nadie lo despertó. (WakeUp llama a Release y después lo despierta.)
func (r *Room) Release(p *Patient) error {
	if p == nil {
		return ErrNilPatient
	}
	for i, o := range r.occupants {
		if o == p { // se comparan punteros: ¿es exactamente el mismo paciente?
			// Quita el elemento i: une lo que hay antes de i con lo que hay después.
			r.occupants = append(r.occupants[:i], r.occupants[i+1:]...)
			r.updateState()

			p.room = nil
			if p.state == AsleepInBed {
				p.state = AsleepInHallway
			}
			return nil
		}
	}
	return fmt.Errorf("%w: %s no está en la habitación %d", ErrNotInRoom, p, r.number)
}

// updateState recalcula el estado según las camas ocupadas. Es el ÚNICO
// lugar que escribe r.state.
func (r *Room) updateState() {
	if len(r.occupants) >= r.capacity {
		r.state = Occupied
	} else {
		r.state = Available
	}
}
