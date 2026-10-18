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
	maxStaff    = 6 // tope de personal contratado (§6); no confundir con maxStat
)

// statWeights son los pesos (de 100) con que se sortean la rapidez y la
// pericia de un candidato, de 1 a maxStat: más probabilidad de salir 2 o 3
// (§6). Es un var y no una const porque Go no permite arreglos literales en
// una const; se usa como solo-lectura igual.
var statWeights = [maxStat]int{15, 30, 30, 15, 10}

// Contratación (§6 y §5.8): 3 candidatos por día, ≈ 66 % médicos.
const (
	candidatesPerDay    = 3
	doctorChancePercent = 66 // el resto, camilleros
)

// doctorSalaryBase, doctorSalaryPerStat, orderlySalaryBase y
// orderlySalaryPerSpeed son la fórmula de la nómina (§5.8):
// médico = base + perStat × (rapidez + pericia); camillero = base + perSpeed × rapidez.
const (
	doctorSalaryBase      = 250
	doctorSalaryPerStat   = 50
	orderlySalaryBase     = 120
	orderlySalaryPerSpeed = 60
)

// doctorSpecialties: el modelo le pide una especialidad a cada médico
// (hospital.NewDoctor). El médico inicial usa "Medicina del sueño"
// (doctorSpecialty, en staff.go); los candidatos de la bolsa de empleo
// sortean una de esta lista corta, inventada para la demo.
var doctorSpecialties = []string{"Medicina del sueño", "Neurología", "Medicina interna", "Medicina general"}

// Despacho (§5.5): el personal camina (walkBase − rapidez) segundos hasta
// donde lo mandan, y otro tanto para llevar al paciente a la cama en camilla.
const walkBase = 8

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

// rushBand describe una franja horaria del turno (minutos desde la
// medianoche, [from, to)) y el factor que multiplica el intervalo de
// llegadas sorteado dentro de ella (§5.3). Pedido de Samuel del 2026-10-17
// (§12.8): la jugabilidad estaba "un poco aburrida", así que el ritmo de
// llegadas ahora varía como en Diner Dash, con dos horas pico.
type rushBand struct {
	from, to int     // minutos desde la medianoche
	factor   float64 // multiplica el intervalo sorteado; menor a 1 = llegan más seguido
	peak     bool    // true si esta franja es "hora pico" (aviso y Snapshot.RushHour)
}

// rushBands cubre las 12 horas del turno (08:00–20:00) sin huecos ni
// traslapes, de la más temprana a la más tardía. Para ajustar el ritmo basta
// cambiar los minutos o el factor de una fila; no hay que tocar la lógica.
var rushBands = []rushBand{
	{8 * 60, 10 * 60, 1.3, false},     // 08:00–10:00 tranquilo: la gente recién va llegando
	{10 * 60, 12 * 60, 0.5, true},     // 10:00–12:00 PICO de media mañana
	{12 * 60, 15 * 60, 1.0, false},    // 12:00–15:00 normal
	{15 * 60, 17*60 + 30, 0.5, true},  // 15:00–17:30 PICO de media tarde
	{17*60 + 30, 20 * 60, 0.9, false}, // 17:30–20:00 bajando el ritmo antes de cerrar
}

// rushAt devuelve el factor y si es hora pico para minute (hora de juego, en
// minutos desde la medianoche). Fuera de las franjas conocidas (no debería
// pasar dentro de un turno de 08:00 a 20:00) no cambia nada: factor 1, sin
// pico.
func rushAt(minute int) (factor float64, peak bool) {
	for _, b := range rushBands {
		if minute >= b.from && minute < b.to {
			return b.factor, b.peak
		}
	}
	return 1, false
}

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
	angryAfter       = 45 * time.Second // sin revisión desde el desplome, se despierta y se va enojado

	mildSleep         = 15 * time.Second // duración del sueño en cama, según el nivel…
	moderateSleep     = 22 * time.Second
	severeSleep       = 30 * time.Second
	sleepSkillPercent = 8 // …con −8 % por punto de pericia del médico que lo revisó

	mildSleepsToDischarge     = 1 // sueños completos antes del alta, según el nivel
	moderateSleepsToDischarge = 2
	severeSleepsToDischarge   = 3
)

// Economía (§5.2 y §5.8).
const (
	startMoney     = 2000             // plata con la que arranca la partida
	reviewFeeMax   = 450              // pago de la revisión si esperó hasta reviewFeeGrace…
	reviewFeeGrace = 15 * time.Second // (la espera, desde el desplome)
	reviewFeeDrop  = 15               // …después baja esto por cada segundo completo…
	reviewFeeMin   = 100              // …sin bajar de esto
	dischargeFee   = 200              // el alta paga aparte
)

// Reputación (§5.9), en centésimas de estrella: 300 = 3 ★.
//
// firstWaitPenaltyAt subió de 20 a 30 s (decisión de Samuel, §12.8: en la
// prueba del 16-10 "le bajaban las estrellas solas" sin darse cuenta; más
// margen es una de las tres correcciones que eligió, junto con el aviso con
// motivo de changeReputationLocked).
const (
	startReputation    = 300
	maxReputation      = 500
	firstWaitPenaltyAt = 30 * time.Second // esperando: la primera penalización a los 30 s…
	firstWaitPenalty   = 50               // …de −0,5 ★…
	waitPenaltyEvery   = 10 * time.Second // …y después una cada 10 s más…
	nextWaitPenalty    = 25               // …de −0,25 ★
	angryPenalty       = 50               // se va enojado: −0,5 ★
	quickPickUpWithin  = 10 * time.Second // recogido en menos de esto desde el desplome…
	quickPickUpBonus   = 10               // …+0,1 ★
	dischargeBonus     = 10               // alta: +0,1 ★
	wakeEarlyPenalty   = 25               // DESPERTAR antes de tiempo: −0,25 ★ (§5.7)
)

// maxReputationChanges: cuántos cambios de reputación recientes, con su
// motivo, guarda la partida para la interfaz (mismo criterio que maxNotices).
const maxReputationChanges = 5

// Puntaje (§5.10).
const (
	reviewPoints       = 100 // por cada revisión (la atención completa)
	dischargePoints    = 250 // por cada alta
	dayCompletedPoints = 500 // por cada día que llega a las 20:00
	finalMoneyDivisor  = 10  // al final: + plata ÷ 10…
	finalStarPoints    = 200 // …+ estrellas × 200
)

// Derrota (§5.10).
const collapseLimit = 5 // pacientes desplomados o en el pasillo a la vez para que el hospital colapse

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
