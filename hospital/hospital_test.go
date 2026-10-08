package hospital_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// newTestHospital arma un hospital con las habitaciones indicadas
// (capacidad 1 cada una) y sin personal.
func newTestHospital(t *testing.T, roomNumbers ...int) *hospital.Hospital {
	t.Helper()
	h := hospital.NewHospital("Hospital de prueba")
	for _, n := range roomNumbers {
		if err := h.AddRoom(n, 1); err != nil {
			t.Fatalf("AddRoom(%d, 1): %v", n, err)
		}
	}
	return h
}

// admit crea y admite un paciente.
func admit(t *testing.T, h *hospital.Hospital, id string, level hospital.NarcolepsyLevel) *hospital.Patient {
	t.Helper()
	p := newTestPatient(t, id, level)
	if err := h.AdmitPatient(p); err != nil {
		t.Fatalf("AdmitPatient(%s): %v", id, err)
	}
	return p
}

// admitAsleep admite un paciente y lo duerme en el pasillo, sin pasar por
// RegisterEpisode (así AssignRoom se prueba solo).
func admitAsleep(t *testing.T, h *hospital.Hospital, id string) *hospital.Patient {
	t.Helper()
	p := admit(t, h, id, hospital.Moderate)
	if err := p.SufferSleepAttack("pasillo 2"); err != nil {
		t.Fatal(err)
	}
	return p
}

// withStaff contrata un camillero para que RegisterEpisode tenga a quién despachar.
func withStaff(t *testing.T, h *hospital.Hospital) {
	t.Helper()
	if err := h.HireStaff(hospital.NewOrderly("C-01", "Wilmer Camargo", 30)); err != nil {
		t.Fatal(err)
	}
}

// ───────────────────────────────────────────────────────────────────────
// Los tres tests que exige el enunciado (Sección 6).
// ───────────────────────────────────────────────────────────────────────

// [Obligatorio 1] Asignación de habitación cuando hay una libre.
func TestAssignRoom_AssignsTheFirstFreeRoom(t *testing.T) {
	h := newTestHospital(t, 101, 102)
	first := admitAsleep(t, h, "P-001")
	second := admitAsleep(t, h, "P-002")

	r1, err := h.AssignRoom(first)
	if err != nil {
		t.Fatalf("AssignRoom(P-001): %v", err)
	}
	r2, err := h.AssignRoom(second)
	if err != nil {
		t.Fatalf("AssignRoom(P-002): %v", err)
	}

	if r1.Number() != 101 || r2.Number() != 102 {
		t.Errorf("habitaciones = %d y %d; se esperaba 101 y 102 (la primera libre)", r1.Number(), r2.Number())
	}
	if r1.State() != hospital.Occupied {
		t.Errorf("la 101 quedó %v; se esperaba ocupada", r1.State())
	}
	if first.State() != hospital.AsleepInBed || first.Room() != r1 {
		t.Errorf("P-001 = (%v, %v); se esperaba (dormido en cama, 101)", first.State(), first.Room())
	}
	checkBedInvariant(t, first)
	checkBedInvariant(t, second)
}

// [Obligatorio 2] Falla cuando no hay habitación libre: devuelve error, el
// paciente sigue en el pasillo y no se inventa ninguna habitación.
func TestAssignRoom_NoFreeRoomReturnsErrorAndPatientStaysInHallway(t *testing.T) {
	h := newTestHospital(t, 101)
	if _, err := h.AssignRoom(admitAsleep(t, h, "P-001")); err != nil {
		t.Fatal(err)
	}
	unlucky := admitAsleep(t, h, "P-004")

	room, err := h.AssignRoom(unlucky)

	if !errors.Is(err, hospital.ErrNoRoomAvailable) {
		t.Fatalf("err = %v; se esperaba ErrNoRoomAvailable", err)
	}
	if room != nil {
		t.Errorf("devolvió la habitación %d; sin cama libre debe ser nil", room.Number())
	}
	if unlucky.State() != hospital.AsleepInHallway || unlucky.Room() != nil {
		t.Errorf("P-004 = (%v, %v); se esperaba (dormido en el pasillo, sin habitación)", unlucky.State(), unlucky.Room())
	}
	if n := len(h.Rooms()); n != 1 {
		t.Errorf("el hospital tiene %d habitaciones; no debe inventar ninguna", n)
	}
	checkBedInvariant(t, unlucky)
}

