package game

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// Game es una partida del modo juego (GAME_DESIGN §5).
//
// # Concurrencia (GAME_DESIGN §11)
//
// Dos goroutines tocan una partida: la goroutine motor (Start), que llama
// Tick cada 100 ms, y la de la interfaz, que llama Snapshot y las acciones
// del jugador. Por eso todo el estado, incluido el hospital, se protege con
// un solo candado: g.mu.
//
// Igual que en el paquete hospital, el sync.Mutex NO es reentrante: los
// métodos públicos toman g.mu y delegan en helpers que terminan en
// "Locked", que asumen que el candado ya está tomado. Así un método nunca
// pide el candado dos veces (eso sería un deadlock).
type Game struct {
	mu sync.Mutex

	rng    *rand.Rand // azar inyectado; solo con g.mu tomado
	looks  *rand.Rand // azar cosmético (la apariencia del personal), aparte para no cambiar la secuencia de rng
	h      *hospital.Hospital
	clock  clock
	paused bool

	staff                     []*StaffMember    // todo el personal, en orden de contratación
	doctorCount, orderlyCount int               // para los IDs: D-01, D-02… y C-01, C-02…
	patients                  []*patient        // los pacientes de la partida, en orden de llegada
	patientCount              int               // para los IDs: P-001, P-002…
	nextArrival               time.Duration     // cuánto falta para que aparezca el paciente siguiente
	notes                     map[string]string // ID del paciente → cómo sale en el Shift Report: "(alta)"
	lastAssignErr             string            // último error de AssignRoom (consulta 5.3); "" si no hubo
	money                     int               // plata del hospital, en pesos
	reputation                int               // en centésimas de estrella: 300 = 3 ★ (§5.9)
	over                      GameOver          // NotOver mientras la partida sigue
	score                     int               // puntaje acumulado (§5.10)
	episodeTimes              map[string]string // ID del episodio → hora del juego ("14:15"), §3.2
	notices                   []string          // últimos avisos para la interfaz, del más viejo al más nuevo

	// Contratación y nómina (§5.8 y §6, F4).
	candidates         []*candidate // los candidatesPerDay candidatos de la bolsa de empleo de hoy
	dayStartMoney      int          // plata al empezar el día de hoy, para el Shift Report diario
	dayReviewIncome    int          // ingresos por revisiones acumulados hoy
	dayDischargeIncome int          // ingresos por altas acumulados hoy
	dayHires           []HireLine   // contrataciones hechas hoy
	lastDayReport      DayReport    // el Shift Report del último día que cerró (ver DayReport())
}

// New crea una partida (GAME_DESIGN §5.2): un hospital con las habitaciones
// 101, 102 y 103, un médico y un camillero, el día 1 a las 08:00. rng es el
// azar de la partida; los tests pasan uno con semilla fija para que todo sea
// repetible. looks es el azar cosmético, solo para la apariencia del
// personal: va aparte para que sortear apariencias no cambie lo que sale de
// rng (los nombres, las llegadas, los ataques).
func New(rng, looks *rand.Rand) (*Game, error) {
	h := hospital.NewHospital("Hospital de los Costeños con Narcolepsia")
	for _, number := range []int{101, 102, 103} {
		if err := h.AddRoom(number, 1); err != nil {
			return nil, fmt.Errorf("creando la habitación %d: %w", number, err)
		}
	}
	g := &Game{rng: rng, looks: looks, h: h, clock: newClock(), episodeTimes: map[string]string{}, notes: map[string]string{}, money: startMoney, reputation: startReputation}

	// Los helpers ...Locked se pueden llamar sin tomar g.mu porque la
	// partida todavía no existe para ninguna otra goroutine.
	if err := g.hireLocked(g.newDoctorLocked(startSpeed, startSkill)); err != nil {
		return nil, fmt.Errorf("contratando al médico inicial: %w", err)
	}
	if err := g.hireLocked(g.newOrderlyLocked(startSpeed)); err != nil {
		return nil, fmt.Errorf("contratando al camillero inicial: %w", err)
	}
	g.nextArrival = arrivalInterval(rng, g.clock.day)
	g.dayStartMoney = g.money
	g.generateCandidatesLocked() // §6: la bolsa de empleo ya tiene 3 candidatos desde el día 1
	return g, nil
}

// Tick avanza la partida dt de tiempo real. Es la ÚNICA fuente de tiempo
// del juego: la llama la goroutine motor cada 100 ms y los tests la llaman
// a mano.
func (g *Game) Tick(dt time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.tickLocked(dt)
}

// tickLocked hace el trabajo de Tick con g.mu ya tomado. En pausa, después
// de las 20:00 o con la partida perdida no hace nada.
func (g *Game) tickLocked(dt time.Duration) {
	if g.paused || g.over != NotOver || g.clock.over() {
		return
	}
	g.clock.advance(dt)
	g.tickArrivalsLocked(dt)
	g.tickPatientsLocked(dt)
	g.tickStaffLocked(dt)
	g.checkDefeatLocked()
	if g.over == NotOver && g.clock.over() {
		// Este es el tick que llegó a las 20:00: los siguientes ya no
		// entran aquí (el reloj se detiene), así que esto corre una sola
		// vez. Primero los +500 del día (§5.10) y después la nómina
		// (§5.8), que puede dejar la plata en negativo y terminar la
		// partida por quiebra.
		g.score += dayCompletedPoints
		g.payPayrollLocked()
	}
}

// Pause congela la partida: reloj, cronómetros y llegadas (todo pasa por Tick).
func (g *Game) Pause() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.paused = true
}

// Resume quita la pausa.
func (g *Game) Resume() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.paused = false
}

// StartNextDay empieza el día siguiente a las 08:00. Solo se puede cuando
// el turno ya terminó (a las 20:00); si no, devuelve ErrDayNotOver. Si la
// partida ya se perdió (incluida la quiebra que pagó tickLocked al cerrar
// el día anterior), devuelve ErrGameOver: no hay día siguiente.
//
// La nómina del día que cierra se paga en tickLocked, no aquí (ver
// payPayrollLocked): a las 20:00, no cuando el jugador decide seguir. Aquí
// solo se reinician las cuentas del día nuevo y se renueva la bolsa de
// empleo (§6).
func (g *Game) StartNextDay() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.over != NotOver {
		return ErrGameOver
	}
	if !g.clock.over() {
		return fmt.Errorf("%w: son las %s", ErrDayNotOver, g.clock)
	}
	g.clock.nextDay()
	g.dayStartMoney = g.money
	g.dayReviewIncome = 0
	g.dayDischargeIncome = 0
	g.dayHires = nil
	g.generateCandidatesLocked()
	return nil
}

// noticeLocked agrega un aviso para la interfaz, con la hora del juego, y
// descarta los más viejos si hay más de maxNotices.
func (g *Game) noticeLocked(msg string) {
	g.notices = append(g.notices, g.clock.String()+" "+msg)
	if extra := len(g.notices) - maxNotices; extra > 0 {
		g.notices = g.notices[extra:]
	}
}
