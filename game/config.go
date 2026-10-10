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

// Personal (§5.2 y §7).
const (
	startSpeed  = 3  // rapidez del médico y del camillero iniciales
	startSkill  = 3  // pericia del médico inicial
	staffMinAge = 25 // edad del personal, de staffMinAge a staffMaxAge
	staffMaxAge = 60
	maxStat     = 5 // la rapidez y la pericia van de 1 a maxStat (§6)
)

// Despacho (§5.5).
const (
	walkBase      = 8               // el personal camina (walkBase − rapidez) segundos
	carryDuration = 2 * time.Second // lo que tarda en llevar al paciente a la cama
)

// maxNotices: cuántos avisos (errores del modelo, altas, eventos) guarda la
// partida para la interfaz; los más viejos se descartan.
const maxNotices = 5
