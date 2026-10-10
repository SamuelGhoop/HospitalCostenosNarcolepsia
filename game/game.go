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
	h      *hospital.Hospital
	clock  clock
	paused bool

	staff                     []*StaffMember    // todo el personal, en orden de contratación
	doctorCount, orderlyCount int               // para los IDs: D-01, D-02… y C-01, C-02…
	patients                  []*patient        // los pacientes de la partida, en orden de llegada
	patientCount              int               // para los IDs: P-001, P-002…
	nextArrival               time.Duration     // cuánto falta para que aparezca el paciente siguiente
	notes                     map[string]string // ID del paciente → cómo sale en el Shift Report: "(alta)"
	episodeTimes              map[string]string // ID del episodio → hora del juego ("14:15"), §3.2
	notices                   []string          // últimos avisos para la interfaz, del más viejo al más nuevo
}

// New crea una partida (GAME_DESIGN §5.2): un hospital con las habitaciones
// 101, 102 y 103, un médico y un camillero, el día 1 a las 08:00. rng es el
// azar de la partida; los tests pasan uno con semilla fija para que todo sea
// repetible.
func New(rng *rand.Rand) (*Game, error) {
	h := hospital.NewHospital("Hospital de los Costeños con Narcolepsia")
	for _, number := range []int{101, 102, 103} {
		if err := h.AddRoom(number, 1); err != nil {
			return nil, fmt.Errorf("creando la habitación %d: %w", number, err)
		}
	}
	g := &Game{rng: rng, h: h, clock: newClock(), episodeTimes: map[string]string{}, notes: map[string]string{}}

	// Los helpers ...Locked se pueden llamar sin tomar g.mu porque la
	// partida todavía no existe para ninguna otra goroutine.
	if err := g.hireLocked(g.newDoctorLocked(startSpeed, startSkill)); err != nil {
		return nil, fmt.Errorf("contratando al médico inicial: %w", err)
	}
	if err := g.hireLocked(g.newOrderlyLocked(startSpeed)); err != nil {
		return nil, fmt.Errorf("contratando al camillero inicial: %w", err)
	}
	g.nextArrival = arrivalInterval(rng, g.clock.day)
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

// tickLocked hace el trabajo de Tick con g.mu ya tomado. En pausa o después
// de las 20:00 no hace nada. (F1.3 y F1.4 le agregan llegadas, pacientes,
// camas, reputación y derrota.)
func (g *Game) tickLocked(dt time.Duration) {
	if g.paused || g.clock.over() {
		return
	}
	g.clock.advance(dt)
	g.tickArrivalsLocked(dt)
	g.tickPatientsLocked(dt)
	g.tickStaffLocked(dt)
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
// el turno ya terminó (a las 20:00); si no, devuelve ErrDayNotOver.
// (En F4 aquí se paga la nómina.)
func (g *Game) StartNextDay() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.clock.over() {
		return fmt.Errorf("%w: son las %s", ErrDayNotOver, g.clock)
	}
	g.clock.nextDay()
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
