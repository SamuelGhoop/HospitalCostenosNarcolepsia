package game_test

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// §5.2: la partida arranca con 1 médico (rapidez 3, pericia 3) y 1 camillero
// (rapidez 3), los dos libres.
func TestNew_HiresOneDoctorAndOneOrderly(t *testing.T) {
	snap := newGame(t).Snapshot()
	if len(snap.Staff) != 2 {
		t.Fatalf("hay %d miembros del personal; se esperaban 2 (§5.2)", len(snap.Staff))
	}

	doc := snap.Staff[0]
	if doc.ID != "D-01" || !doc.IsDoctor || doc.Speed != 3 || doc.Skill != 3 || doc.Activity != game.Free {
		t.Errorf("médico inicial = %+v; se esperaba D-01, rapidez 3, pericia 3, libre", doc)
	}
	if !strings.HasPrefix(doc.Name, "Dr. ") && !strings.HasPrefix(doc.Name, "Dra. ") {
		t.Errorf("el nombre del médico debe empezar con Dr. o Dra.: %q", doc.Name)
	}

	ord := snap.Staff[1]
	if ord.ID != "C-01" || ord.IsDoctor || ord.Speed != 3 || ord.Skill != 0 || ord.Activity != game.Free {
		t.Errorf("camillero inicial = %+v; se esperaba C-01, rapidez 3, sin pericia, libre", ord)
	}
	if len(strings.Fields(ord.Name)) != 2 {
		t.Errorf("el camillero debe tener nombre y apellido: %q", ord.Name)
	}
}

// Los nombres salen del *rand.Rand de la partida: con la misma semilla se
// repiten (tests repetibles) y con otra semilla cambian.
func TestNew_StaffNamesComeFromTheGameRandomness(t *testing.T) {
	a, b := newGame(t).Snapshot().Staff, newGame(t).Snapshot().Staff
	if a[0].Name != b[0].Name || a[1].Name != b[1].Name {
		t.Errorf("con la misma semilla salieron nombres distintos: %q/%q y %q/%q", a[0].Name, a[1].Name, b[0].Name, b[1].Name)
	}

	names := map[string]bool{}
	for seed := int64(1); seed <= 20; seed++ {
		g, err := game.New(rand.New(rand.NewSource(seed)), rand.New(rand.NewSource(seed)))
		if err != nil {
			t.Fatalf("New(semilla %d): %v", seed, err)
		}
		names[g.Snapshot().Staff[0].Name] = true
	}
	if len(names) < 2 {
		t.Errorf("con 20 semillas distintas el médico siempre se llamó igual: %v", names)
	}
}

// Cada miembro del personal tiene su apariencia (la que dibuja la interfaz;
// en F4 el candidato de la bolsa es el mismo que camina). Sale de un azar
// cosmético aparte: con otra semilla cosmética cambia la apariencia pero no
// los nombres, que salen del azar de la partida (así los tests con semilla
// fija no cambian). Y no cambia de un momento a otro.
func TestNew_StaffLooksComeFromTheirOwnRandomness(t *testing.T) {
	newWithLooks := func(looks int64) game.Snapshot {
		g, err := game.New(rand.New(rand.NewSource(1)), rand.New(rand.NewSource(looks)))
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		return g.Snapshot()
	}
	base := newWithLooks(1)
	differs := make([]bool, len(base.Staff)) // la apariencia de CADA uno sale del azar cosmético
	for seed := int64(2); seed <= 20; seed++ {
		snap := newWithLooks(seed)
		for i, s := range snap.Staff {
			if s.Name != base.Staff[i].Name {
				t.Fatalf("con otra semilla cosmética %s se llama %q (antes %q): la apariencia no debe tocar el azar de la partida", s.ID, s.Name, base.Staff[i].Name)
			}
			differs[i] = differs[i] || s.Look != base.Staff[i].Look
		}
	}
	for i, d := range differs {
		if !d {
			t.Errorf("con 20 semillas cosméticas %s siempre se vio igual: su apariencia no sale del azar cosmético", base.Staff[i].ID)
		}
	}
	if again := newWithLooks(1); again.Staff[0].Look != base.Staff[0].Look {
		t.Error("con la misma semilla cosmética el médico se ve distinto")
	}

	g := newGame(t)
	before := g.Snapshot().Staff[0].Look
	tickFor(g, 30*time.Second)
	if after := g.Snapshot().Staff[0].Look; after != before {
		t.Errorf("la apariencia del médico cambió con el tiempo: %+v → %+v", before, after)
	}
}
