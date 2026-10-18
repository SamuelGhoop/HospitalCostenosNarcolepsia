package ui

import (
	"image"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// Los paneles del fin de día y del Game Over abren y cierran animados,
// como la bolsa de empleo (anim.go, pedido de Samuel el 2026-10-17): no
// aceptan clics mientras animan, y la acción (StartNextDay, NUEVA PARTIDA,
// MENÚ) pasa al terminar de cerrar, no al hacer clic.
//
// Estos tests llaman updateDayOver/updateGameOver DIRECTO (no s.update,
// que en su primera línea refresca s.snap con s.g.Snapshot() de la
// partida real, y la pisaría): es la animación del panel lo que se prueba
// acá, no las reglas de cuándo termina el turno o la partida (esas ya
// tienen sus pruebas en game/ y en game_scene_test.go).

// clickCenter es el centro de r, para hacerle clic a un botón.
func clickCenter(r image.Rectangle) image.Point { return r.Min.Add(r.Max).Div(2) }

// TestGameScene_DayOverOpensAndClosesAnimated comprueba la animación del
// panel del fin de día: abre desde 0, no acepta clics hasta llegar a 1, y
// StartNextDay solo se llama (acá, con el reloj real todavía sin llegar a
// las 20:00, así que da ErrDayNotOver y queda como aviso de la escena) al
// terminar de cerrar, no al hacer clic en SIGUIENTE DÍA.
func TestGameScene_DayOverOpensAndClosesAnimated(t *testing.T) {
	s := newTestGameScene(t)
	s.snap.DayOver = true

	if _, err := s.updateDayOver(input{}); err != nil {
		t.Fatal(err)
	}
	if s.dayOverAnim.phase != panelOpening || s.dayOverAnim.openness() != 0 {
		t.Fatalf("recién visto DayOver: fase %v, openness %v; se esperaba abriendo en 0", s.dayOverAnim.phase, s.dayOverAnim.openness())
	}

	// Un clic en SIGUIENTE DÍA mientras abre no hace nada.
	if _, err := s.updateDayOver(input{click: true, mouse: clickCenter(dayOverButton)}); err != nil {
		t.Fatal(err)
	}
	if s.dayOverAnim.phase != panelOpening || s.messageTicks != 0 {
		t.Error("un clic en SIGUIENTE DÍA mientras el panel abre no debía hacer nada")
	}

	for i := 0; i < panelOpenTicks/2; i++ {
		s.updateDayOver(input{})
	}
	if o := s.dayOverAnim.openness(); o <= 0 || o >= 1 {
		t.Errorf("a mitad de abrir: openness %v; se esperaba entre 0 y 1", o)
	}
	for s.dayOverAnim.phase != panelOpen {
		s.updateDayOver(input{})
	}
	if o := s.dayOverAnim.openness(); o != 1 {
		t.Errorf("abierto: openness %v; se esperaba 1", o)
	}

	if _, err := s.updateDayOver(input{click: true, mouse: clickCenter(dayOverButton)}); err != nil {
		t.Fatal(err)
	}
	if s.dayOverAnim.phase != panelClosing {
		t.Fatal("SIGUIENTE DÍA no arrancó el cierre")
	}
	for i := 0; i < panelCloseTicks-1; i++ {
		if _, err := s.updateDayOver(input{}); err != nil {
			t.Fatal(err)
		}
		if s.messageTicks != 0 {
			t.Fatalf("tick %d de %d: ya llamó StartNextDay (dejó aviso); debía esperar a terminar de cerrar", i, panelCloseTicks)
		}
	}
	if _, err := s.updateDayOver(input{}); err != nil {
		t.Fatal(err)
	}
	// El reloj real de la partida de prueba no llegó a las 20:00: StartNextDay
	// da ErrDayNotOver, que queda como aviso de la escena (say()). Lo que
	// importa acá es que pasó justo en este tick, no antes.
	if s.messageTicks == 0 {
		t.Error("al terminar de cerrar debía intentar StartNextDay (y dejar su error como aviso)")
	}
	if s.dayOverAnim != nil {
		t.Error("tras terminar de cerrar, dayOverAnim debía volver a nil (para abrir de cero el próximo fin de día)")
	}
}

// TestGameScene_GameOverOpensAnimatedAndClosesToChosenAction comprueba
// que NUEVA PARTIDA solo llama a newGameSceneFn al terminar de cerrar, no
// al hacer clic.
func TestGameScene_GameOverOpensAnimatedAndClosesToChosenAction(t *testing.T) {
	saveAgentHooks(t)
	called := 0
	newGameSceneFn = func() (*GameScene, error) { called++; return nil, nil }

	s := newTestGameScene(t)
	s.snap.GameOver = game.LostLicense

	if _, err := s.updateGameOver(input{}); err != nil {
		t.Fatal(err)
	}
	if s.gameOverAnim.phase != panelOpening || s.gameOverAnim.openness() != 0 {
		t.Fatalf("recién visto GameOver: fase %v, openness %v; se esperaba abriendo en 0", s.gameOverAnim.phase, s.gameOverAnim.openness())
	}

	center := clickCenter(newGameButton)
	if _, err := s.updateGameOver(input{click: true, mouse: center}); err != nil {
		t.Fatal(err)
	}
	if s.gameOverAnim.phase != panelOpening || called != 0 {
		t.Error("un clic en NUEVA PARTIDA mientras el panel abre no debía hacer nada")
	}

	for i := 0; i < panelOpenTicks/2; i++ {
		s.updateGameOver(input{})
	}
	if o := s.gameOverAnim.openness(); o <= 0 || o >= 1 {
		t.Errorf("a mitad de abrir: openness %v; se esperaba entre 0 y 1", o)
	}
	for s.gameOverAnim.phase != panelOpen {
		s.updateGameOver(input{})
	}

	if _, err := s.updateGameOver(input{click: true, mouse: center}); err != nil {
		t.Fatal(err)
	}
	if s.gameOverAnim.phase != panelClosing || called != 0 {
		t.Fatal("NUEVA PARTIDA no arrancó el cierre, o ya llamó newGameSceneFn antes de terminar")
	}
	for i := 0; i < panelCloseTicks-1; i++ {
		if _, err := s.updateGameOver(input{}); err != nil {
			t.Fatal(err)
		}
		if called != 0 {
			t.Fatalf("tick %d de %d: ya llamó newGameSceneFn; debía esperar a terminar de cerrar", i, panelCloseTicks)
		}
	}
	if _, err := s.updateGameOver(input{}); err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Errorf("newGameSceneFn se llamó %d veces; se esperaba 1, justo al terminar de cerrar", called)
	}
}
