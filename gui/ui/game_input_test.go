package ui

import (
	"image"
	"testing"
)

// BLOQUE 1 del pase de juice (§12.10): la hitbox agrandada atrapa un clic
// que antes caía justo afuera de figureBox, y el imán atrapa uno todavía
// más lejos (dentro de clickMagnetRadius) cuando no hay nadie más cerca.
func TestFigureAt_GrownHitboxCatchesAClickJustOutsideTheOldBox(t *testing.T) {
	s := newTestGameScene(t)
	f, ok := s.figure("C-01")
	if !ok {
		t.Fatal("C-01 no está en la escena")
	}
	box := s.figureBox(f)

	// Un punto justo 1 px afuera del borde derecho de la caja original: con
	// la hitbox tal cual no pegaba, agrandada +50% sí.
	p := image.Pt(box.Max.X+1, box.Min.Y+box.Dy()/2)
	if p.In(box) {
		t.Fatalf("el punto de prueba %v ya estaba dentro de la caja original %v", p, box)
	}
	got, ok := s.figureAt(p)
	if !ok || got.id != "C-01" {
		t.Errorf("figureAt(%v) = (%q, %v); se esperaba (C-01, true) por la hitbox agrandada", p, got.id, ok)
	}
}

func TestFigureAt_MagnetCatchesAClickWithinRadiusOfTheNearestFigure(t *testing.T) {
	s := newTestGameScene(t)
	f, ok := s.figure("C-01")
	if !ok {
		t.Fatal("C-01 no está en la escena")
	}
	cx, cy := boxCenter(s.figureBox(f))

	// clickMagnetRadius (40 px) a la derecha del centro, bien afuera de la
	// hitbox agrandada (que mide unos 32 px de ancho): solo el imán lo agarra.
	p := image.Pt(int(cx)+int(clickMagnetRadius)-2, int(cy))
	if p.In(growRect(s.figureBox(f), hitboxGrowFactor)) {
		t.Fatalf("el punto de prueba %v ya caía en la hitbox agrandada", p)
	}
	got, ok := s.figureAt(p)
	if !ok || got.id != "C-01" {
		t.Errorf("figureAt(%v) = (%q, %v); se esperaba que el imán eligiera a C-01", p, got.id, ok)
	}
}

// BLOQUE 2 del pase de juice (§12.10): un despacho exitoso deja la línea
// punteada hasta el paciente, y un clic inválido tiembla la pantalla y
// deja el destello rojo sobre a quien se le clicó.
func TestClick_SuccessfulDispatchLeavesADashLine(t *testing.T) {
	s := newTestGameScene(t)
	id := firstCollapsed(t, s)
	if n := len(s.fx.lines); n != 0 {
		t.Fatalf("ya había %d líneas antes del despacho", n)
	}
	clickOn(s, id, false)
	clickOn(s, "C-01", false)
	if n := len(s.fx.lines); n != 1 {
		t.Fatalf("después de un despacho exitoso hay %d líneas; se esperaba 1", n)
	}
	if p := s.fx.punches["C-01"]; p == nil {
		t.Error("C-01 no quedó con su squash&stretch tras el clic")
	}
}

func TestClick_InvalidDispatchShakesAndFlashesRed(t *testing.T) {
	s := newTestGameScene(t)
	id := firstCollapsed(t, s)
	// Con C-01 (camillero) libre, mandar al médico a recoger es inválido
	// (§5.5: solo se manda al médico si no hay camillero libre).
	clickOn(s, id, false)
	clickOn(s, "D-01", false)
	if s.fx.shake.current() <= 0 {
		t.Error("un despacho inválido no sacudió la pantalla")
	}
	p := s.fx.punches["D-01"]
	if p == nil || p.flashColor != colRed {
		t.Errorf("destello de D-01 = %+v; se esperaba rojo (clic inválido)", p)
	}
}

func TestFigureAt_BeyondMagnetRadiusHitsNobody(t *testing.T) {
	s := newTestGameScene(t)
	f, ok := s.figure("C-01")
	if !ok {
		t.Fatal("C-01 no está en la escena")
	}
	cx, cy := boxCenter(s.figureBox(f))
	p := image.Pt(int(cx)+int(clickMagnetRadius)+200, int(cy))
	if _, ok := s.figureAt(p); ok {
		t.Errorf("figureAt(%v), lejísimos de todos, encontró a alguien; no se esperaba nadie", p)
	}
}
