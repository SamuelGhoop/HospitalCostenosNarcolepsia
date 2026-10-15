package game

// Tests internos (package game, no game_test): revisan cosas que no se ven
// desde afuera del paquete, como el hospital que está dentro de Game.

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// newInternalGame es el newGame de los tests internos (semilla fija).
func newInternalGame(t *testing.T) *Game {
	t.Helper()
	g, err := New(rand.New(rand.NewSource(1)), rand.New(rand.NewSource(2)))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return g
}

// Invariante de §3.1: en el hospital del juego no hay ningún Attender sin
// envolver. Un Orderly suelto siempre dice IsAvailable() == true y el
// round-robin se lo llevaría en cada despacho.
func TestNew_EveryAttenderInTheHospitalIsWrapped(t *testing.T) {
	g := newInternalGame(t)

	staff := g.h.Staff()
	if len(staff) != 2 {
		t.Fatalf("el hospital tiene %d miembros del personal; se esperaban 2", len(staff))
	}
	for _, a := range staff {
		if _, ok := a.(*StaffMember); !ok {
			t.Errorf("%s (%T) no está envuelto en StaffMember", a.ID(), a)
		}
	}

	// HireStaff solo registra como médico a un *hospital.Doctor real: los
	// médicos envueltos NO salen en Doctors() (por eso Game guarda su lista).
	if n := len(g.h.Doctors()); n != 0 {
		t.Errorf("Doctors() tiene %d médicos; los envueltos no deberían salir ahí", n)
	}
}

// Los médicos llevan "Dra." o "Dr." según la lista de la que sale el nombre.
func TestDoctorName_TitleMatchesTheFirstName(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 200; i++ {
		name := doctorName(rng)
		parts := strings.Fields(name) // por ejemplo ["Dra.", "Yeimy", "Polo"]
		if len(parts) != 3 {
			t.Fatalf("%q: se esperaba título, nombre y apellido", name)
		}
		title, first, last := parts[0], parts[1], parts[2]
		switch {
		case slices.Contains(femaleFirstNames, first) && title != "Dra.":
			t.Errorf("%q: con %s el título es Dra.", name, first)
		case slices.Contains(maleFirstNames, first) && title != "Dr.":
			t.Errorf("%q: con %s el título es Dr.", name, first)
		case !slices.Contains(femaleFirstNames, first) && !slices.Contains(maleFirstNames, first):
			t.Errorf("%q: %s no está en ninguna de las dos listas", name, first)
		}
		if !slices.Contains(surnames, last) {
			t.Errorf("%q: %s no está en la lista de apellidos", name, last)
		}
	}
}

// §3.2: el "de guardia" se apaga aunque RegisterEpisode falle. Si no, ese
// miembro del personal quedaría disponible para siempre y el round-robin se
// lo llevaría en los despachos siguientes. El error tampoco se pierde: sale
// como aviso.
func TestPickUp_OnCallIsOffEvenIfRegisterEpisodeFails(t *testing.T) {
	g := newInternalGame(t)

	// Un paciente que el hospital nunca admitió: RegisterEpisode falla con
	// ErrNotAdmitted. En el juego no pasa, pero así se prueba el camino del error.
	ghost := &patient{p: hospital.NewPatient("P-404", "Fantasma", 30, hospital.Mild), stage: Collapsed, zone: Cafeteria}
	g.patients = append(g.patients, ghost)

	if err := g.Dispatch("P-404", "C-01"); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	for i := 0; i < 50; i++ { // 5 s: C-01 llega y RegisterEpisode falla
		g.Tick(tickInterval)
	}

	orderly := g.staff[1]
	if orderly.onCall {
		t.Error("C-01 quedó de guardia después de que RegisterEpisode falló")
	}
	if orderly.job != nil {
		t.Errorf("C-01 debe quedar libre; está %v", orderly.job.activity)
	}
	if ghost.stage != Collapsed {
		t.Errorf("P-404 quedó %v; debe seguir desplomado para que el jugador vuelva a intentarlo", ghost.stage)
	}
	if len(g.notices) != 1 || !strings.Contains(g.notices[0], hospital.ErrNotAdmitted.Error()) {
		t.Errorf("avisos = %q; se esperaba uno con el error del modelo", g.notices)
	}
}

// Los avisos que ve la interfaz son solo los últimos maxNotices.
func TestNotices_KeepsOnlyTheLastOnes(t *testing.T) {
	g := newInternalGame(t)
	for i := 1; i <= maxNotices+2; i++ {
		g.noticeLocked(fmt.Sprintf("aviso %d", i))
	}
	if len(g.notices) != maxNotices {
		t.Fatalf("hay %d avisos; se esperaban %d", len(g.notices), maxNotices)
	}
	if first := g.notices[0]; !strings.HasSuffix(first, "aviso 3") {
		t.Errorf("el aviso más viejo que queda es %q; se esperaba el 3", first)
	}
	if want := fmt.Sprintf("08:00 aviso %d", maxNotices+2); g.notices[maxNotices-1] != want {
		t.Errorf("el último aviso es %q; se esperaba %q, con la hora del juego", g.notices[maxNotices-1], want)
	}
}

// Mientras el camillero lleva al paciente a la cama ya se puede mandar al
// médico a revisarlo. Como el traslado dura (8 − rapidez) s del camillero
// (2026-10-11), un médico rápido puede llegar antes que la camilla: entonces
// espera junto a la cama y lo revisa cuando termina el traslado.
func TestDispatch_TheDoctorWaitsForTheCarryToEnd(t *testing.T) {
	g, err := New(rand.New(rand.NewSource(1)), rand.New(rand.NewSource(2)))
	if err != nil {
		t.Fatal(err)
	}
	g.StopArrivalsForTest()
	g.staff[0].speed, g.staff[1].speed = maxStat, 1 // médico rápido (3 s), camillero lento (7 s)
	id, err := g.CollapseForTest("Yeimy Padilla", hospital.Severe, Cafeteria)
	if err != nil {
		t.Fatal(err)
	}
	ticks := func(d time.Duration) {
		for elapsed := time.Duration(0); elapsed < d; elapsed += 100 * time.Millisecond {
			g.Tick(100 * time.Millisecond)
		}
	}
	if err := g.Dispatch(id, "C-01"); err != nil {
		t.Fatal(err)
	}
	ticks(7 * time.Second) // llega, lo carga y empieza a llevarlo (7 s)
	if err := g.Dispatch(id, "D-01"); err != nil {
		t.Fatal(err)
	}
	ticks(3 * time.Second) // el médico ya llegó; a la camilla le faltan 4 s
	if pt := g.patients[0]; pt.stage != AwaitingReview || g.staff[0].job == nil {
		t.Fatalf("con la camilla en camino P-001 está %v y el médico %v; se esperaba que el médico esperara", pt.stage, g.staff[0].job)
	}
	ticks(4*time.Second + 100*time.Millisecond) // termina el traslado y el médico lo revisa
	if pt := g.patients[0]; pt.stage != InBed || pt.reviewedBy != g.staff[0] || g.staff[0].job != nil {
		t.Errorf("después del traslado P-001 está %v (revisado por %v) y el médico %v; se esperaba revisado y el médico libre", pt.stage, pt.reviewedBy, g.staff[0].job)
	}
}
