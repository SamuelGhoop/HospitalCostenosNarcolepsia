package game_test

import (
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// tick es el paso del motor: 100 ms (GAME_DESIGN §5.1).
const tick = 100 * time.Millisecond

// newGame crea una partida con azar de semilla fija, para tests repetibles.
func newGame(t *testing.T) *game.Game {
	t.Helper()
	g, err := game.New(rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return g
}

// newQuietGame es una partida sin llegadas por la calle, para probar el
// reloj solo (ver StopArrivalsForTest en export_test.go).
func newQuietGame(t *testing.T) *game.Game {
	t.Helper()
	g := newGame(t)
	g.StopArrivalsForTest()
	return g
}

// tickFor avanza el juego d de tiempo real, en ticks de 100 ms (como el motor).
func tickFor(g *game.Game, d time.Duration) {
	for elapsed := time.Duration(0); elapsed < d; elapsed += tick {
		g.Tick(tick)
	}
}

func TestNew_StartsDay1At0800WithThreeRooms(t *testing.T) {
	snap := newGame(t).Snapshot()

	if snap.Day != 1 || snap.Clock != "08:00" || snap.Progress != 0 {
		t.Errorf("inicio = (día %d, %s, progreso %v); se esperaba (día 1, 08:00, 0)", snap.Day, snap.Clock, snap.Progress)
	}
	if snap.Paused || snap.DayOver {
		t.Errorf("una partida nueva no arranca en pausa ni con el día terminado: %+v", snap)
	}
	if len(snap.Rooms) != 3 {
		t.Fatalf("hay %d habitaciones; se esperaban 3 (§5.2)", len(snap.Rooms))
	}
	for i, want := range []int{101, 102, 103} {
		if r := snap.Rooms[i]; r.Number != want || r.State != hospital.Available {
			t.Errorf("habitación %d = (%d, %v); se esperaba (%d, disponible)", i, r.Number, r.State, want)
		}
	}
}

// Pedido por Samuel: el reloj usa aritmética entera, así que nunca redondea
// hacia arriba (no puede decir "20:00" ni terminar el día antes de tiempo).
func TestClock_NeverRoundsUp(t *testing.T) {
	g := newQuietGame(t)

	g.Tick(tick)
	if got := g.Snapshot().Clock; got != "08:00" {
		t.Errorf("tras 1 tick de 100 ms: %s; se esperaba 08:00", got)
	}

	tickFor(g, 299*time.Second+800*time.Millisecond) // en total: 299,9 s
	snap := g.Snapshot()
	if snap.Clock != "19:59" || snap.DayOver {
		t.Errorf("a los 299,9 s: (%s, terminado=%v); se esperaba (19:59, false)", snap.Clock, snap.DayOver)
	}
}

func TestTick_ShiftLasts300RealSeconds(t *testing.T) {
	g := newQuietGame(t)

	tickFor(g, 150*time.Second)
	if snap := g.Snapshot(); snap.Clock != "14:00" || snap.Progress != 0.5 {
		t.Errorf("a los 150 s: (%s, progreso %v); se esperaba (14:00, 0.5)", snap.Clock, snap.Progress)
	}

	tickFor(g, 150*time.Second)
	if snap := g.Snapshot(); snap.Clock != "20:00" || !snap.DayOver {
		t.Errorf("a los 300 s: (%s, terminado=%v); se esperaba (20:00, true)", snap.Clock, snap.DayOver)
	}
}

// 12 h de juego en 300 s reales: 1 s real = 2,4 min de juego, así que 25 s
// son 60 min.
func TestTick_OneRealSecondIsTwoPointFourGameMinutes(t *testing.T) {
	g := newGame(t)

	tickFor(g, 25*time.Second)

	if got := g.Snapshot().Clock; got != "09:00" {
		t.Errorf("a los 25 s: %s; se esperaba 09:00", got)
	}
}

func TestTick_ClockStopsAt2000(t *testing.T) {
	g := newQuietGame(t)

	tickFor(g, 400*time.Second)
	g.Tick(10 * time.Second)

	snap := g.Snapshot()
	if snap.Clock != "20:00" || snap.Progress != 1 || !snap.DayOver {
		t.Errorf("después de las 20:00: (%s, progreso %v, terminado=%v); se esperaba (20:00, 1, true)",
			snap.Clock, snap.Progress, snap.DayOver)
	}
}

func TestPause_FreezesTheClock(t *testing.T) {
	g := newGame(t)

	g.Pause()
	tickFor(g, 60*time.Second)
	if snap := g.Snapshot(); snap.Clock != "08:00" || !snap.Paused {
		t.Errorf("en pausa: (%s, pausado=%v); se esperaba (08:00, true)", snap.Clock, snap.Paused)
	}

	g.Resume()
	tickFor(g, 25*time.Second)
	if snap := g.Snapshot(); snap.Clock != "09:00" || snap.Paused {
		t.Errorf("después de reanudar: (%s, pausado=%v); se esperaba (09:00, false)", snap.Clock, snap.Paused)
	}
}

func TestStartNextDay(t *testing.T) {
	g := newQuietGame(t)

	if err := g.StartNextDay(); !errors.Is(err, game.ErrDayNotOver) {
		t.Fatalf("antes de las 20:00: err = %v; se esperaba ErrDayNotOver", err)
	}

	tickFor(g, 300*time.Second)
	if err := g.StartNextDay(); err != nil {
		t.Fatalf("a las 20:00: %v", err)
	}

	snap := g.Snapshot()
	if snap.Day != 2 || snap.Clock != "08:00" || snap.DayOver {
		t.Errorf("día siguiente = (día %d, %s, terminado=%v); se esperaba (día 2, 08:00, false)", snap.Day, snap.Clock, snap.DayOver)
	}
}

func TestSnapshot_ReturnsCopies(t *testing.T) {
	g := newGame(t)

	snap := g.Snapshot()
	snap.Rooms[0].Number = 999
	snap.Rooms[0].Occupants = append(snap.Rooms[0].Occupants, "X-999")

	again := g.Snapshot()
	if again.Rooms[0].Number != 101 || len(again.Rooms[0].Occupants) != 0 {
		t.Error("modificar la foto cambió la partida; Snapshot() debe devolver copias")
	}
}
