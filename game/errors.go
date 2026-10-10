package game

import "errors"

// Errores centinela del paquete game. Igual que en hospital, quien llama los
// reconoce con errors.Is.
var (
	// ErrDemoFinished: se pidió otro paso cuando la demo ya mostró los 12.
	ErrDemoFinished = errors.New("la demostración ya terminó")

	// ErrDayNotOver: se pidió empezar el día siguiente antes de las 20:00.
	ErrDayNotOver = errors.New("el turno todavía no ha terminado")
)
