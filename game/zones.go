package game

import "fmt"

// Zone es una zona del hospital por donde deambulan los pacientes
// (GAME_DESIGN §5.4). Es una constante tipada y no un string suelto: así la
// interfaz sabe a qué zona del mapa corresponde, y un error de dedo no
// compila.
type Zone int

const (
	Lobby     Zone = iota // la recepción, donde entra todo paciente
	Cafeteria             // cafetería
	Hallway1              // pasillo 1
	Hallway2              // pasillo 2
	Radiology             // radiología
)

// wanderZones son las zonas por donde deambula un paciente despierto.
var wanderZones = []Zone{Lobby, Cafeteria, Hallway1, Hallway2, Radiology}

// String es la ubicación que recibe RegisterEpisode y que queda en el
// historial del modelo ("recepción" es la ubicación con la que el modelo
// crea a todo paciente).
func (z Zone) String() string {
	switch z {
	case Lobby:
		return "recepción"
	case Cafeteria:
		return "cafetería"
	case Hallway1:
		return "pasillo 1"
	case Hallway2:
		return "pasillo 2"
	case Radiology:
		return "radiología"
	default:
		return fmt.Sprintf("Zone(%d)", int(z))
	}
}
