package game_test

import (
	"errors"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// runSteps avanza la demo n pasos y devuelve el último.
func runSteps(t *testing.T, d *game.Demo, n int) game.DemoStep {
	t.Helper()
	var step game.DemoStep
	for i := 0; i < n; i++ {
		var err error
		step, err = d.Next()
		if err != nil {
			t.Fatalf("paso %d: Next() devolvió error: %v", i+1, err)
		}
	}
	return step
}

// patientView busca un paciente en la foto de la demo.
func patientView(t *testing.T, snap game.DemoSnapshot, id string) game.PatientView {
	t.Helper()
	for _, p := range snap.Patients {
		if p.ID == id {
			return p
		}
	}
	t.Fatalf("el paciente %s no está en la foto de la demo", id)
	return game.PatientView{}
}

func TestDemo_RunsTheTwelveStepsInOrder(t *testing.T) {
	d := game.NewDemo()

	for want := 1; want <= game.DemoSteps; want++ {
		step, err := d.Next()
		if err != nil {
			t.Fatalf("paso %d: %v", want, err)
		}
		if step.Number != want {
			t.Errorf("Number = %d; se esperaba %d", step.Number, want)
		}
		// Cada paso trae lo que la barra de subtítulos necesita (sección 4.2).
		if step.Subtitle == "" || step.Call == "" || step.Result == "" {
			t.Errorf("paso %d incompleto: %+v", want, step)
		}
	}
	if !d.Done() {
		t.Error("después de 12 pasos Done() debe ser true")
	}
}

func TestDemo_StartsEmpty(t *testing.T) {
	snap := game.NewDemo().Snapshot()

	if snap.Step != 0 || snap.TotalSteps != game.DemoSteps {
		t.Errorf("Step/TotalSteps = %d/%d; se esperaba 0/%d", snap.Step, snap.TotalSteps, game.DemoSteps)
	}
	if len(snap.Patients) != 0 || len(snap.Staff) != 0 || len(snap.Rooms) != 0 {
		t.Errorf("antes del primer paso no debe haber nadie: %+v", snap)
	}
}

func TestDemo_SetupStepsBuildTheHospitalOfSection6(t *testing.T) {
	d := game.NewDemo()
	runSteps(t, d, 4) // contratar, habitaciones, admitir, diagnosticar

	snap := d.Snapshot()
	if len(snap.Staff) != 3 || len(snap.Rooms) != 3 || len(snap.Patients) != 5 {
		t.Fatalf("personal/habitaciones/pacientes = %d/%d/%d; se esperaba 3/3/5",
			len(snap.Staff), len(snap.Rooms), len(snap.Patients))
	}
	if !snap.Staff[0].IsDoctor || !snap.Staff[1].IsDoctor || snap.Staff[2].IsDoctor {
		t.Errorf("se esperaban dos médicos y un camillero: %+v", snap.Staff)
	}
	if got := patientView(t, snap, "P-004").TreatingDoctor; got != "Dr. Efraín Barraza" {
		t.Errorf("tratante de P-004 = %q; se esperaba Dr. Efraín Barraza", got)
	}
}

// Pasos 5–7: los tres primeros ataques consiguen cama, en orden.
func TestDemo_FirstThreeAttacksGetRooms101To103(t *testing.T) {
	d := game.NewDemo()
	runSteps(t, d, 7)

	snap := d.Snapshot()
	for id, room := range map[string]int{"P-001": 101, "P-002": 102, "P-003": 103} {
		p := patientView(t, snap, id)
		if p.Room != room || p.State != hospital.AsleepInBed {
			t.Errorf("%s = (habitación %d, %v); se esperaba (habitación %d, dormido en cama)", id, p.Room, p.State, room)
		}
	}
}

// Paso 8: Wilfrido no consigue cama y el paso trae el error REAL del modelo.
func TestDemo_Step8CarriesTheRealNoRoomError(t *testing.T) {
	d := game.NewDemo()

	step := runSteps(t, d, 8)

	if !errors.Is(step.Err, hospital.ErrNoRoomAvailable) {
		t.Fatalf("Err = %v; se esperaba hospital.ErrNoRoomAvailable", step.Err)
	}
	p := patientView(t, d.Snapshot(), "P-004")
	if p.State != hospital.AsleepInHallway || p.Room != 0 {
		t.Errorf("P-004 = (%v, habitación %d); se esperaba (dormido en el pasillo, sin habitación)", p.State, p.Room)
	}
}

// Paso 9: la consulta 5.1 muestra a Wilfrido en el pasillo.
func TestDemo_Step9ShowsWilfridoInTheHallway(t *testing.T) {
	d := game.NewDemo()
	runSteps(t, d, 9)

	hallway := d.Report().Hallway
	if len(hallway) != 1 || hallway[0].ID != "P-004" || hallway[0].Location != "pasillo 2, segundo piso" {
		t.Fatalf("5.1 = %+v; se esperaba solo P-004 en el pasillo 2", hallway)
	}
}

// Pasos 10–11: Yeimy se despierta, libera la 101 y Wilfrido la consigue.
func TestDemo_WakingYeimyFreesRoom101ForWilfrido(t *testing.T) {
	d := game.NewDemo()
	runSteps(t, d, 11)

	snap := d.Snapshot()
	if p := patientView(t, snap, "P-001"); p.State != hospital.Awake || p.Room != 0 {
		t.Errorf("P-001 = (%v, habitación %d); se esperaba despierta y sin cama", p.State, p.Room)
	}
	if p := patientView(t, snap, "P-004"); p.State != hospital.AsleepInBed || p.Room != 101 {
		t.Errorf("P-004 = (%v, habitación %d); se esperaba en la 101", p.State, p.Room)
	}
}

// El Shift Report final da lo mismo que `go run .` en la raíz.
func TestDemo_FinalReportMatchesTheConsoleScenario(t *testing.T) {
	d := game.NewDemo()
	if err := d.Skip(); err != nil {
		t.Fatalf("Skip: %v", err)
	}

	r := d.Report()

	// 5.1: ya nadie en el pasillo.
	if len(r.Hallway) != 0 {
		t.Errorf("5.1 = %+v; se esperaba vacía", r.Hallway)
	}
	// 5.2: la Dra. Ospina atendió a P-001 y P-004; el Dr. Barraza a P-002.
	wantEpisodes := map[string][]string{"D-01": {"P-001", "P-004"}, "D-02": {"P-002"}}
	if len(r.Episodes) != 2 {
		t.Fatalf("5.2 tiene %d médicos; se esperaban 2", len(r.Episodes))
	}
	for _, de := range r.Episodes {
		want := wantEpisodes[de.DoctorID]
		if len(de.Episodes) != len(want) {
			t.Errorf("5.2 %s: %d episodios; se esperaban %d", de.DoctorID, len(de.Episodes), len(want))
			continue
		}
		for i, e := range de.Episodes {
			if e.PatientID != want[i] {
				t.Errorf("5.2 %s episodio %d: paciente %s; se esperaba %s", de.DoctorID, i+1, e.PatientID, want[i])
			}
		}
	}
	// 5.3: 101 → P-004, 102 → P-002, 103 → P-003, y el último error de AssignRoom.
	wantRooms := map[int]string{101: "P-004", 102: "P-002", 103: "P-003"}
	for _, room := range r.Rooms {
		if len(room.Occupants) != 1 || room.Occupants[0] != wantRooms[room.Number] {
			t.Errorf("5.3 habitación %d = %v; se esperaba [%s]", room.Number, room.Occupants, wantRooms[room.Number])
		}
	}
	if r.LastAssignError == "" {
		t.Error("5.3 debe recordar el error de AssignRoom del paso 8")
	}
	// 5.4: ordenado por ID, como en la consola.
	wantSevere := []game.SevereLine{
		{PatientID: "P-001", PatientName: "Yeimy Padilla", Episodes: 1},
		{PatientID: "P-004", PatientName: "Wilfrido Berrío", Episodes: 1},
		{PatientID: "P-005", PatientName: "Breiner Julio", Episodes: 0},
	}
	if len(r.Severe) != len(wantSevere) {
		t.Fatalf("5.4 = %+v; se esperaba %+v", r.Severe, wantSevere)
	}
	for i := range wantSevere {
		if r.Severe[i] != wantSevere[i] {
			t.Errorf("5.4 fila %d = %+v; se esperaba %+v", i, r.Severe[i], wantSevere[i])
		}
	}
}

func TestDemo_NextAfterTheEndReturnsErrDemoFinished(t *testing.T) {
	d := game.NewDemo()
	runSteps(t, d, game.DemoSteps)

	if _, err := d.Next(); !errors.Is(err, game.ErrDemoFinished) {
		t.Fatalf("err = %v; se esperaba ErrDemoFinished", err)
	}
}

func TestDemo_SkipRunsOnlyTheRemainingSteps(t *testing.T) {
	d := game.NewDemo()
	runSteps(t, d, 2)

	if err := d.Skip(); err != nil {
		t.Fatalf("Skip: %v", err)
	}

	if snap := d.Snapshot(); !d.Done() || snap.Step != game.DemoSteps || snap.Last.Number != game.DemoSteps {
		t.Errorf("tras Skip: Done=%v, Step=%d, Last=%d; se esperaba el paso %d", d.Done(), snap.Step, snap.Last.Number, game.DemoSteps)
	}
	if err := d.Skip(); err != nil {
		t.Errorf("Skip con la demo terminada no debe fallar: %v", err)
	}
}
