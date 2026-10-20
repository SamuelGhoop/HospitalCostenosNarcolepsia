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
