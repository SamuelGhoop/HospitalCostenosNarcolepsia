package game_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// collapse deja a un paciente nuevo desplomado en zone (ver export_test.go).
// El ID lo pone el juego, igual que con las llegadas reales; wantID es el que
// el test espera, para escribir los tests con IDs fijos y legibles.
func collapse(t *testing.T, g *game.Game, wantID, name string, level hospital.NarcolepsyLevel, zone game.Zone) {
	t.Helper()
	id, err := g.CollapseForTest(name, level, zone)
	if err != nil {
		t.Fatalf("CollapseForTest(%s): %v", name, err)
	}
	if id != wantID {
		t.Fatalf("CollapseForTest le puso %s a %s; el test esperaba %s", id, name, wantID)
	}
}

// staffByID busca a un miembro del personal en la foto.
func staffByID(t *testing.T, snap game.Snapshot, id string) game.StaffMemberView {
	t.Helper()
	for _, s := range snap.Staff {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("%s no está en la foto del personal", id)
	return game.StaffMemberView{}
}

// patientByID busca a un paciente en la foto.
func patientByID(t *testing.T, snap game.Snapshot, id string) game.GamePatientView {
	t.Helper()
	for _, p := range snap.Patients {
		if p.ID == id {
			return p
		}
	}
	t.Fatalf("%s no está en la foto de los pacientes", id)
	return game.GamePatientView{}
}

// §5.5, recoger: el camillero camina (8 − rapidez) s, lo carga, ahí se hace
// el despacho "de guardia" y lo lleva a la cama en 2 s.
func TestDispatch_OrderlyWalksPicksUpAndCarries(t *testing.T) {
	g := newGame(t)
	collapse(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria)

	if err := g.Dispatch("P-001", "C-01"); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	snap := g.Snapshot()
	if s := staffByID(t, snap, "C-01"); s.Activity != game.GoingToPickUp || s.PatientID != "P-001" {
		t.Errorf("C-01 = (%v, %q); se esperaba (va a recogerlo, P-001)", s.Activity, s.PatientID)
	}
	if p := patientByID(t, snap, "P-001"); p.Stage != game.Collapsed || p.State != hospital.Awake {
		t.Errorf("P-001 = (%v, %v); mientras nadie lo recoge el modelo no se entera", p.Stage, p.State)
	}

	// Con rapidez 3 camina 5 s: a los 4,9 s todavía no ha llegado.
	tickFor(g, 4900*time.Millisecond)
	if p := patientByID(t, g.Snapshot(), "P-001"); p.Stage != game.Collapsed {
		t.Fatalf("a los 4,9 s P-001 ya está %v; el camillero no había llegado", p.Stage)
	}

	g.Tick(tick) // 5 s: llega y lo carga
	snap = g.Snapshot()
	if p := patientByID(t, snap, "P-001"); p.Stage != game.AwaitingReview || p.State != hospital.AsleepInBed || p.Room != 101 {
		t.Errorf("P-001 = (%v, %v, hab. %d); se esperaba (esperando revisión, dormido en cama, 101)", p.Stage, p.State, p.Room)
	}
	if s := staffByID(t, snap, "C-01"); s.Activity != game.Carrying || s.PatientID != "P-001" {
		t.Errorf("C-01 = (%v, %q); se esperaba que lo llevara a la cama", s.Activity, s.PatientID)
	}

	tickFor(g, 1900*time.Millisecond)
	if s := staffByID(t, g.Snapshot(), "C-01"); s.Activity != game.Carrying {
		t.Errorf("a los 1,9 s de cargarlo C-01 ya está %v; llevarlo dura 2 s", s.Activity)
	}
	g.Tick(tick)
	if s := staffByID(t, g.Snapshot(), "C-01"); s.Activity != game.Free || s.PatientID != "" {
		t.Errorf("C-01 = (%v, %q); después de dejarlo en la cama queda libre", s.Activity, s.PatientID)
	}
}

func TestDispatch_UnknownPatientOrStaff(t *testing.T) {
	g := newGame(t)
	collapse(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria)

	if err := g.Dispatch("P-999", "C-01"); !errors.Is(err, game.ErrUnknownPatient) {
		t.Errorf("paciente inexistente: err = %v; se esperaba ErrUnknownPatient", err)
	}
	if err := g.Dispatch("P-001", "X-01"); !errors.Is(err, game.ErrUnknownStaff) {
		t.Errorf("personal inexistente: err = %v; se esperaba ErrUnknownStaff", err)
	}
}

// dispatch llama Dispatch y corta el test si falla.
func dispatch(t *testing.T, g *game.Game, patientID, staffID string) {
	t.Helper()
	if err := g.Dispatch(patientID, staffID); err != nil {
		t.Fatalf("Dispatch(%s, %s): %v", patientID, staffID, err)
	}
}

// El mecanismo "de guardia" (§3.2). El personal quedó en el orden
// [D-01, C-01], así que el round-robin de RegisterEpisode, por turno,
// elegiría al médico. Si el jugador manda al camillero, el episodio tiene
// que quedar a nombre del camillero y no del médico.
func TestDispatch_TheChosenOneAttendsEvenIfItIsNotTheirTurn(t *testing.T) {
	g := newGame(t)
	collapse(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria)
	dispatch(t, g, "P-001", "C-01")
	tickFor(g, 5*time.Second)

	rep := g.Report()
	if len(rep.Episodes) != 1 || rep.Episodes[0].DoctorID != "D-01" {
		t.Fatalf("la consulta 5.2 debe traer el bloque de D-01, aunque los médicos envueltos no salgan en Doctors(): %+v", rep.Episodes)
	}
	if n := len(rep.Episodes[0].Episodes); n != 0 {
		t.Errorf("D-01 tiene %d episodios; a P-001 lo recogió C-01", n)
	}
}

// §5.5: un médico solo recoge si no hay ningún camillero libre. Así la
// consulta 5.2 (MyEpisodes) también tiene datos en el modo juego, con la
// hora del juego.
func TestDispatch_DoctorPicksUpOnlyWhenNoOrderlyIsFree(t *testing.T) {
	g := newGame(t)
	collapse(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria)
	collapse(t, g, "P-002", "Kevin Mercado", hospital.Moderate, game.Radiology)

	if err := g.Dispatch("P-001", "D-01"); !errors.Is(err, game.ErrOrderlyAvailable) {
		t.Errorf("con C-01 libre, mandar a D-01 a recoger: err = %v; se esperaba ErrOrderlyAvailable", err)
	}
	dispatch(t, g, "P-001", "C-01")
	dispatch(t, g, "P-002", "D-01") // ahora C-01 está ocupado
	tickFor(g, 5*time.Second)       // llegan a los 5 s: 08:12 en el juego

	rep := g.Report()
	if len(rep.Episodes) != 1 || len(rep.Episodes[0].Episodes) != 1 {
		t.Fatalf("D-01 debe tener 1 episodio en la consulta 5.2: %+v", rep.Episodes)
	}
	e := rep.Episodes[0].Episodes[0]
	if e.PatientID != "P-002" || e.AttendedByID != "D-01" || e.Location != "radiología" || e.Time != "08:12" {
		t.Errorf("episodio de D-01 = %+v; se esperaba P-002 en radiología a las 08:12 (hora del juego)", e)
	}
}

// §5.5: sin cama, el paciente queda en el pasillo, sale el aviso rojo y el
// personal queda libre de una vez (no hay a dónde llevarlo).
func TestDispatch_WithoutBedThePatientStaysInTheHallway(t *testing.T) {
	g := newGame(t)
	for _, id := range []string{"P-001", "P-002", "P-003"} { // llenan la 101, la 102 y la 103
		collapse(t, g, id, "Paciente "+id, hospital.Mild, game.Cafeteria)
		dispatch(t, g, id, "C-01")
		tickFor(g, 7*time.Second) // 5 s caminando + 2 s llevándolo
	}
	collapse(t, g, "P-004", "Wilfrido Berrío", hospital.Severe, game.Hallway2)
	dispatch(t, g, "P-004", "C-01")
	tickFor(g, 5*time.Second)

	snap := g.Snapshot()
	if p := patientByID(t, snap, "P-004"); p.Stage != game.InHallway || p.State != hospital.AsleepInHallway || p.Room != 0 {
		t.Errorf("P-004 = (%v, %v, hab. %d); se esperaba (en el pasillo, dormido en el pasillo, sin cama)", p.Stage, p.State, p.Room)
	}
	if s := staffByID(t, snap, "C-01"); s.Activity != game.Free {
		t.Errorf("C-01 está %v; sin cama no hay a dónde llevarlo y queda libre de una vez", s.Activity)
	}
	if n := len(snap.Notices); n == 0 || !strings.Contains(snap.Notices[n-1], "P-004") ||
		!strings.Contains(snap.Notices[n-1], hospital.ErrNoRoomAvailable.Error()) {
		t.Errorf("falta el aviso de que P-004 se quedó sin cama: %q", snap.Notices)
	}
}

// §5.5, revisar: el médico camina hasta la habitación, lo revisa y el
// paciente queda en cama. La revisión es solo del juego: no toca el modelo
// (tampoco DiagnosePatient, que llenaría el cupo de 4 del médico para siempre).
func TestDispatch_DoctorReviewsWithoutTouchingTheModel(t *testing.T) {
	g := newGame(t)
	collapse(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria)
	dispatch(t, g, "P-001", "C-01")
	tickFor(g, 5*time.Second) // C-01 lo recoge y empieza a llevarlo a la 101

	// Ya se puede mandar al médico aunque C-01 lo siga llevando: el médico
	// tarda más en llegar (mínimo 3 s) que el traslado (2 s).
	dispatch(t, g, "P-001", "D-01")
	if s := staffByID(t, g.Snapshot(), "D-01"); s.Activity != game.GoingToReview || s.PatientID != "P-001" {
		t.Errorf("D-01 = (%v, %q); se esperaba (va a revisarlo, P-001)", s.Activity, s.PatientID)
	}

	before := g.Report()
	tickFor(g, 5*time.Second)
	snap := g.Snapshot()

	p := patientByID(t, snap, "P-001")
	if p.Stage != game.InBed || p.ReviewedBy != "D-01" {
		t.Errorf("P-001 = (%v, revisado por %q); se esperaba (en cama, D-01)", p.Stage, p.ReviewedBy)
	}
	if p.State != hospital.AsleepInBed || p.Room != 101 || p.TreatingDoctor != "" {
		t.Errorf("P-001 en el modelo = (%v, hab. %d, tratante %q); la revisión no debe cambiar el modelo", p.State, p.Room, p.TreatingDoctor)
	}
	if s := staffByID(t, snap, "D-01"); s.Activity != game.Free {
		t.Errorf("D-01 está %v; después de revisar queda libre", s.Activity)
	}
	if after := g.Report(); !reflect.DeepEqual(before, after) {
		t.Errorf("la revisión cambió las consultas del modelo:\nantes:   %+v\ndespués: %+v", before, after)
	}
}

func TestDispatch_OnlyADoctorCanReview(t *testing.T) {
	g := newGame(t)
	collapse(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria)
	dispatch(t, g, "P-001", "C-01")
	tickFor(g, 7*time.Second) // C-01 lo deja en la cama y queda libre

	if err := g.Dispatch("P-001", "C-01"); !errors.Is(err, game.ErrNotADoctor) {
		t.Errorf("mandar al camillero a revisar: err = %v; se esperaba ErrNotADoctor", err)
	}
}

// Dispatch rechaza lo que no se puede hacer, y al rechazar no cambia nada.
func TestDispatch_RejectsWhatCannotBeDone(t *testing.T) {
	g := newGame(t)
	collapse(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria)
	collapse(t, g, "P-002", "Kevin Mercado", hospital.Moderate, game.Radiology)
	dispatch(t, g, "P-001", "C-01")

	if err := g.Dispatch("P-002", "C-01"); !errors.Is(err, game.ErrStaffBusy) {
		t.Errorf("C-01 ya va hacia P-001: err = %v; se esperaba ErrStaffBusy", err)
	}
	if err := g.Dispatch("P-001", "D-01"); !errors.Is(err, game.ErrAlreadyDispatched) {
		t.Errorf("a P-001 ya lo van a recoger: err = %v; se esperaba ErrAlreadyDispatched", err)
	}
	if s := staffByID(t, g.Snapshot(), "D-01"); s.Activity != game.Free {
		t.Errorf("después de un despacho rechazado D-01 quedó %v; debía seguir libre", s.Activity)
	}

	tickFor(g, 7*time.Second) // C-01 lo deja en la 101
	dispatch(t, g, "P-001", "D-01")
	tickFor(g, 5*time.Second) // D-01 lo revisa: queda en cama

	for _, staffID := range []string{"C-01", "D-01"} {
		if err := g.Dispatch("P-001", staffID); !errors.Is(err, game.ErrNothingToDispatch) {
			t.Errorf("P-001 ya está revisado y en cama; mandarle a %s: err = %v; se esperaba ErrNothingToDispatch", staffID, err)
		}
	}
}

// La pausa también congela al personal: todo avanza dentro de Tick.
func TestPause_FreezesTheStaff(t *testing.T) {
	g := newGame(t)
	collapse(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria)
	dispatch(t, g, "P-001", "C-01")

	g.Pause()
	tickFor(g, 10*time.Second)
	if p := patientByID(t, g.Snapshot(), "P-001"); p.Stage != game.Collapsed {
		t.Errorf("en pausa C-01 llegó donde P-001 (quedó %v)", p.Stage)
	}

	g.Resume()
	tickFor(g, 5*time.Second)
	if p := patientByID(t, g.Snapshot(), "P-001"); p.Stage != game.AwaitingReview {
		t.Errorf("después de reanudar y 5 s, P-001 está %v; se esperaba esperando revisión", p.Stage)
	}
}

// Los avisos de la foto son una copia: la interfaz no puede cambiar los del juego.
func TestSnapshot_NoticesAreCopies(t *testing.T) {
	g := newGame(t)
	for _, id := range []string{"P-001", "P-002", "P-003", "P-004"} { // P-004 se queda sin cama
		collapse(t, g, id, "Paciente "+id, hospital.Mild, game.Cafeteria)
		dispatch(t, g, id, "C-01")
		tickFor(g, 7*time.Second)
	}

	snap := g.Snapshot()
	if len(snap.Notices) == 0 {
		t.Fatal("se esperaba el aviso de P-004 sin cama")
	}
	snap.Notices[0] = "cambiado por la interfaz"
	if got := g.Snapshot().Notices[0]; got == "cambiado por la interfaz" {
		t.Error("cambiar los avisos de la foto cambió los del juego")
	}
}
