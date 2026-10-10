package game

import "time"

// Snapshot es la foto de la partida que dibuja la interfaz en cada cuadro.
//
// Solo tiene valores (números, textos, estados), nunca punteros del modelo:
// se arma con g.mu tomado y, una vez devuelta, la goroutine de la interfaz
// la puede leer sin candado aunque el motor siga avanzando la partida.
type Snapshot struct {
	Day        int
	Clock      string  // hora de juego, por ejemplo "14:15"
	Progress   float64 // cuánto va del turno, de 0 a 1
	Paused     bool
	DayOver    bool // ya son las 20:00 (la nómina y el Shift Report llegan en F4)
	Rooms      []RoomView
	Staff      []StaffMemberView // en orden de contratación
	Patients   []GamePatientView // en orden de llegada
	Notices    []string          // últimos avisos, del más viejo al más nuevo: "08:12 …"
	DoorOpen   bool              // la puerta del hospital está abierta: alguien la va a cruzar (§5.3)
	Money      int               // plata del hospital, en pesos
	Reputation int               // en centésimas de estrella (300 = 3 ★); la interfaz la muestra como "3,00 ★"
	GameOver   GameOver          // NotOver mientras la partida sigue
	Score      int               // puntaje acumulado (§5.10)
	FinalScore int               // puntaje final, solo con la partida perdida; 0 mientras sigue
}

// StaffMemberView es la foto de un miembro del personal en el modo juego.
//
// Embebe StaffView (ID, nombre y si es médico), la misma vista de la demo,
// igual que Doctor embebe Person: sus campos se leen directo, v.ID o v.Name.
type StaffMemberView struct {
	StaffView
	Speed, Skill int // Skill es 0 en los camilleros
	Activity     Activity
	PatientID    string // a quién va o a quién lleva; "" si está libre
}

// GamePatientView es la foto de un paciente en el modo juego: los datos del
// modelo (PatientView embebida, la misma de la demo) más su estado en el
// juego.
type GamePatientView struct {
	PatientView
	Stage      Stage
	Zone       Zone          // dónde está; la interfaz lo hace caminar hasta ahí
	Look       Appearance    // cómo se ve
	Waited     time.Duration // cronómetro de espera del episodio (§5.4): del desplome a la revisión
	ReviewedBy string        // ID del médico que lo revisó; "" si nadie
}

// Snapshot devuelve la foto actual de la partida.
func (g *Game) Snapshot() Snapshot {
	g.mu.Lock()
	defer g.mu.Unlock()

	snap := Snapshot{
		Day:        g.clock.day,
		Clock:      g.clock.String(),
		Progress:   g.clock.progress(),
		Paused:     g.paused,
		DayOver:    g.clock.over(),
		Money:      g.money,
		Reputation: g.reputation,
		GameOver:   g.over,
		Score:      g.score,
		// Copia: si se pasara g.notices tal cual, la interfaz compartiría el
		// arreglo de fondo con el juego mientras el motor le agrega avisos.
		Notices: append([]string(nil), g.notices...),
	}
	if g.over != NotOver {
		snap.FinalScore = g.finalScoreLocked()
	}
	for _, r := range g.h.Rooms() {
		snap.Rooms = append(snap.Rooms, roomViewOf(r))
	}
	for _, s := range g.staff {
		snap.Staff = append(snap.Staff, s.view())
	}
	for _, pt := range g.patients {
		snap.Patients = append(snap.Patients, pt.view())
		if pt.nearTheDoor() {
			snap.DoorOpen = true
		}
	}
	return snap
}
