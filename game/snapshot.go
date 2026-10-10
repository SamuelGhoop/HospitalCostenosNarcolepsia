package game

// Snapshot es la foto de la partida que dibuja la interfaz en cada cuadro.
//
// Solo tiene valores (números, textos, estados), nunca punteros del modelo:
// se arma con g.mu tomado y, una vez devuelta, la goroutine de la interfaz
// la puede leer sin candado aunque el motor siga avanzando la partida.
// Las fases siguientes le agregan personal, pacientes, plata y reputación.
type Snapshot struct {
	Day      int
	Clock    string  // hora de juego, por ejemplo "14:15"
	Progress float64 // cuánto va del turno, de 0 a 1
	Paused   bool
	DayOver  bool // ya son las 20:00 (la nómina y el Shift Report llegan en F4)
	Rooms    []RoomView
}

// Snapshot devuelve la foto actual de la partida.
func (g *Game) Snapshot() Snapshot {
	g.mu.Lock()
	defer g.mu.Unlock()

	snap := Snapshot{
		Day:      g.clock.day,
		Clock:    g.clock.String(),
		Progress: g.clock.progress(),
		Paused:   g.paused,
		DayOver:  g.clock.over(),
	}
	for _, r := range g.h.Rooms() {
		snap.Rooms = append(snap.Rooms, roomViewOf(r))
	}
	return snap
}
