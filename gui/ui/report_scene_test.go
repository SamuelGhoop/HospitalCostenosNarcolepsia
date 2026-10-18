package ui

import (
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// El Shift Report abre animado: arranca abriendo en openness 0, no acepta
// Espacio ni clic hasta llegar a panelOpen, y ahí la openness es 1 (mismo
// comportamiento que la pausa, §10.10).
func TestReport_OpenAnimationPhasesAndOpenness(t *testing.T) {
	r := NewReportScene(game.Report{})
	if r.anim.phase != panelOpening || r.anim.openness() != 0 {
		t.Fatalf("recién creado: fase %v, openness %v; se esperaba abriendo en 0", r.anim.phase, r.anim.openness())
	}
	if r.anim.acceptsInput() {
		t.Error("recién creado acepta entrada; mientras abre no debería")
	}
	for i := 0; i < reportOpenTicks/2; i++ {
		r.update(input{})
	}
	if o := r.anim.openness(); o <= 0 || o >= 1 {
		t.Errorf("a mitad de abrir: openness %v; se esperaba entre 0 y 1", o)
	}
	if next, _ := r.update(input{space: true}); next != Scene(r) || r.anim.phase != panelOpening {
		t.Error("mientras abre, Espacio cerró el reporte; no debería aceptar entrada")
	}
	for i := 0; r.anim.phase != panelOpen; i++ {
		if i > reportOpenTicks {
			t.Fatalf("el reporte no terminó de abrir en %d ticks", reportOpenTicks)
		}
		r.update(input{})
	}
	if r.anim.openness() != 1 {
		t.Errorf("abierto: openness %v; se esperaba 1", r.anim.openness())
	}
}

// En la demo (back nil) el reporte abre desde el centro de la pantalla,
// igual que la pausa cuando pausa la demo; en el modo juego, desde el
// botón REPORTE del HUD.
func TestReport_OpensFromTheRightSpot(t *testing.T) {
	demo := NewReportScene(game.Report{})
	wantDemo := vec{640, 360}
	if demo.anim.from != wantDemo {
		t.Errorf("en la demo abre desde %v; se esperaba el centro %v", demo.anim.from, wantDemo)
	}

	fromButton := reportButton.Min.Add(reportButton.Max).Div(2)
	inGame := newReportScene(game.Report{}, "", &plainScene{})
	wantGame := vec{float64(fromButton.X), float64(fromButton.Y)}
	if inGame.anim.from != wantGame {
		t.Errorf("en el juego abre desde %v; se esperaba el botón REPORTE %v", inGame.anim.from, wantGame)
	}
}
