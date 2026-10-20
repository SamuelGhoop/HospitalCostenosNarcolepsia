package game_test

import (
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// shiftLength es lo que dura un turno en tiempo real (§5.1).
const shiftLength = 300 * time.Second

// playGreedily despacha, en cada tick, a cualquier miembro libre del
// personal hacia el primer paciente que lo necesite. Es un jugador
// automático tosco: solo sirve para llenar camas y ocupar al personal antes
// de cerrar el día.
func playGreedily(g *game.Game) {
	snap := g.Snapshot()
	for _, p := range snap.Patients {
		if p.Stage != game.Collapsed && p.Stage != game.InHallway && p.Stage != game.AwaitingReview {
			continue
		}
		for _, s := range snap.Staff {
			if s.Activity == game.Free && g.Dispatch(p.ID, s.ID) == nil {
				snap = g.Snapshot()
				break
			}
		}
	}
}

// playUntilDayOver juega el día completo con playGreedily y contrata a 3
// más al empezar, para que la partida no se pierda antes de las 20:00.
func playUntilDayOver(t *testing.T, g *game.Game) {
	t.Helper()
	g.SetMoneyForTest(100000)
	for i := 0; i < 3; i++ {
		if err := g.Hire(0); err != nil {
			t.Fatalf("Hire: %v", err)
		}
	}
	for !g.Snapshot().DayOver {
		if g.Snapshot().GameOver != game.NotOver {
			t.Fatal("la partida se perdió antes de las 20:00")
		}
		playGreedily(g)
		g.Tick(tick)
	}
}

func TestStartNextDay_EmptiesTheHospital(t *testing.T) {
	g := newGame(t)
	playUntilDayOver(t, g)

	before := g.Snapshot()
	if len(before.Patients) == 0 {
		t.Fatal("la prueba necesita pacientes en el mapa a las 20:00")
	}
	busy := false
	for _, s := range before.Staff {
		busy = busy || s.Activity != game.Free
	}
	if !busy {
		t.Fatal("la prueba necesita a alguien del personal ocupado a las 20:00")
	}

	if err := g.StartNextDay(); err != nil {
		t.Fatalf("StartNextDay: %v", err)
	}

	snap := g.Snapshot()
	if len(snap.Patients) != 0 {
		t.Errorf("día 2: quedaron %d pacientes en el mapa; se esperaba ninguno", len(snap.Patients))
	}
	for _, r := range snap.Rooms {
		if len(r.Occupants) != 0 {
			t.Errorf("día 2: la habitación %d sigue ocupada por %v", r.Number, r.Occupants)
		}
	}
	if hall := g.Report().Hallway; len(hall) != 0 {
		t.Errorf("día 2: el modelo todavía tiene %d en el pasillo", len(hall))
	}
	if len(snap.Notices) != 0 {
		t.Errorf("día 2: quedaron avisos del día anterior: %v", snap.Notices)
	}
	if snap.Money != before.Money || snap.Reputation != before.Reputation || snap.Score != before.Score {
		t.Errorf("día 2: (plata, reputación, puntaje) = (%d, %d, %d); se esperaba lo mismo que al cerrar: (%d, %d, %d)",
			snap.Money, snap.Reputation, snap.Score, before.Money, before.Reputation, before.Score)
	}
}

func TestStartNextDay_BackToStartingStaff(t *testing.T) {
	g := newGame(t)
	playUntilDayOver(t, g)
	if err := g.StartNextDay(); err != nil {
		t.Fatalf("StartNextDay: %v", err)
	}

	snap := g.Snapshot()
	var ids []string
	for _, s := range snap.Staff {
		ids = append(ids, s.ID)
		if s.Activity != game.Free || s.PatientID != "" {
			t.Errorf("día 2: %s empieza ocupado (%v, %q); se esperaba libre", s.ID, s.Activity, s.PatientID)
		}
	}
	if len(ids) != 2 || ids[0] != "D-01" || ids[1] != "C-01" {
		t.Errorf("día 2: personal = %v; se esperaba solo el inicial [D-01 C-01]", ids)
	}
}

func TestHire_AfterNewDayKeepsCountingIDs(t *testing.T) {
	g := newGame(t)
	playUntilDayOver(t, g) // contrata a 3 el día 1
	if err := g.StartNextDay(); err != nil {
		t.Fatalf("StartNextDay: %v", err)
	}
	g.SetMoneyForTest(100000)
	if err := g.Hire(0); err != nil {
		t.Fatalf("Hire el día 2: %v", err)
	}

	staff := g.Snapshot().Staff
	got := staff[len(staff)-1].ID
	for _, s := range staff[:len(staff)-1] {
		if s.ID == got {
			t.Fatalf("el contratado del día 2 repite el ID %s", got)
		}
	}
	if got == "D-02" || got == "C-02" {
		// El día 1 ya se usaron IDs después de D-01 y C-01: el modelo
		// todavía los tiene, así que no se pueden repetir.
		t.Errorf("el contratado del día 2 es %s: repite un ID que ya usó el día 1", got)
	}
}

func TestHireCost_GrowsEachDayUpToTheCap(t *testing.T) {
	g := newQuietGame(t)
	g.SetMoneyForTest(1000000)
	for day := 1; day <= 12; day++ {
		snap := g.Snapshot()
		percent := 100 + 20*(day-1)
		if percent > 300 {
			percent = 300
		}
		for _, c := range snap.Candidates {
			want := c.Salary * percent / 100
			if c.Cost != want {
				t.Errorf("día %d: %s cuesta %d; se esperaba %d (sueldo %d × %d %%)", day, c.Name, c.Cost, want, c.Salary, percent)
			}
		}
		tickFor(g, shiftLength)
		if err := g.StartNextDay(); err != nil {
			t.Fatalf("StartNextDay del día %d: %v", day, err)
		}
	}
}

func TestHire_ChargesTheDayCost(t *testing.T) {
	g := newQuietGame(t)
	tickFor(g, shiftLength)
	if err := g.StartNextDay(); err != nil { // día 2: ×1,2
		t.Fatalf("StartNextDay: %v", err)
	}
	g.SetMoneyForTest(100000)
	cost := g.Snapshot().Candidates[0].Cost
	if err := g.Hire(0); err != nil {
		t.Fatalf("Hire: %v", err)
	}
	if money := g.Snapshot().Money; money != 100000-cost {
		t.Errorf("plata después de contratar = %d; se esperaba %d (cobra el costo del día, %d)", money, 100000-cost, cost)
	}
}