// [Obligatorio 3] Una de las consultas: pacientes dormidos en el pasillo (5.1).
func TestPatientsInHallway_ReturnsOnlyPatientsAsleepInHallway(t *testing.T) {
	h := newTestHospital(t, 101)
	admit(t, h, "P-001", hospital.Mild) // despierto
	inBed := admitAsleep(t, h, "P-002")
	if _, err := h.AssignRoom(inBed); err != nil {
		t.Fatal(err)
	}
	inHallway := admitAsleep(t, h, "P-003")

	got := h.PatientsInHallway()

	if len(got) != 1 || got[0] != inHallway {
		t.Fatalf("PatientsInHallway() = %v; se esperaba solo [%v]", got, inHallway)
	}
}

// ───────────────────────────────────────────────────────────────────────
// Consultas restantes y despacho polimórfico.
// ───────────────────────────────────────────────────────────────────────

// 5.4: cruza los pacientes Severe con los episodios del día en el historial.
func TestSevereReport_CrossesSeverePatientsWithTodaysEpisodes(t *testing.T) {
	h := newTestHospital(t) // sin habitaciones: todos se quedan en el pasillo
	withStaff(t, h)
	twice := admit(t, h, "P-002", hospital.Severe)
	never := admit(t, h, "P-005", hospital.Severe)
	mild := admit(t, h, "P-001", hospital.Mild)

	for _, step := range []func() error{
		func() error { return h.RegisterEpisode(twice, "cafetería") },
		func() error { return h.WakePatient(twice) },
		func() error { return h.RegisterEpisode(twice, "parqueadero") },
		func() error { return h.RegisterEpisode(mild, "sala de espera") },
	} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}

	report := h.SevereReport()

	if len(report) != 2 {
		t.Fatalf("el reporte tiene %d pacientes; se esperaban los 2 Severe: %v", len(report), report)
	}
	if report[twice] != 2 {
		t.Errorf("P-002 tiene %d episodios; se esperaban 2", report[twice])
	}
	if n, ok := report[never]; !ok || n != 0 {
		t.Errorf("P-005 = (%d, presente=%v); un Severe sin episodios debe aparecer con 0", n, ok)
	}
	if _, ok := report[mild]; ok {
		t.Error("P-001 es Mild y no debe aparecer en el reporte")
	}
}

// RegisterEpisode reparte las emergencias por turnos (round-robin) entre
// todo el personal, sin saber si cada uno es Doctor u Orderly. Eso alimenta
// la consulta 5.2: cada doctor ve en MyEpisodes solo lo que atendió.
func TestRegisterEpisode_DispatchesStaffInTurns(t *testing.T) {
	h := newTestHospital(t, 101, 102, 103, 104)
	karen := hospital.NewDoctor("D-01", "Dra. Karen Ospina", 45, "Neurología del sueño")
	rafa := hospital.NewDoctor("D-02", "Dr. Rafa Escalona", 50, "Medicina interna")
	wilmer := hospital.NewOrderly("C-01", "Wilmer Camargo", 30)
	for _, err := range []error{h.HireDoctor(karen), h.HireDoctor(rafa), h.HireStaff(wilmer)} {
		if err != nil {
			t.Fatal(err)
		}
	}

	var attenders []string
	for i := 1; i <= 4; i++ {
		p := admit(t, h, fmt.Sprintf("P-%03d", i), hospital.Moderate)
		if err := h.RegisterEpisode(p, "pasillo"); err != nil {
			t.Fatal(err)
		}
		history := h.History()
		attenders = append(attenders, history[len(history)-1].AttendedBy().ID())
	}

	want := []string{"D-01", "D-02", "C-01", "D-01"}
	if fmt.Sprint(attenders) != fmt.Sprint(want) {
		t.Errorf("atendieron %v; se esperaba %v", attenders, want)
	}
	if n := len(karen.MyEpisodes()); n != 2 {
		t.Errorf("MyEpisodes de la Dra. Ospina = %d; se esperaban 2", n)
	}
	if wilmer.Transfers() != 1 {
		t.Errorf("el camillero hizo %d traslados; se esperaba 1", wilmer.Transfers())
	}
}

