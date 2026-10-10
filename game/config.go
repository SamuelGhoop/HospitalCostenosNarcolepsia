package game

import "time"

// Constantes de balance del juego (GAME_DESIGN §5). Viven todas aquí para
// poder ajustarlas sin tocar la lógica.
const (
	// El turno va de las 08:00 a las 20:00 en hora de juego (minutos desde
	// la medianoche) y dura shiftDuration de tiempo real: 1 s real = 2,4 min.
	shiftStart    = 8 * 60
	shiftEnd      = 20 * 60
	shiftDuration = 300 * time.Second

	// tickInterval: cada cuánto la goroutine motor llama Tick (§5.1).
	tickInterval = 100 * time.Millisecond
)
