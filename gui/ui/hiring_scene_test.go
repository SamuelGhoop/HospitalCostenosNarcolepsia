package ui

import (
	"image"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// openHiring abre la bolsa sobre una partida de prueba y avanza la
// animación de abrir hasta que queda quieta (hiringOpen), lista para
// recibir clics y teclas.
func openHiring(t *testing.T) *HiringScene {
	t.Helper()
	under := newTestGameScene(t)
	s := newHiringScene(under)
	for i := 0; i < hiringOpenTicks; i++ {
		if _, err := s.update(input{}); err != nil {
			t.Fatal(err)
		}
	}
	if s.phase != hiringOpen {
		t.Fatalf("fase %v tras %d ticks; se esperaba abierta", s.phase, hiringOpenTicks)
	}
	return s
}

// Abrir la bolsa pausa la partida; cerrarla (Esc) la reanuda y devuelve la
// escena de abajo, no una nueva.
func TestHiring_OpenPausesAndCloseResumes(t *testing.T) {
	under := newTestGameScene(t)
	s := newHiringScene(under)
	if !under.g.Snapshot().Paused {
		t.Fatal("al abrir la bolsa la partida debía quedar en pausa")
	}

	for i := 0; i < hiringOpenTicks; i++ {
		s.update(input{})
	}
	if s.phase != hiringOpen {
		t.Fatalf("fase %v; se esperaba abierta", s.phase)
	}

	next, err := s.update(input{back: true})
	if err != nil {
		t.Fatal(err)
	}
	if next != Scene(s) || s.phase != hiringClosing {
		t.Fatalf("Esc no arrancó el cierre: fase %v, next %T", s.phase, next)
	}
	for i := 0; i < hiringCloseTicks-1; i++ {
		if next, _ = s.update(input{}); next != Scene(s) {
			t.Fatalf("tick %d de %d: ya cerró; debía esperar %d", i, hiringCloseTicks, hiringCloseTicks)
		}
	}
	next, err = s.update(input{})
	if err != nil {
		t.Fatal(err)
	}
	if next != Scene(under) {
		t.Fatalf("al terminar de cerrar devolvió %T; se esperaba el juego de abajo", next)
	}
	if under.g.Snapshot().Paused {
		t.Error("al cerrar la bolsa la partida debía reanudar")
	}
}

// Un clic afuera del tablero (y de las flechas) también cierra la bolsa.
func TestHiring_ClickOutsideCloses(t *testing.T) {
	s := openHiring(t)
	if _, err := s.update(input{click: true, mouse: image.Pt(10, 10)}); err != nil {
		t.Fatal(err)
	}
	if s.phase != hiringClosing {
		t.Errorf("clic afuera del tablero: fase %v; se esperaba cerrando", s.phase)
	}
}

// La bolsa no se puede pausar: Esc, en ella, solo la cierra (lo atiende
// ella misma); App no le abre además el menú de pausa por encima.
func TestHiring_IsNotPausable(t *testing.T) {
	s := openHiring(t)
	if _, ok := Scene(s).(pausable); ok {
		t.Error("la bolsa cumple la interfaz pausable; Esc abriría además la pausa")
	}
}

// ←/→ cambian de hoja con tope: no pasan antes de la primera ni después de
// la última, y una pulsación a mitad de la transición se ignora.
func TestHiring_PagingHasLimits(t *testing.T) {
	s := openHiring(t)
	n := len(s.snap.Candidates)
	if n < 2 {
		t.Fatal("la partida de prueba no trajo al menos 2 candidatos: ajustar la semilla")
	}

	// ← en la primera hoja no hace nada.
	s.update(input{left: true})
	if s.index != 0 {
		t.Errorf("← en la hoja 1: quedó en %d; se esperaba 0", s.index)
	}

	// → recorre todas las hojas.
	for i := 0; i < n-1; i++ {
		s.update(input{right: true})
		for s.pageTicks > 0 {
			s.update(input{})
		}
		if s.index != i+1 {
			t.Fatalf("tras %d →: hoja %d; se esperaba %d", i+1, s.index, i+1)
		}
	}
	// Una → de más, en la última hoja, no hace nada.
	s.update(input{right: true})
	for s.pageTicks > 0 {
		s.update(input{})
	}
	if s.index != n-1 {
		t.Errorf("→ en la última hoja: quedó en %d; se esperaba %d (tope)", s.index, n-1)
	}

	// Mientras una transición está en curso, otra pulsación se ignora.
	s.update(input{left: true})
	if s.pageTicks == 0 {
		t.Fatal("el test necesita que la transición siga en curso para probar que se ignora")
	}
	s.update(input{left: true}) // debería ignorarse: todavía no llega ni a la N-2
	for s.pageTicks > 0 {
		s.update(input{})
	}
	if s.index != n-2 {
		t.Errorf("una ← de más durante la transición: terminó en %d; se esperaba %d", s.index, n-2)
	}
}

// CONTRATAR con plata llama Game.Hire: el candidato sale de la lista de la
// bolsa y entra al personal de la partida.
func TestHiring_HireWithMoneyRemovesCandidateAndAddsStaff(t *testing.T) {
	s := openHiring(t)
	cands := s.snap.Candidates
	if len(cands) == 0 || !cands[0].Affordable {
		t.Fatal("el primer candidato de la partida de prueba no alcanza: ajustar la semilla")
	}
	name := cands[0].Name
	before := len(cands)
	staffBefore := len(s.under.g.Snapshot().Staff)

	center := image.Pt((hiringHireButton.Min.X+hiringHireButton.Max.X)/2, (hiringHireButton.Min.Y+hiringHireButton.Max.Y)/2)
	if _, err := s.update(input{click: true, mouse: center}); err != nil {
		t.Fatal(err)
	}

	if len(s.snap.Candidates) != before-1 {
		t.Errorf("tras contratar quedan %d candidatos; se esperaban %d", len(s.snap.Candidates), before-1)
	}
	for _, c := range s.snap.Candidates {
		if c.Name == name {
			t.Errorf("%s sigue en la bolsa después de contratarlo", name)
		}
	}
	if got := len(s.under.g.Snapshot().Staff); got != staffBefore+1 {
		t.Errorf("el personal tiene %d; se esperaban %d (uno más)", got, staffBefore+1)
	}
}

// Sin plata, CONTRATAR no hace nada: ni llama a Hire ni cambia la lista
// (se revisa directo sobre hire(), sin un *game.Game real: el candidato sin
// plata no debe ni intentar contratarse).
func TestHiring_UnaffordableHireDoesNothing(t *testing.T) {
	s := &HiringScene{phase: hiringOpen, snap: game.Snapshot{
		Candidates: []game.CandidateView{{Name: "Carísimo", Salary: 9999, Cost: 9999, Affordable: false}},
	}}
	s.hire()
	if len(s.snap.Candidates) != 1 {
		t.Error("contrató a un candidato sin plata para pagarle")
	}
}

// BUG (commit 2a6253d): "DRA. YORLEDIS CASSIANI" a 24 px se salía del papel
// por la derecha (pasaba de x = 946). "DRA. YORLEDIS CASSIANI" y
// "DR. DAGOBERTO CANTILLO" son la combinación título+nombre+apellido más
// larga de las listas de GAME_DESIGN §7 (22 letras las dos), ya en
// mayúsculas como las dibuja drawCandidateName. fitCandidateName tiene que
// bajar a 16 px (o partir en dos renglones) para que el ancho dibujado
// quepa en la columna, de x = 526 a x = 946.
func TestFitCandidateName_LongestNameFits(t *testing.T) {
	for _, name := range []string{"DRA. YORLEDIS CASSIANI", "DR. DAGOBERTO CANTILLO"} {
		lines, size := fitCandidateName(name, candidateNameMaxWidth)
		if len(lines) == 0 {
			t.Fatalf("%q: fitCandidateName no devolvió ningún renglón", name)
		}
		for _, l := range lines {
			if w := pixelTextWidth(l, size); w > candidateNameMaxWidth {
				t.Errorf("%q: el renglón %q a %v px mide %d px; no cabe en %d", name, l, size, w, candidateNameMaxWidth)
			}
		}
		// A 24 px (el tamaño de la vista aprobada) este nombre no cabe: si
		// fitCandidateName no hubiera bajado el tamaño, seguiría dando 24.
		if size == 24 {
			t.Errorf("%q: fitCandidateName devolvió 24 px; a ese tamaño no cabe (ese era el bug)", name)
		}
	}
}

// Un nombre que sí cabe a 24 px (el caso normal) se dibuja en un solo
// renglón a 24 px, sin bajarlo de tamaño.
func TestFitCandidateName_ShortNameKeeps24(t *testing.T) {
	lines, size := fitCandidateName("DR. ÉDER POLO", candidateNameMaxWidth)
	if len(lines) != 1 || size != 24 {
		t.Errorf("fitCandidateName(%q) = %v, %v; se esperaba un renglón a 24 px", "DR. ÉDER POLO", lines, size)
	}
}