// El despacho salta al doctor sin cupo y busca al siguiente disponible.
func TestRegisterEpisode_SkipsDoctorWithFullCaseload(t *testing.T) {
	h := newTestHospital(t, 101)
	busy := hospital.NewDoctor("D-01", "Dra. Karen Ospina", 45, "Neurología del sueño")
	free := hospital.NewDoctor("D-02", "Dr. Rafa Escalona", 50, "Medicina interna")
	if err := h.HireDoctor(busy); err != nil {
		t.Fatal(err)
	}
	if err := h.HireDoctor(free); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 4; i++ {
		if err := busy.DiagnosePatient(admit(t, h, fmt.Sprintf("P-%03d", i), hospital.Mild)); err != nil {
			t.Fatal(err)
		}
	}
	p := admit(t, h, "P-010", hospital.Severe)

	if err := h.RegisterEpisode(p, "radiología"); err != nil {
		t.Fatal(err)
	}

	if len(busy.MyEpisodes()) != 0 || len(free.MyEpisodes()) != 1 {
		t.Errorf("episodios: D-01=%d, D-02=%d; se esperaba que atendiera D-02",
			len(busy.MyEpisodes()), len(free.MyEpisodes()))
	}
}

// ───────────────────────────────────────────────────────────────────────
// RegisterEpisode y el flujo del escenario.
// ───────────────────────────────────────────────────────────────────────

func TestRegisterEpisode_PutsPatientInFirstFreeRoomAndRecordsIt(t *testing.T) {
	h := newTestHospital(t, 101)
	withStaff(t, h)
	p := admit(t, h, "P-001", hospital.Severe)

	if err := h.RegisterEpisode(p, "cafetería"); err != nil {
		t.Fatalf("RegisterEpisode: %v", err)
	}

	if p.State() != hospital.AsleepInBed || p.Room().Number() != 101 {
		t.Errorf("P-001 = (%v, %v); se esperaba (dormido en cama, 101)", p.State(), p.Room())
	}
	history := h.History()
	if len(history) != 1 || history[0].Patient() != p || history[0].Location() != "cafetería" {
		t.Fatalf("History() = %v; se esperaba un episodio de P-001 en la cafetería", history)
	}
	if history[0].Room() == nil || history[0].Room().Number() != 101 {
		t.Errorf("el registro dice habitación %v; se esperaba la 101", history[0].Room())
	}
	checkBedInvariant(t, p)
}

// No tener cama NO es un error de RegisterEpisode: el episodio queda
// registrado sin habitación y el paciente sigue en el pasillo (5.3).
func TestRegisterEpisode_WithoutFreeRoomStillRecordsTheEpisode(t *testing.T) {
	h := newTestHospital(t)
	withStaff(t, h)
	p := admit(t, h, "P-004", hospital.Severe)

	if err := h.RegisterEpisode(p, "pasillo 2, segundo piso"); err != nil {
		t.Fatalf("RegisterEpisode devolvió %v; sin cama debe devolver nil", err)
	}

	if p.State() != hospital.AsleepInHallway {
		t.Errorf("State() = %v; se esperaba dormido en el pasillo", p.State())
	}
	history := h.History()
	if len(history) != 1 || history[0].Room() != nil {
		t.Errorf("History() = %v; se esperaba un episodio sin habitación", history)
	}
}

