package hospital

import (
	"fmt"
	"sync/atomic"
	"time"
)

// episodeSeq numera los episodios: EP-001, EP-002…
//
// Es una variable del paquete porque quien crea el registro es el Attender
// (dentro de Attend), y el Attender no conoce al hospital. atomic.Int64
// garantiza que dos goroutines nunca reciban el mismo número, aunque
// atiendan al mismo tiempo (bonus de concurrencia).
var episodeSeq atomic.Int64

// EpisodeRecord es el registro de un episodio de sueño: quién se durmió,
// dónde, a qué hora, quién lo atendió y a qué habitación fue a parar.
//
// Es inmutable: se crea completo en newEpisodeRecord y no tiene métodos que
// lo cambien. Copia el médico tratante y la habitación DEL MOMENTO; si después
// el paciente se despierta o cambia de médico, el registro no cambia.
type EpisodeRecord struct {
	id             string
	at             time.Time
	patient        *Patient
	attendedBy     Attender // Doctor u Orderly: la interfaz acepta cualquiera
	treatingDoctor *Doctor  // médico tratante en ese momento; nil si no tenía
	location       string   // dónde se quedó dormido
	room           *Room    // habitación asignada; nil = se quedó en el pasillo
}

// newEpisodeRecord arma el registro con el estado actual del paciente.
// Va en minúscula porque solo los Attender del paquete deben crear registros.
func newEpisodeRecord(p *Patient, by Attender, location string) EpisodeRecord {
	return EpisodeRecord{
		id:             fmt.Sprintf("EP-%03d", episodeSeq.Add(1)), // %03d rellena con ceros: 7 → "007"
		at:             p.asleepSince,
		patient:        p,
		attendedBy:     by,
		treatingDoctor: p.assignedDoctor,
		location:       location,
		room:           p.room,
	}
}

// ID devuelve el identificador del episodio, por ejemplo "EP-001".
func (e EpisodeRecord) ID() string { return e.id }

// At devuelve la fecha y hora en que el paciente se quedó dormido.
func (e EpisodeRecord) At() time.Time { return e.at }

// Patient devuelve el paciente del episodio.
func (e EpisodeRecord) Patient() *Patient { return e.patient }

// AttendedBy devuelve quién lo atendió (un Doctor o un Orderly).
func (e EpisodeRecord) AttendedBy() Attender { return e.attendedBy }

// TreatingDoctor devuelve el médico tratante que tenía el paciente en ese
// momento, o nil si no tenía.
func (e EpisodeRecord) TreatingDoctor() *Doctor { return e.treatingDoctor }

// Location devuelve dónde se quedó dormido.
func (e EpisodeRecord) Location() string { return e.location }

// Room devuelve la habitación asignada, o nil si se quedó en el pasillo.
func (e EpisodeRecord) Room() *Room { return e.room }

// Summary devuelve el episodio en una línea legible, por ejemplo:
//
//	EP-001  2026-10-20 14:15  P-001 Yeimy Padilla  cafetería → habitación 101 · atendió: Dra. Karen Ospina · tratante: Dr. Rafa Escalona
//
// Usa receiver de valor: el registro es pequeño y nunca se modifica.
func (e EpisodeRecord) Summary() string {
	destination := "pasillo (sin cama)"
	if e.room != nil {
		destination = fmt.Sprintf("habitación %d", e.room.number)
	}
	treating := "sin tratante"
	if e.treatingDoctor != nil {
		treating = e.treatingDoctor.Name()
	}
	// Go formatea fechas con una fecha de referencia fija: 2006-01-02 15:04
	// significa "año-mes-día hora:minuto".
	return fmt.Sprintf("%s  %s  %s  %s → %s · atendió: %s · tratante: %s",
		e.id, e.at.Format("2006-01-02 15:04"), e.patient, e.location,
		destination, e.attendedBy.Name(), treating)
}
