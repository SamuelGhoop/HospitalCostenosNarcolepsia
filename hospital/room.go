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