func TestRegisterEpisode_FailsWithoutChangingAnything(t *testing.T) {
	t.Run("sin personal disponible", func(t *testing.T) {
		h := newTestHospital(t, 101) // nadie contratado
		p := admit(t, h, "P-001", hospital.Mild)

		if err := h.RegisterEpisode(p, "cafetería"); !errors.Is(err, hospital.ErrNoStaffAvailable) {
			t.Fatalf("err = %v; se esperaba ErrNoStaffAvailable", err)
		}
		if p.State() != hospital.Awake || len(h.History()) != 0 {
			t.Errorf("con error no debe cambiar nada: estado %v, historial %d", p.State(), len(h.History()))
		}
	})

	t.Run("paciente no admitido", func(t *testing.T) {
		h := newTestHospital(t, 101)
		withStaff(t, h)
		outsider := newTestPatient(t, "P-999", hospital.Mild)

		if err := h.RegisterEpisode(outsider, "portería"); !errors.Is(err, hospital.ErrNotAdmitted) {
			t.Fatalf("err = %v; se esperaba ErrNotAdmitted", err)
		}
		if outsider.State() != hospital.Awake {
			t.Error("un paciente ajeno al hospital no debe cambiar de estado")
		}
	})

	t.Run("paciente ya dormido", func(t *testing.T) {
		h := newTestHospital(t, 101)
		withStaff(t, h)
		p := admit(t, h, "P-001", hospital.Mild)
		if err := h.RegisterEpisode(p, "cafetería"); err != nil {
			t.Fatal(err)
		}

		if err := h.RegisterEpisode(p, "parqueadero"); !errors.Is(err, hospital.ErrAlreadyAsleep) {
			t.Fatalf("err = %v; se esperaba ErrAlreadyAsleep", err)
		}
		if len(h.History()) != 1 {
			t.Errorf("el historial tiene %d episodios; el intento fallido no debe sumar", len(h.History()))
		}
	})
}

