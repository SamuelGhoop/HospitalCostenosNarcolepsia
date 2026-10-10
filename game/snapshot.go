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
	Staff    []StaffMemberView // en orden de contratación
	Patients []GamePatientView // en orden de llegada
	Notices  []string          // últimos avisos, del más viejo al más nuevo: "08:12 …"
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
	ReviewedBy string // ID del médico que lo revisó; "" si nadie
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
		// Copia: si se pasara g.notices tal cual, la interfaz compartiría el
		// arreglo de fondo con el juego mientras el motor le agrega avisos.
		Notices: append([]string(nil), g.notices...),
	}
	for _, r := range g.h.Rooms() {
		snap.Rooms = append(snap.Rooms, roomViewOf(r))
	}
	for _, s := range g.staff {
		snap.Staff = append(snap.Staff, s.view())
	}
	for _, pt := range g.patients {
		snap.Patients = append(snap.Patients, pt.view())
	}
	return snap
}
