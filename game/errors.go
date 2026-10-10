package game

import "errors"

// Errores centinela del paquete game. Igual que en hospital, quien llama los
// reconoce con errors.Is.
var (
	// ErrDemoFinished: se pidió otro paso cuando la demo ya mostró los 12.
	ErrDemoFinished = errors.New("la demostración ya terminó")

	// ErrDayNotOver: se pidió empezar el día siguiente antes de las 20:00.
	ErrDayNotOver = errors.New("el turno todavía no ha terminado")

	// Errores de Dispatch (§5.5).
	ErrUnknownPatient = errors.New("no hay ningún paciente con ese ID en la partida")
	ErrUnknownStaff   = errors.New("no hay nadie del personal con ese ID")
	// ErrOrderlyAvailable: un médico solo recoge si no hay camillero libre.
	ErrOrderlyAvailable = errors.New("hay un camillero libre: recoger pacientes le toca a él")
	// ErrNotADoctor: revisar a un paciente le toca a un médico.
	ErrNotADoctor = errors.New("solo un médico puede revisar al paciente")

	ErrStaffBusy         = errors.New("ese miembro del personal está ocupado")
	ErrAlreadyDispatched = errors.New("ya alguien va en camino hacia ese paciente")
	ErrNothingToDispatch = errors.New("ese paciente no necesita que le manden a nadie")
)
