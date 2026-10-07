package hospital

import (
	"fmt"
	"time"
)

// Patient es un paciente costeño con narcolepsia.
//
// Embebe Person (composición en vez de herencia): un Patient TIENE un Person
// adentro, y Go promueve sus métodos, así que p.Name() o fmt.Println(p)
// funcionan como si fueran de Patient. Los demás campos son privados del
// paquete: desde afuera solo se leen con getters y se cambian con métodos.
//
// Invariante: state == AsleepInBed  ⇔  room != nil  ⇔  p está en room.occupants.
// Solo Room.Occupy y Room.Release tocan room, para que nunca se rompa.
type Patient struct {
	Person // embebido por valor: sin nombre de campo, solo el tipo

	level           NarcolepsyLevel
	state           PatientState
	currentLocation string
	room            *Room     // habitación donde duerme; nil si no tiene cama
	asleepSince     time.Time // hora del último ataque; vacío si está despierto
}

// NewPatient crea un paciente despierto que acaba de llegar por recepción.
//
// Devuelve un puntero porque cada paciente existe UNA sola vez: el hospital,
// su habitación y los registros de episodios apuntan al mismo objeto. Si se
// pasaran copias, cada copia tendría su propio estado y se desincronizarían.
func NewPatient(id, name string, age int, level NarcolepsyLevel) *Patient {
	return &Patient{
		Person:          NewPerson(id, name, age),
		level:           level,
		currentLocation: "recepción",
		// state no se escribe: su valor cero ya es Awake.
	}
}

// SufferSleepAttack duerme al paciente justo donde está: queda
// AsleepInHallway (todavía sin cama) y se guarda la hora del ataque.
//
// Receiver de puntero (*Patient): el método modifica el paciente ORIGINAL.
// Con receiver de valor modificaría una copia y el cambio se perdería.
func (p *Patient) SufferSleepAttack(location string) error {
	if p.state != Awake {
		// %w envuelve el centinela (errors.Is lo encuentra) y %s agrega
		// el paciente usando su String() promovido de Person.
		return fmt.Errorf("%w: %s", ErrAlreadyAsleep, p)
	}
	p.state = AsleepInHallway
	p.currentLocation = location
	p.asleepSince = time.Now()
	return nil
}

// WakeUp despierta al paciente. Si estaba en una cama, primero le pide a su
// habitación que lo saque, para que la cama quede libre para otro.
func (p *Patient) WakeUp() error {
	if p.state == Awake {
		return fmt.Errorf("%w: %s", ErrAlreadyAwake, p)
	}
	if p.room != nil {
		if err := p.room.Release(p); err != nil {
			return err
		}
	}
	p.state = Awake
	p.asleepSince = time.Time{} // valor cero de time.Time = "sin hora"
	return nil
}

// Level devuelve el nivel de narcolepsia.
func (p *Patient) Level() NarcolepsyLevel { return p.level }

// State devuelve el estado actual (despierto, en el pasillo o en cama).
func (p *Patient) State() PatientState { return p.state }

// Location devuelve dónde está el paciente ahora: donde le dio el último
// ataque, o "habitación N" si ya lo acostaron.
func (p *Patient) Location() string { return p.currentLocation }

// Room devuelve la habitación donde duerme, o nil si no tiene cama.
func (p *Patient) Room() *Room { return p.room }

// AsleepSince devuelve la hora del último ataque, o la hora cero
// (IsZero() == true) si está despierto.
func (p *Patient) AsleepSince() time.Time { return p.asleepSince }

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
