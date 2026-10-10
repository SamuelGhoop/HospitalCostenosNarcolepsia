package game

import (
	"sort"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// Vistas: lo que la interfaz necesita para dibujar, copiado en valores
// simples (textos, números y los estados tipados del modelo).
//
// Ninguna vista guarda un *hospital.Patient ni un *hospital.Room: si la
// interfaz recibiera punteros del modelo podría leerlo o cambiarlo sin pasar
// por game, y los getters del modelo no son seguros con goroutines.

// PatientView es la foto de un paciente.
type PatientView struct {
	ID, Name       string
	Age            int
	Level          hospital.NarcolepsyLevel
	State          hospital.PatientState // estado en el modelo
	Location       string                // dónde está ahora
	Room           int                   // número de habitación; 0 = sin cama
	TreatingDoctor string                // "" si nadie lo ha diagnosticado
}

// StaffView es la foto de un miembro del personal.
type StaffView struct {
	ID, Name string
	IsDoctor bool // false = camillero
}

// RoomView es la foto de una habitación.
type RoomView struct {
	Number, Capacity int
	State            hospital.RoomState
	Occupants        []string // IDs de los pacientes que duermen ahí
}

// EpisodeLine es un episodio del historial, listo para mostrar.
type EpisodeLine struct {
	ID, Time       string // Time: "15:04" (hora real en la demo, hora del juego en el modo juego)
	PatientID      string
	PatientName    string
	Location       string // dónde se quedó dormido
	Room           int    // habitación asignada; 0 = se quedó en el pasillo
	AttendedBy     string // quién lo atendió (médico o camillero)
	AttendedByID   string // su ID: la interfaz lo usa para saber a quién mover
	TreatingDoctor string // médico tratante en ese momento; "" si no tenía
	PatientNote    string // modo juego: "(alta)" o "(se fue enojado)"; "" si sigue en el mapa
}

// DoctorEpisodes agrupa los episodios que atendió un médico (consulta 5.2).
type DoctorEpisodes struct {
	DoctorID, DoctorName string
	Episodes             []EpisodeLine
}

// SevereLine es una fila del reporte de severidad (consulta 5.4).
type SevereLine struct {
	PatientID, PatientName string
	Episodes               int
	Note                   string // modo juego: "(alta)" o "(se fue enojado)"
}

// Report reúne las cuatro consultas de la Sección 5 (el Shift Report).
type Report struct {
	Hallway         []PatientView    // 5.1: dormidos en el pasillo
	Episodes        []DoctorEpisodes // 5.2: un bloque por médico
	Rooms           []RoomView       // 5.3: estado de las habitaciones…
	LastAssignError string           // …y el último error de AssignRoom ("" si no hubo)
	Severe          []SevereLine     // 5.4: pacientes Severe, ordenados por ID
}

// buildReport arma el Report llamando las consultas REALES del modelo; aquí
// no se recalcula nada, solo se copian los resultados a vistas.
//
//   - doctors: los médicos cuya consulta 5.2 se muestra. La demo pasa
//     h.Doctors(); el modo juego pasará los médicos envueltos en StaffMember,
//     que no aparecen en h.Doctors().
//   - timeOf: función que decide qué hora mostrar para cada episodio. Se
//     recibe como parámetro (las funciones también son valores en Go) para
//     que la demo muestre la hora real y el juego la hora del juego.
func buildReport(h *hospital.Hospital, doctors []*hospital.Doctor, lastAssignErr string,
	timeOf func(hospital.EpisodeRecord) string) Report {

	r := Report{LastAssignError: lastAssignErr}

	for _, p := range h.PatientsInHallway() {
		r.Hallway = append(r.Hallway, patientViewOf(p))
	}

	for _, d := range doctors {
		block := DoctorEpisodes{DoctorID: d.ID(), DoctorName: d.Name()}
		for _, rec := range d.MyEpisodes() {
			block.Episodes = append(block.Episodes, episodeLineOf(rec, timeOf))
		}
		r.Episodes = append(r.Episodes, block)
	}

	for _, room := range h.Rooms() {
		r.Rooms = append(r.Rooms, roomViewOf(room))
	}

	// SevereReport devuelve un map, que no tiene orden: se pasa a un slice
	// y se ordena por ID para que el reporte salga igual siempre.
	for p, n := range h.SevereReport() {
		r.Severe = append(r.Severe, SevereLine{PatientID: p.ID(), PatientName: p.Name(), Episodes: n})
	}
	sort.Slice(r.Severe, func(i, j int) bool {
		return r.Severe[i].PatientID < r.Severe[j].PatientID
	})

	return r
}

// patientViewOf copia los datos de un paciente del modelo a una vista.
func patientViewOf(p *hospital.Patient) PatientView {
	v := PatientView{
		ID:       p.ID(),
		Name:     p.Name(),
		Age:      p.Age(),
		Level:    p.Level(),
		State:    p.State(),
		Location: p.Location(),
	}
	if r := p.Room(); r != nil {
		v.Room = r.Number()
	}
	if d := p.AssignedDoctor(); d != nil {
		v.TreatingDoctor = d.Name()
	}
	return v
}

// roomViewOf copia los datos de una habitación del modelo a una vista.
func roomViewOf(r *hospital.Room) RoomView {
	v := RoomView{Number: r.Number(), Capacity: r.Capacity(), State: r.State()}
	for _, p := range r.Occupants() {
		v.Occupants = append(v.Occupants, p.ID())
	}
	return v
}

// episodeLineOf copia un registro del historial a una línea para mostrar.
func episodeLineOf(rec hospital.EpisodeRecord, timeOf func(hospital.EpisodeRecord) string) EpisodeLine {
	line := EpisodeLine{
		ID:           rec.ID(),
		Time:         timeOf(rec),
		PatientID:    rec.Patient().ID(),
		PatientName:  rec.Patient().Name(),
		Location:     rec.Location(),
		AttendedBy:   rec.AttendedBy().Name(),
		AttendedByID: rec.AttendedBy().ID(),
	}
	if r := rec.Room(); r != nil {
		line.Room = r.Number()
	}
	if d := rec.TreatingDoctor(); d != nil {
		line.TreatingDoctor = d.Name()
	}
	return line
}

// realTime muestra la hora real en que ocurrió el episodio (At()), como la
// salida de consola de main.go. La usa la demo.
func realTime(rec hospital.EpisodeRecord) string {
	return rec.At().Format("15:04")
}

// Report arma las cuatro consultas de la Sección 5 de la partida (el Shift
// Report), con el modelo real.
//
// Los médicos de la consulta 5.2 salen de la lista de Game, porque los
// envueltos en StaffMember no aparecen en h.Doctors() (§3.1). Sus episodios
// salen de MyEpisodes: como StaffMember delega Attend, el Doctor real los
// guarda.
func (g *Game) Report() Report {
	g.mu.Lock()
	defer g.mu.Unlock()
	r := buildReport(g.h, g.doctorsLocked(), g.lastAssignErr, g.gameTimeOf)

	// Los que ya se fueron siguen en el modelo (no tiene alta): el reporte
	// los marca con la nota que les dejó el juego al irse.
	for i := range r.Episodes {
		for j := range r.Episodes[i].Episodes {
			line := &r.Episodes[i].Episodes[j] // puntero: para cambiar el elemento, no una copia
			line.PatientNote = g.notes[line.PatientID]
		}
	}
	for i := range r.Severe {
		r.Severe[i].Note = g.notes[r.Severe[i].PatientID]
	}
	return r
}

// doctorsLocked devuelve los médicos reales (sin envolver) de la partida.
func (g *Game) doctorsLocked() []*hospital.Doctor {
	var doctors []*hospital.Doctor
	for _, s := range g.staff {
		if s.isDoctor() {
			doctors = append(doctors, s.doctor)
		}
	}
	return doctors
}

// gameTimeOf muestra la hora del JUEGO en que ocurrió el episodio, guardada
// al recogerlo. La usa el modo juego, siempre con g.mu tomado.
func (g *Game) gameTimeOf(rec hospital.EpisodeRecord) string {
	return g.episodeTimes[rec.ID()]
}
