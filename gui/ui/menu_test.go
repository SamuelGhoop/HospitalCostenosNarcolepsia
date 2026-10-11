package ui

import (
	"image"
	"testing"
)

// press arma la entrada de un tick con una tecla apretada.
func press(up, down, confirm bool) input { return input{up: up, down: down, confirm: confirm} }

// Sin partida guardada (hasta F6) el menú arranca en NUEVA PARTIDA y las
// flechas dan la vuelta saltándose CONTINUAR, que sale en gris.
func TestMenu_StartsOnNewGameAndSkipsContinue(t *testing.T) {
	m := newMenu(false)
	if m.selected != optNew {
		t.Fatalf("arranca en %v; se esperaba NUEVA PARTIDA", m.selected)
	}
	for _, want := range []option{optDemo, optSettings, optCredits, optExit, optNew} {
		m.update(press(false, true, false), nil)
		if m.selected != want {
			t.Errorf("↓: quedó en %v; se esperaba %v", m.selected, want)
		}
	}
	m.update(press(true, false, false), nil)
	if m.selected != optExit {
		t.Errorf("↑ desde NUEVA PARTIDA: quedó en %v; se esperaba SALIR (da la vuelta sin pasar por CONTINUAR)", m.selected)
	}

	saved := newMenu(true) // con partida guardada (F6), CONTINUAR sí se elige
	if saved.selected != optContinue {
		t.Errorf("con partida guardada arranca en %v; se esperaba CONTINUAR", saved.selected)
	}
}

// El cursor alterna sus 2 cuadros cada 0,35 s (21 ticks).
func TestMenu_CursorAlternatesEvery21Ticks(t *testing.T) {
	m := newMenu(false)
	for tick := 0; tick < 64; tick++ {
		if want := tick / 21 % 2; m.cursorFrame() != want {
			t.Fatalf("tick %d: cuadro %d; se esperaba %d", tick, m.cursorFrame(), want)
		}
		m.update(input{}, nil)
	}
}

// Al confirmar, el cursor se queda quieto en el cuadro 0 durante 0,45 s
// (27 ticks) y recién ahí sale la acción. Mientras tanto no se mueve.
func TestMenu_ConfirmFreezesTheCursorThenActs(t *testing.T) {
	m := newMenu(false)
	for i := 0; i < 21; i++ { // que el cursor esté en el cuadro 1
		m.update(input{}, nil)
	}
	if _, done := m.update(press(false, false, true), nil); done {
		t.Fatal("la acción salió de una; debía esperar 27 ticks")
	}
	for tick := 1; tick < 27; tick++ {
		if m.cursorFrame() != 0 {
			t.Fatalf("confirmando, tick %d: el cursor está en el cuadro %d; debía quedarse en el 0", tick, m.cursorFrame())
		}
		if _, done := m.update(press(false, true, false), nil); done {
			t.Fatalf("la acción salió a los %d ticks; debía esperar 27", tick)
		}
	}
	chosen, done := m.update(input{}, nil)
	if !done || chosen != optNew {
		t.Errorf("a los 27 ticks: (%v, %v); se esperaba (NUEVA PARTIDA, true), sin que ↓ la cambiara", chosen, done)
	}
}

// Sin partida guardada se usan las imágenes con CONTINUAR en gris.
func TestMenu_ImageForTheSelection(t *testing.T) {
	m := newMenu(false)
	if got, want := m.image(), "title/clipboard/clipboard_sincontinuar_nueva_0.png"; got != want {
		t.Errorf("imagen = %q; se esperaba %q", got, want)
	}
	if got, want := clipboardPath(optCredits, true, 1), "title/clipboard/clipboard_creditos_1.png"; got != want {
		t.Errorf("con partida guardada: %q; se esperaba %q", got, want)
	}
}

// hitboxes.json trae los 4 vértices de cada renglón inclinado: un punto
// adentro del polígono elige ese renglón.
func TestHitboxes_PointInsideEachRow(t *testing.T) {
	boxes, err := loadHitboxes()
	if err != nil {
		t.Fatal(err)
	}
	if len(boxes) != len(menuOptions) {
		t.Fatalf("hay %d polígonos; se esperaban %d", len(boxes), len(menuOptions))
	}
	for _, o := range menuOptions {
		poly := boxes[o]
		var c image.Point
		for _, p := range poly {
			c = c.Add(p)
		}
		c = c.Div(len(poly))
		if got, ok := boxes.at(c); !ok || got != o {
			t.Errorf("el centro de %v (%v) cae en (%v, %v)", o, c, got, ok)
		}
	}
	if _, ok := boxes.at(image.Pt(10, 10)); ok {
		t.Error("un punto lejos del portapapeles cayó en un renglón")
	}
	// El borde de arriba de CONTINUAR baja hacia la derecha (de y = 368 a
	// 381): en x = 1150 queda en y ≈ 380,7.
	if _, ok := boxes.at(image.Pt(1150, 378)); ok {
		t.Error("(1150, 378) está por encima del borde inclinado de CONTINUAR y cayó en un renglón")
	}
	if got, ok := boxes.at(image.Pt(1150, 383)); !ok || got != optContinue {
		t.Errorf("(1150, 383) está justo debajo del borde inclinado: cayó en (%v, %v)", got, ok)
	}
}

// Mover el mouse sobre un renglón lo selecciona (menos CONTINUAR, que está
// en gris), y el clic lo confirma.
func TestMenu_MouseHoverAndClick(t *testing.T) {
	boxes, err := loadHitboxes()
	if err != nil {
		t.Fatal(err)
	}
	center := func(o option) image.Point {
		var c image.Point
		for _, p := range boxes[o] {
			c = c.Add(p)
		}
		return c.Div(len(boxes[o]))
	}

	m := newMenu(false)
	m.update(input{mouse: center(optDemo), mouseMoved: true}, boxes)
	if m.selected != optDemo {
		t.Errorf("con el mouse sobre DEMO quedó %v", m.selected)
	}
	m.update(input{mouse: center(optContinue), mouseMoved: true}, boxes)
	if m.selected != optDemo {
		t.Errorf("el mouse sobre CONTINUAR (en gris) cambió la selección a %v", m.selected)
	}
	m.update(input{mouse: center(optCredits), click: true}, boxes)
	var chosen option
	done := false
	for i := 0; i < 27 && !done; i++ {
		chosen, done = m.update(input{}, boxes)
	}
	if !done || chosen != optCredits {
		t.Errorf("el clic en CRÉDITOS dio (%v, %v)", chosen, done)
	}
}