// El corazón del escenario de la Sección 6: sin cama → error; alguien se
// despierta → la cama queda libre → el del pasillo la consigue.
func TestWakePatient_FreesTheRoomForThePatientInTheHallway(t *testing.T) {
	h := newTestHospital(t, 101)
	withStaff(t, h)
	sleeper := admit(t, h, "P-001", hospital.Severe)
	waiting := admit(t, h, "P-004", hospital.Severe)
	if err := h.RegisterEpisode(sleeper, "cafetería"); err != nil {
		t.Fatal(err)
	}
	if err := h.RegisterEpisode(waiting, "pasillo 2"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.AssignRoom(waiting); !errors.Is(err, hospital.ErrNoRoomAvailable) {
		t.Fatalf("antes de despertar a nadie: err = %v; se esperaba ErrNoRoomAvailable", err)
	}

	if err := h.WakePatient(sleeper); err != nil {
		t.Fatalf("WakePatient: %v", err)
	}
	room, err := h.AssignRoom(waiting)

	if err != nil || room.Number() != 101 {
		t.Fatalf("AssignRoom(P-004) = (%v, %v); se esperaba la 101", room, err)
	}
	if sleeper.State() != hospital.Awake {
		t.Errorf("P-001 = %v; se esperaba despierto", sleeper.State())
	}
	checkBedInvariant(t, sleeper)
	checkBedInvariant(t, waiting)
}

// ───────────────────────────────────────────────────────────────────────
// Altas de pacientes, personal y habitaciones.
// ───────────────────────────────────────────────────────────────────────

func TestAdmitPatient_RejectsNilAndDuplicates(t *testing.T) {
	h := newTestHospital(t)
	admit(t, h, "P-001", hospital.Mild)

	if err := h.AdmitPatient(nil); !errors.Is(err, hospital.ErrNilPatient) {
		t.Errorf("nil: err = %v; se esperaba ErrNilPatient", err)
	}
	// Otro objeto con el mismo ID también es un duplicado.
	if err := h.AdmitPatient(newTestPatient(t, "P-001", hospital.Severe)); !errors.Is(err, hospital.ErrAlreadyAdmitted) {
		t.Errorf("duplicado: err = %v; se esperaba ErrAlreadyAdmitted", err)
	}
	if n := len(h.Patients()); n != 1 {
		t.Errorf("Patients() tiene %d; se esperaba 1", n)
	}
}

func TestHire_RejectsNilAndDuplicatesAndKeepsDoctorsListed(t *testing.T) {
	h := newTestHospital(t)
	karen := hospital.NewDoctor("D-01", "Dra. Karen Ospina", 45, "Neurología del sueño")
	if err := h.HireDoctor(karen); err != nil {
		t.Fatal(err)
	}

	if err := h.HireDoctor(nil); !errors.Is(err, hospital.ErrNilAttender) {
		t.Errorf("HireDoctor(nil): err = %v; se esperaba ErrNilAttender", err)
	}
	if err := h.HireStaff(nil); !errors.Is(err, hospital.ErrNilAttender) {
		t.Errorf("HireStaff(nil): err = %v; se esperaba ErrNilAttender", err)
	}
	if err := h.HireStaff(karen); !errors.Is(err, hospital.ErrAlreadyHired) {
		t.Errorf("contratarla dos veces: err = %v; se esperaba ErrAlreadyHired", err)
	}

	// Un doctor contratado por HireStaff también debe quedar en Doctors().
	rafa := hospital.NewDoctor("D-02", "Dr. Rafa Escalona", 50, "Medicina interna")
	if err := h.HireStaff(rafa); err != nil {
		t.Fatal(err)
	}
	if len(h.Doctors()) != 2 || len(h.Staff()) != 2 {
		t.Errorf("Doctors()=%d, Staff()=%d; se esperaba 2 y 2", len(h.Doctors()), len(h.Staff()))
	}
}

func TestAddRoom_RejectsDuplicatesAndInvalidCapacity(t *testing.T) {
	h := newTestHospital(t, 101)

	if err := h.AddRoom(101, 1); !errors.Is(err, hospital.ErrDuplicateRoom) {
		t.Errorf("101 repetida: err = %v; se esperaba ErrDuplicateRoom", err)
	}
	if err := h.AddRoom(102, 0); !errors.Is(err, hospital.ErrInvalidCapacity) {
		t.Errorf("capacidad 0: err = %v; se esperaba ErrInvalidCapacity", err)
	}
	if n := len(h.Rooms()); n != 1 {
		t.Errorf("Rooms() tiene %d; se esperaba 1", n)
	}
}

func TestOperationsRejectPatientsFromOutside(t *testing.T) {
	h := newTestHospital(t, 101)
	outsider := newTestPatient(t, "P-999", hospital.Mild)
	if err := outsider.SufferSleepAttack("portería"); err != nil {
		t.Fatal(err)
	}

	if _, err := h.AssignRoom(outsider); !errors.Is(err, hospital.ErrNotAdmitted) {
		t.Errorf("AssignRoom: err = %v; se esperaba ErrNotAdmitted", err)
	}
	if err := h.WakePatient(outsider); !errors.Is(err, hospital.ErrNotAdmitted) {
		t.Errorf("WakePatient: err = %v; se esperaba ErrNotAdmitted", err)
	}
	if _, err := h.AssignRoom(nil); !errors.Is(err, hospital.ErrNilPatient) {
		t.Errorf("AssignRoom(nil): err = %v; se esperaba ErrNilPatient", err)
	}
}

func TestGettersReturnCopies(t *testing.T) {
	h := newTestHospital(t, 101)
	admit(t, h, "P-001", hospital.Mild)

	h.Patients()[0] = nil
	h.Rooms()[0] = nil

	if h.Patients()[0] == nil || h.Rooms()[0] == nil {
		t.Error("modificar los slices devueltos cambió el hospital; los getters deben devolver copias")
	}
}

// ───────────────────────────────────────────────────────────────────────
// Concurrencia: dos goroutines no pueden quedarse con la misma cama.
// (Se corre también con -race en Docker; ver README.)
// ───────────────────────────────────────────────────────────────────────

func TestAssignRoom_IsSafeWhenManyGoroutinesCompeteForBeds(t *testing.T) {
	h := newTestHospital(t, 101, 102, 103)
	var patients []*hospital.Patient
	for i := 1; i <= 10; i++ {
		patients = append(patients, admitAsleep(t, h, fmt.Sprintf("P-%03d", i)))
	}

	var wg sync.WaitGroup
	results := make(chan error, len(patients)) // con buffer: nadie se bloquea al enviar
	for _, p := range patients {
		p := p // copia explícita: con go 1.21 la variable del for es la misma en cada vuelta
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := h.AssignRoom(p)
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	assigned, noRoom := 0, 0
	for err := range results {
		switch {
		case err == nil:
			assigned++
		case errors.Is(err, hospital.ErrNoRoomAvailable):
			noRoom++
		default:
			t.Errorf("error inesperado: %v", err)
		}
	}
	if assigned != 3 || noRoom != 7 {
		t.Errorf("asignados=%d, sin cama=%d; se esperaba 3 y 7", assigned, noRoom)
	}
	for _, r := range h.Rooms() {
		if n := len(r.Occupants()); n != 1 {
			t.Errorf("la habitación %d quedó con %d ocupantes; se esperaba 1", r.Number(), n)
		}
	}
}
