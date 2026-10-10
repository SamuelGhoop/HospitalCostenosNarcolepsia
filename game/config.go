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

// Llegadas de pacientes (§5.3).
const (
	arrivalMin          = 18 * time.Second // intervalo del día 1: de arrivalMin…
	arrivalMax          = 25 * time.Second // …a arrivalMax
	arrivalDailyPercent = 90               // cada día el intervalo es el 90 % del anterior
	arrivalFloor        = 8 * time.Second  // pero nunca menos de esto
	maxPatientsOnMap    = 10
	mildPercent         = 40 // Mild 40 %, Moderate 35 %; el resto (25 %) es Severe
	moderatePercent     = 35
	patientMinAge       = 18
	patientMaxAge       = 80
	streetWalk          = 4 * time.Second // lo que camina por la acera hasta la puerta
	doorWindow          = 1 * time.Second // la puerta se abre cuando alguien está a esto de cruzarla
)

// Comportamiento del paciente (§5.4).
const (
	mildAwakeMin     = 60 * time.Second // tiempo despierto antes del ataque, según el nivel
	mildAwakeMax     = 90 * time.Second
	moderateAwakeMin = 35 * time.Second
	moderateAwakeMax = 55 * time.Second
	severeAwakeMin   = 15 * time.Second
	severeAwakeMax   = 30 * time.Second
	drowsyDuration   = 2 * time.Second  // el cabeceo antes de desplomarse
	wanderMin        = 10 * time.Second // despierto, cambia de zona cada wanderMin…
	wanderMax        = 20 * time.Second // …a wanderMax
	exitWalk         = 4 * time.Second  // lo que tarda en llegar a la puerta cuando se va

	mildSleep         = 15 * time.Second // duración del sueño en cama, según el nivel…
	moderateSleep     = 22 * time.Second
	severeSleep       = 30 * time.Second
	sleepSkillPercent = 8 // …con −8 % por punto de pericia del médico que lo revisó

	mildSleepsToDischarge     = 1 // sueños completos antes del alta, según el nivel
	moderateSleepsToDischarge = 2
	severeSleepsToDischarge   = 3
)

// Apariencia (§7): cuántos valores tiene cada paleta. La interfaz decide qué
// color es cada índice.
const (
	skinTones   = 4 // tonos de piel
	shirtColors = 6 // colores de camisa
	hairColors  = 4 // negro, castaño oscuro, castaño claro y canoso
)

// maxNotices: cuántos avisos (errores del modelo, altas, eventos) guarda la
// partida para la interfaz; los más viejos se descartan.
const maxNotices = 5
