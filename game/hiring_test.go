package game_test

import (
	"errors"
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// §6: la bolsa de empleo tiene 3 candidatos desde el día 1.
func TestNew_StartsWithThreeCandidates(t *testing.T) {
	snap := newGame(t).Snapshot()
	if len(snap.Candidates) != 3 {
		t.Fatalf("hay %d candidatos; se esperaban 3 (§6)", len(snap.Candidates))
	}
	for _, c := range snap.Candidates {
		if c.Speed < 1 || c.Speed > 5 {
			t.Errorf("candidato %+v: rapidez fuera de 1..5", c)
		}
		if c.IsDoctor {
			if c.Skill < 1 || c.Skill > 5 {
				t.Errorf("médico %+v: pericia fuera de 1..5", c)
			}
			if c.Specialty == "" {
				t.Errorf("médico %+v: le falta especialidad", c)
			}
		} else if c.Skill != 0 || c.Specialty != "" {
			t.Errorf("camillero %+v: no debe tener pericia ni especialidad", c)
		}
		if c.Cost != c.Salary {
			t.Errorf("candidato %+v: el costo de contratar debe ser el sueldo de un día (§5.8)", c)
		}
		if c.Age < 25 || c.Age > 60 {
			t.Errorf("candidato %+v: edad fuera de 25..60", c)
		}
	}
}

// Hire paga el sueldo de un día por adelantado, contrata al candidato
// (aparece en Snapshot como el resto del personal) y lo saca de la lista
// del día. El nuevo también se puede despachar "de guardia" como cualquier
// otro miembro del personal (§3.2).
func TestHire_PaysAddsStaffAndCanBeDispatched(t *testing.T) {
	g := newGame(t)
	before := g.Snapshot()
	c := before.Candidates[0]

	if err := g.Hire(0); err != nil {
		t.Fatalf("Hire(0): %v", err)
	}
	after := g.Snapshot()

	if got := before.Money - after.Money; got != c.Cost {
		t.Errorf("pagó $%d; se esperaban $%d (el costo del candidato)", got, c.Cost)
	}
	if len(after.Staff) != 3 {
		t.Fatalf("hay %d miembros del personal; se esperaban 3", len(after.Staff))
	}
	if len(after.Candidates) != 2 {
		t.Errorf("quedan %d candidatos; se esperaban 2: el contratado sale de la lista", len(after.Candidates))
	}

	hired := after.Staff[2] // el tercero: los dos primeros son el médico y el camillero iniciales
	if hired.Name != c.Name || hired.IsDoctor != c.IsDoctor || hired.Speed != c.Speed || hired.Skill != c.Skill {
		t.Errorf("el nuevo personal = %+v; se esperaba el candidato contratado %+v", hired, c)
	}
	if hired.Activity != game.Free {
		t.Errorf("el nuevo personal llega ocupado: %v", hired.Activity)
	}

	// "De guardia": se le puede despachar como a cualquiera (§3.2).
	pid, err := g.CollapseForTest("Prueba Dispatch", hospital.Mild, game.Cafeteria)
	if err != nil {
		t.Fatalf("CollapseForTest: %v", err)
	}
	if hired.IsDoctor {
		// Un médico revisa en la habitación; primero hay que recogerlo y
		// llevarlo a la cama con el camillero que ya había.
		if err := g.Dispatch(pid, "C-01"); err != nil {
			t.Fatalf("Dispatch a C-01 (recoger): %v", err)
		}
		tickFor(g, 10*time.Second) // sobra tiempo para que C-01 lo recoja y lo acueste
		if err := g.Dispatch(pid, hired.ID); err != nil {
			t.Errorf("Dispatch al médico contratado para revisar: %v", err)
		}
	} else {
		if err := g.Dispatch(pid, hired.ID); err != nil {
			t.Errorf("Dispatch al camillero contratado para recoger: %v", err)
		}
	}
}

// §6: sin plata, Hire devuelve ErrNotEnoughMoney y no cambia nada.
func TestHire_NotEnoughMoney(t *testing.T) {
	g := newGame(t)
	g.SetMoneyForTest(0)
	before := g.Snapshot()

	err := g.Hire(0)
	if !errors.Is(err, game.ErrNotEnoughMoney) {
		t.Fatalf("Hire sin plata: err = %v; se esperaba ErrNotEnoughMoney", err)
	}

	after := g.Snapshot()
	if after.Money != before.Money {
		t.Errorf("la plata cambió: %d → %d", before.Money, after.Money)
	}
	if len(after.Staff) != len(before.Staff) || len(after.Candidates) != len(before.Candidates) {
		t.Errorf("el personal o los candidatos cambiaron: %+v → %+v", before, after)
	}
}

// §6: índice fuera de la bolsa de empleo (negativo o demasiado grande) es
// ErrNoCandidate, sin cambiar nada.
func TestHire_InvalidIndex(t *testing.T) {
	g := newGame(t)
	before := g.Snapshot()

	for _, i := range []int{-1, 3, 99} {
		if err := g.Hire(i); !errors.Is(err, game.ErrNoCandidate) {
			t.Errorf("Hire(%d): err = %v; se esperaba ErrNoCandidate", i, err)
		}
	}

	after := g.Snapshot()
	if after.Money != before.Money || len(after.Staff) != len(before.Staff) {
		t.Errorf("un índice inválido cambió la partida: %+v → %+v", before, after)
	}
}

// §6: máximo 6 miembros del personal. Con plata de sobra y candidatos
// renovados a mano (RegenerateCandidatesForTest, ver export_test.go), el
// quinto intento (2 iniciales + 4 contrataciones) debe fallar.
func TestHire_MaxSixStaff(t *testing.T) {
	g := newGame(t)
	g.SetMoneyForTest(1_000_000)

	for i := 0; i < 4; i++ { // 2 iniciales + 4 = 6
		g.RegenerateCandidatesForTest()
		if err := g.Hire(0); err != nil {
			t.Fatalf("contratación %d: %v", i+1, err)
		}
	}
	if n := len(g.Snapshot().Staff); n != 6 {
		t.Fatalf("hay %d miembros del personal; se esperaban 6", n)
	}

	before := g.Snapshot()
	g.RegenerateCandidatesForTest()
	if err := g.Hire(0); !errors.Is(err, game.ErrStaffFull) {
		t.Fatalf("con 6 contratados: err = %v; se esperaba ErrStaffFull", err)
	}
	after := g.Snapshot()
	if after.Money != before.Money || len(after.Staff) != 6 {
		t.Errorf("ErrStaffFull cambió la partida: %+v → %+v", before, after)
	}
}

// Con la partida perdida, Hire devuelve ErrGameOver (igual que Dispatch y
// WakeEarly, §5.10) y no cobra ni contrata.
func TestHire_AfterGameOverIsErrGameOver(t *testing.T) {
	g := newGame(t)
	for _, id := range []string{"P-001", "P-002", "P-003", "P-004", "P-005"} {
		collapse(t, g, id, "Paciente "+id, hospital.Mild, game.Cafeteria)
	}
	g.Tick(tick) // colapsa: 5 desplomados a la vez

	if over := g.Snapshot().GameOver; over != game.HospitalCollapsed {
		t.Fatalf("GameOver = %v; se esperaba el colapso", over)
	}
	before := g.Snapshot()
	if err := g.Hire(0); !errors.Is(err, game.ErrGameOver) {
		t.Fatalf("Hire con la partida perdida: err = %v; se esperaba ErrGameOver", err)
	}
	after := g.Snapshot()
	if after.Money != before.Money || len(after.Staff) != len(before.Staff) {
		t.Errorf("Hire cambió la partida perdida: %+v → %+v", before, after)
	}
}

// §6 y §7: los candidatos son deterministas con la misma semilla (mismo
// azar de la partida) y cambian al empezar el día siguiente.
func TestCandidates_AreDeterministicAndChangeTheNextDay(t *testing.T) {
	a := newQuietGame(t).Snapshot().Candidates
	b := newQuietGame(t).Snapshot().Candidates
	if len(a) != 3 || len(b) != 3 {
		t.Fatalf("se esperaban 3 candidatos en cada partida: %d y %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("con la misma semilla, el candidato %d salió distinto: %+v vs %+v", i, a[i], b[i])
		}
	}

	g := newQuietGame(t)
	day1 := g.Snapshot().Candidates
	tickFor(g, 300*time.Second) // llega a las 20:00
	if err := g.StartNextDay(); err != nil {
		t.Fatalf("StartNextDay: %v", err)
	}
	day2 := g.Snapshot().Candidates

	same := len(day1) == len(day2)
	for i := 0; same && i < len(day1); i++ {
		same = day1[i] == day2[i]
	}
	if same {
		t.Errorf("los candidatos del día 2 son iguales a los del día 1: %+v", day1)
	}
}

// §5.8 y §5.10: la nómina se paga al llegar a las 20:00 (médico inicial
// $550, camillero inicial $300: total $850) y queda en el DayReport del
// día, junto con la plata inicial y final y las cuatro consultas.
func TestPayroll_PaidAt2000AndInTheDayReport(t *testing.T) {
	g := newQuietGame(t)
	tickFor(g, 300*time.Second)

	if got := g.Snapshot().Money; got != 2000-850 {
		t.Fatalf("plata después de la nómina = $%d; se esperaban $%d", got, 2000-850)
	}

	r := g.DayReport()
	if r.Day != 1 || r.StartMoney != 2000 || r.EndMoney != 2000-850 || r.PayrollTotal != 850 {
		t.Errorf("DayReport = %+v; se esperaba día 1, $2.000 → $1.150, nómina $850", r)
	}
	if len(r.Payroll) != 2 {
		t.Fatalf("la nómina tiene %d líneas; se esperaban 2", len(r.Payroll))
	}
	wantSalaries := map[bool]int{true: 550, false: 300} // true = médico
	for _, line := range r.Payroll {
		if want := wantSalaries[line.IsDoctor]; line.Salary != want {
			t.Errorf("nómina de %q (médico=%v) = $%d; se esperaban $%d", line.Name, line.IsDoctor, line.Salary, want)
		}
	}
	if r.ReviewIncome != 0 || r.DischargeIncome != 0 || len(r.Hires) != 0 {
		t.Errorf("sin pacientes ni contrataciones: DayReport = %+v; se esperaban todos en 0", r)
	}
	if len(r.Report.Rooms) != 3 {
		t.Errorf("el DayReport debe traer las 4 consultas (acá, las habitaciones): %+v", r.Report)
	}
}

// §5.8: ingresos del día y contrataciones también quedan en el DayReport.
func TestPayroll_DayReportTracksIncomeAndHires(t *testing.T) {
	g := newQuietGame(t)
	reviewAt(t, g, hospital.Mild, 10*time.Second) // +$450, revisión
	tickFor(g, 11400*time.Millisecond)            // duerme y recibe el alta: +$200

	c := g.Snapshot().Candidates[0]
	if err := g.Hire(0); err != nil {
		t.Fatalf("Hire: %v", err)
	}

	tickFor(g, 300*time.Second) // de sobra para llegar a las 20:00 (los ticks de más no hacen nada)

	r := g.DayReport()
	if r.ReviewIncome != 450 || r.DischargeIncome != 200 {
		t.Errorf("ingresos del día = (revisión %d, alta %d); se esperaban (450, 200)", r.ReviewIncome, r.DischargeIncome)
	}
	if len(r.Hires) != 1 || r.Hires[0].Name != c.Name || r.Hires[0].Cost != c.Cost {
		t.Errorf("contrataciones del día = %+v; se esperaba una, %q por $%d", r.Hires, c.Name, c.Cost)
	}
}

// §5.8 y §5.10: si la nómina deja la plata en negativo, la partida pierde
// por quiebra y queda congelada como las otras derrotas.
func TestPayroll_BankruptcyEndsTheGame(t *testing.T) {
	g := newQuietGame(t)
	tickFor(g, 299*time.Second+800*time.Millisecond) // todavía no son las 20:00
	g.SetMoneyForTest(100)                           // 100 − 850 de nómina = −750
	tickFor(g, 300*time.Millisecond)                 // cruza las 20:00

	snap := g.Snapshot()
	if snap.GameOver != game.Bankrupt || snap.GameOver.String() != "¡QUIEBRA!" {
		t.Fatalf("GameOver = %v; se esperaba la quiebra", snap.GameOver)
	}
	if snap.Money != 100-850 {
		t.Errorf("plata final = $%d; se esperaban $%d", snap.Money, 100-850)
	}

	if err := g.StartNextDay(); !errors.Is(err, game.ErrGameOver) {
		t.Errorf("StartNextDay con quiebra: err = %v; se esperaba ErrGameOver", err)
	}
	if err := g.Hire(0); !errors.Is(err, game.ErrGameOver) {
		t.Errorf("Hire con quiebra: err = %v; se esperaba ErrGameOver", err)
	}
}
