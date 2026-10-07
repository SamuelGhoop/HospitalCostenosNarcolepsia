package hospital

import "fmt"

// NarcolepsyLevel es el nivel de narcolepsia de un paciente.
//
// Es un tipo propio basado en int, no un int suelto ni un string: el
// compilador no deja asignar un PatientState donde va un NarcolepsyLevel,
// y un error de dedo como "Sever" no compila (con strings sí compilaría).
type NarcolepsyLevel int

// Niveles de narcolepsia. iota es un contador que vale 0, 1, 2… en cada
// línea del bloque const; las líneas sin "= iota" repiten el tipo y la
// fórmula de la anterior.
const (
	Mild     NarcolepsyLevel = iota // leve (0, el valor cero)
	Moderate                        // moderada (1)
	Severe                          // severa (2)
)

// String implementa fmt.Stringer: fmt.Println(nivel) imprime "severa"
// en vez del número 2.
func (l NarcolepsyLevel) String() string {
	switch l {
	case Mild:
		return "leve"
	case Moderate:
		return "moderada"
	case Severe:
		return "severa"
	default:
		// Se convierte a int a propósito: con %v sobre l, Sprintf volvería
		// a llamar a String y el programa entraría en recursión infinita.
		return fmt.Sprintf("NarcolepsyLevel(%d)", int(l))
	}
}

// PatientState es dónde y cómo está un paciente en este momento.
type PatientState int

// Estados del paciente. Awake va primero para que sea el valor cero:
// un paciente recién creado arranca despierto sin inicializarlo a mano.
const (
	Awake           PatientState = iota // despierto
	AsleepInHallway                     // dormido donde le dio el ataque, sin cama
	AsleepInBed                         // dormido en una habitación
)

// String implementa fmt.Stringer para PatientState.
func (s PatientState) String() string {
	switch s {
	case Awake:
		return "despierto"
	case AsleepInHallway:
		return "dormido en el pasillo"
	case AsleepInBed:
		return "dormido en cama"
	default:
		return fmt.Sprintf("PatientState(%d)", int(s))
	}
}
