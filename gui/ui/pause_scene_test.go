package ui

import (
	"image"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// La pausa arranca en REANUDAR y ↑/↓ recorren sus 3 opciones dando la
// vuelta.
func TestPauseMenu_StartsOnResumeAndWraps(t *testing.T) {
	m := newPauseMenu()
	if m.selected != optResume {
		t.Fatalf("arranca en %v; se esperaba REANUDAR", m.selected)
	}
	for _, want := range []option{optSettings, optToTitle, optResume} {
		m.update(press(false, true, false), nil)
		if m.selected != want {
			t.Errorf("↓: quedó en %v; se esperaba %v", m.selected, want)
		}
	}
	m.update(press(true, false, false), nil)
	if m.selected != optToTitle {
		t.Errorf("↑ desde REANUDAR: quedó en %v; se esperaba SALIR AL MENÚ (da la vuelta)", m.selected)
	}
}

// Las imágenes son pause/pause_<opción>_<cuadro>.png, y el cursor alterna
// cada 0,35 s como en la portada.
func TestPauseMenu_Images(t *testing.T) {
	m := newPauseMenu()
	if got, want := m.image(), "pause/pause_reanudar_0.png"; got != want {
		t.Errorf("imagen = %q; se esperaba %q", got, want)
	}
	for i := 0; i < cursorTicks; i++ {
		m.update(input{}, nil)
	}
	if got, want := m.image(), "pause/pause_reanudar_1.png"; got != want {
		t.Errorf("a los %d ticks: %q; se esperaba %q", cursorTicks, got, want)
	}
	m.update(press(false, true, false), nil)
	if got, want := m.image(), "pause/pause_ajustes_1.png"; got != want {
		t.Errorf("con ↓: %q; se esperaba %q", got, want)
	}
	if got, want := pausePath(optToTitle, 0), "pause/pause_salir_0.png"; got != want {
		t.Errorf("SALIR AL MENÚ: %q; se esperaba %q", got, want)
	}
}

// hitboxes.json de la pausa trae rectángulos [x, y, ancho, alto] sin giro:
// cada renglón cubre exactamente los píxeles de image.Rect(x, y, x+ancho,
// y+alto), ni uno más ni uno menos.
func TestPauseHitboxes_RectsFromTheJSON(t *testing.T) {
	boxes, err := loadPauseHitboxes()
	if err != nil {
		t.Fatal(err)
	}
	want := map[option]image.Rectangle{
		optResume:   image.Rect(490, 344, 790, 381),
		optSettings: image.Rect(490, 385, 790, 422),
		optToTitle:  image.Rect(490, 427, 790, 464),
	}
	if len(boxes) != len(want) {
		t.Fatalf("hay %d renglones; se esperaban %d", len(boxes), len(want))
	}
	for o, r := range want {
		last := r.Max.Sub(image.Pt(1, 1))
		center := r.Min.Add(r.Max).Div(2)
		for _, p := range []image.Point{r.Min, {last.X, r.Min.Y}, {r.Min.X, last.Y}, last, center} {
			if got, ok := boxes.at(p); !ok || got != o {
				t.Errorf("%v está adentro de %v y cayó en (%v, %v)", p, o, got, ok)
			}
		}
		for _, p := range []image.Point{{r.Min.X - 1, r.Min.Y}, {r.Max.X, r.Min.Y}, {r.Min.X, r.Min.Y - 1}, {r.Min.X, r.Max.Y}} {
			if got, ok := boxes.at(p); ok && got == o {
				t.Errorf("%v está afuera de %v y cayó en él", p, o)
			}
		}
	}
	if _, ok := boxes.at(image.Pt(10, 10)); ok {
		t.Error("un punto lejos del portapapeles cayó en un renglón")
	}
}

// Los 6 portapapeles de la pausa vienen embebidos y miden 407×311.
func TestPauseAssets_SixClipboardsOf407x311(t *testing.T) {
	for _, o := range pauseOptions {
		for frame := 0; frame < 2; frame++ {
			img, err := decodePNG(assets.Pause, pausePath(o, frame))
			if err != nil {
				t.Fatal(err)
			}
			if b := img.Bounds(); b.Dx() != 407 || b.Dy() != 311 {
				t.Errorf("%s mide %d×%d; se esperaba 407×311", pausePath(o, frame), b.Dx(), b.Dy())
			}
		}
	}
}

// fakeGame es una escena que se puede pausar y cuenta lo que le pasa: así
// se prueba la pausa antes de que exista el modo juego (F3).
type fakeGame struct{ updates, paused, resumed, quit int }

func (f *fakeGame) Update() (Scene, error) { f.updates++; return f, nil }
func (f *fakeGame) Draw(*ebiten.Image)     {}
func (f *fakeGame) onPause()               { f.paused++ }
func (f *fakeGame) onResume()              { f.resumed++ }
func (f *fakeGame) onQuit()                { f.quit++ }

// plainScene no se puede pausar, como la portada o los créditos.
type plainScene struct{ updates int }

func (p *plainScene) Update() (Scene, error) { p.updates++; return p, nil }
func (p *plainScene) Draw(*ebiten.Image)     {}

// choosePause baja hasta la opción o en la pausa, la confirma con Enter y
// avanza hasta que termina la espera del cursor. Devuelve lo que devolvió
// la pausa en ese último tick.
func choosePause(t *testing.T, p *PauseScene, o option) (Scene, error) {
	t.Helper()
	for i := 0; p.menu.selected != o; i++ {
		if i > len(pauseOptions) {
			t.Fatalf("no se pudo llegar a %v con ↓", o)
		}
		if _, err := p.update(input{down: true}); err != nil {
			t.Fatal(err)
		}
	}
	next, err := p.update(input{confirm: true})
	for i := 0; i < confirmTicks && next == Scene(p) && err == nil; i++ {
		next, err = p.update(input{})
	}
	return next, err
}

// Esc abre la pausa sobre una escena que se puede pausar (la demo; el juego
// en F3) y le avisa una vez, sin darle un Update. Sobre la portada, los
// créditos y el reporte no abre nada: esas escenas siguen recibiendo el
// tick (en los créditos, Esc vuelve a la portada).
func TestApp_EscPausesOnlyPausableScenes(t *testing.T) {
	g := &fakeGame{}
	a := &App{scene: g}
	if err := a.update(true); err != nil {
		t.Fatal(err)
	}
	p, ok := a.scene.(*PauseScene)
	if !ok || p.under != pausable(g) {
		t.Fatalf("con Esc la escena quedó en %T; se esperaba la pausa encima del juego", a.scene)
	}
	if g.paused != 1 || g.updates != 0 {
		t.Errorf("al abrir la pausa: onPause %d veces y %d Update; se esperaba 1 y 0", g.paused, g.updates)
	}
	if p.menu.ticks != 0 { // con el mismo Esc todavía apretado, se cerraría de una
		t.Errorf("la pausa recibió %d ticks en el tick del Esc que la abrió; se esperaban 0", p.menu.ticks)
	}

	plain := &plainScene{}
	a = &App{scene: plain}
	if err := a.update(true); err != nil {
		t.Fatal(err)
	}
	if a.scene != Scene(plain) || plain.updates != 1 {
		t.Errorf("Esc sobre una escena que no se pausa: quedó %T con %d Update; se esperaba la misma con 1", a.scene, plain.updates)
	}

	for _, s := range []Scene{NewTitleScene(), NewCreditsScene(), NewReportScene(game.Report{})} {
		if _, ok := s.(pausable); ok {
			t.Errorf("%T se puede pausar; no debía", s)
		}
	}
	if _, ok := Scene(NewDemoScene(game.NewDemo())).(pausable); !ok {
		t.Error("la demo no se puede pausar")
	}
}

// Mientras la pausa está abierta, la escena de abajo no recibe ni un
// Update: su reloj, sus animaciones y su avance automático se quedan
// quietos.
func TestApp_PausedSceneIsNotUpdated(t *testing.T) {
	g := &fakeGame{}
	a := &App{scene: g}
	a.update(true)
	for i := 0; i < 600; i++ { // 10 s
		if err := a.update(false); err != nil {
			t.Fatal(err)
		}
	}
	if g.updates != 0 {
		t.Errorf("en pausa el juego recibió %d Update; se esperaban 0", g.updates)
	}
	if _, ok := a.scene.(*PauseScene); !ok {
		t.Errorf("la pausa se cerró sola: la escena es %T", a.scene)
	}
}

// Con la pausa abierta, Esc la cierra al instante (sin esperar al cursor)
// y vuelve a la MISMA escena, no a una nueva.
func TestPause_EscResumesTheSameSceneAtOnce(t *testing.T) {
	g := &fakeGame{}
	p := newPauseScene(g)
	next, err := p.update(input{back: true})
	if err != nil || next != Scene(g) {
		t.Fatalf("Esc devolvió (%T, %v); se esperaba el mismo juego", next, err)
	}
	if g.resumed != 1 {
		t.Errorf("onResume se llamó %d veces; se esperaba 1", g.resumed)
	}
}

// REANUDAR con Enter espera los 0,45 s del cursor y vuelve a la misma
// escena.
func TestPause_ResumeWaitsForTheCursor(t *testing.T) {
	g := &fakeGame{}
	p := newPauseScene(g)
	if next, _ := p.update(input{confirm: true}); next != Scene(p) {
		t.Fatal("REANUDAR salió de una; debía esperar al cursor")
	}
	for i := 1; i < confirmTicks; i++ {
		if next, _ := p.update(input{}); next != Scene(p) {
			t.Fatalf("REANUDAR salió a los %d ticks; debía esperar %d", i, confirmTicks)
		}
	}
	if next, _ := p.update(input{}); next != Scene(g) || g.resumed != 1 {
		t.Errorf("a los %d ticks: %T y onResume %d veces; se esperaba el juego y 1", confirmTicks, next, g.resumed)
	}
}

// Mientras se confirma una opción el menú ya no escucha: Esc no cierra la
// pausa a mitad de SALIR AL MENÚ.
func TestPause_EscIsIgnoredWhileConfirming(t *testing.T) {
	g := &fakeGame{}
	p := newPauseScene(g)
	p.update(input{down: true})
	p.update(input{down: true}) // SALIR AL MENÚ
	p.update(input{confirm: true})
	if next, _ := p.update(input{back: true}); next != Scene(p) || g.resumed != 0 {
		t.Errorf("Esc confirmando SALIR AL MENÚ: %T y onResume %d veces; se esperaba seguir en la pausa", next, g.resumed)
	}
}

// AJUSTES (hasta F6) muestra "Próximamente" 1,5 s y la pausa sigue abierta.
func TestPause_SettingsSaysComingSoon(t *testing.T) {
	g := &fakeGame{}
	p := newPauseScene(g)
	next, err := choosePause(t, p, optSettings)
	if err != nil || next != Scene(p) {
		t.Fatalf("AJUSTES devolvió (%T, %v); se esperaba seguir en la pausa", next, err)
	}
	if p.notice <= 0 {
		t.Error("no salió el aviso Próximamente")
	}
	if g.resumed != 0 || g.quit != 0 {
		t.Errorf("AJUSTES llamó onResume %d y onQuit %d veces; se esperaba 0 y 0", g.resumed, g.quit)
	}
	for i := 0; i < comingSoonTicks; i++ {
		p.update(input{})
	}
	if p.notice > 0 {
		t.Errorf("el aviso sigue después de %d ticks", comingSoonTicks)
	}
}

// SALIR AL MENÚ le avisa a la escena (en F3 el juego cancela su context y
// se pierde el día en curso) y vuelve a la portada.
func TestPause_QuitGoesToTheTitle(t *testing.T) {
	g := &fakeGame{}
	next, err := choosePause(t, newPauseScene(g), optToTitle)
	if err != nil || reflect.TypeOf(next) != reflect.TypeOf(&TitleScene{}) {
		t.Fatalf("SALIR AL MENÚ devolvió (%T, %v); se esperaba la portada", next, err)
	}
	if g.quit != 1 || g.resumed != 0 {
		t.Errorf("onQuit %d y onResume %d veces; se esperaba 1 y 0", g.quit, g.resumed)
	}
}

// La demo con el avance automático (3 s) prendido: 10 s de pausa no la
// mueven, y al reanudar sigue contando desde donde iba, no desde cero.
func TestPause_DemoAutoAdvanceFreezesAndContinues(t *testing.T) {
	if err := loadFonts(); err != nil { // los puestos dependen de los rótulos
		t.Fatal(err)
	}
	demo := NewDemoScene(game.NewDemo())
	demo.auto.toggle()
	a := &App{scene: demo}
	const before = 100
	for i := 0; i < before; i++ {
		a.update(false)
	}
	a.update(true)
	for i := 0; i < 600; i++ {
		a.update(false)
	}
	if demo.snap.Step != 0 {
		t.Fatalf("en pausa la demo avanzó al paso %d", demo.snap.Step)
	}
	p, ok := a.scene.(*PauseScene)
	if !ok {
		t.Fatalf("la escena es %T; se esperaba la pausa", a.scene)
	}
	next, _ := p.update(input{back: true})
	a.scene = next
	for i := before; i < autoAdvanceTicks-1; i++ {
		a.update(false)
	}
	if demo.snap.Step != 0 {
		t.Fatalf("la demo avanzó antes de completar los 3 s (paso %d)", demo.snap.Step)
	}
	a.update(false)
	if demo.snap.Step != 1 {
		t.Errorf("a los 3 s, contando los de antes de la pausa, la demo está en el paso %d; se esperaba el 1", demo.snap.Step)
	}
}
