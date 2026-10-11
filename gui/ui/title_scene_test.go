package ui

import (
	"errors"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// choose baja hasta la opción o en la portada, la confirma con Enter y
// avanza hasta que termina la espera del cursor. Devuelve lo que devolvió
// la portada en ese último tick.
func choose(t *testing.T, s *TitleScene, o option) (Scene, error) {
	t.Helper()
	for i := 0; s.menu.selected != o; i++ {
		if i > len(menuOptions) {
			t.Fatalf("no se pudo llegar a %v con ↓", o)
		}
		if _, err := s.update(input{down: true}); err != nil {
			t.Fatal(err)
		}
	}
	next, err := s.update(input{confirm: true})
	for i := 0; i < confirmTicks && next == Scene(s) && err == nil; i++ {
		next, err = s.update(input{})
	}
	return next, err
}

// NUEVA PARTIDA (hasta F3) y AJUSTES (hasta F6) muestran "Próximamente" un
// rato y la portada sigue ahí.
func TestTitle_NewGameAndSettingsSayComingSoon(t *testing.T) {
	for _, o := range []option{optNew, optSettings} {
		s := NewTitleScene()
		next, err := choose(t, s, o)
		if err != nil || next != Scene(s) {
			t.Fatalf("%v: devolvió (%T, %v); se esperaba seguir en la portada", o, next, err)
		}
		if s.notice <= 0 {
			t.Errorf("%v: no salió el aviso Próximamente", o)
		}
		for i := 0; i < comingSoonTicks; i++ {
			s.update(input{})
		}
		if s.notice > 0 {
			t.Errorf("%v: el aviso sigue después de %d ticks", o, comingSoonTicks)
		}
	}
}

// DEMO SECCIÓN 6 abre la demo que ya existe; CRÉDITOS, la pantalla de
// créditos; SALIR cierra el programa limpio (ebiten.Termination).
func TestTitle_DemoCreditsAndExit(t *testing.T) {
	if next, _ := choose(t, NewTitleScene(), optDemo); reflect.TypeOf(next) != reflect.TypeOf(&DemoScene{}) {
		t.Errorf("DEMO SECCIÓN 6 llevó a %T; se esperaba *DemoScene", next)
	}
	if next, _ := choose(t, NewTitleScene(), optCredits); reflect.TypeOf(next) != reflect.TypeOf(&CreditsScene{}) {
		t.Errorf("CRÉDITOS llevó a %T; se esperaba *CreditsScene", next)
	}
	if _, err := choose(t, NewTitleScene(), optExit); !errors.Is(err, ebiten.Termination) {
		t.Errorf("SALIR devolvió %v; se esperaba ebiten.Termination (cierre limpio)", err)
	}
}

// Los créditos son exactamente los que pidió Samuel, y Enter o Esc vuelven
// a la portada.
func TestCredits_LinesAndGoingBack(t *testing.T) {
	want := []string{
		"Hospital de los Costeños con Narcolepsia",
		"Samuel Giraldo",
		"Código con ayuda de Claude",
		"Fuente Press Start 2P (OFL)",
		"Motor: Ebitengine",
	}
	if !reflect.DeepEqual(creditsLines, want) {
		t.Errorf("créditos = %q; se esperaba %q", creditsLines, want)
	}

	c := NewCreditsScene()
	if next, _ := c.update(input{}); next != Scene(c) {
		t.Errorf("sin tocar nada los créditos llevaron a %T", next)
	}
	for _, in := range []input{{confirm: true}, {back: true}} {
		if next, _ := NewCreditsScene().update(in); reflect.TypeOf(next) != reflect.TypeOf(&TitleScene{}) {
			t.Errorf("con %+v los créditos llevaron a %T; se esperaba la portada", in, next)
		}
	}
}

// Al terminar la demo, el Shift Report vuelve a la portada con Espacio o clic.
func TestReport_GoesBackToTheTitle(t *testing.T) {
	d := game.NewDemo()
	if err := d.Skip(); err != nil {
		t.Fatal(err)
	}
	r := NewReportScene(d.Report())
	if next, _ := r.update(input{}); next != Scene(r) {
		t.Errorf("sin tocar nada el reporte llevó a %T", next)
	}
	for _, in := range []input{{space: true}, {click: true}} {
		if next, _ := NewReportScene(d.Report()).update(in); reflect.TypeOf(next) != reflect.TypeOf(&TitleScene{}) {
			t.Errorf("con %+v el reporte llevó a %T; se esperaba la portada", in, next)
		}
	}
}
