package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// App es el juego para Ebitengine: cumple la interfaz ebiten.Game
// (Update, Draw y Layout) y le delega todo a la escena actual.
type App struct {
	scene Scene
}

// Chequeo en compilación: si App dejara de cumplir ebiten.Game, no compila.
var _ ebiten.Game = (*App)(nil)

// NewApp carga las fuentes, los sprites, el mapa, la portada y la pausa, y
// arranca en la portada.
func NewApp() (*App, error) {
	if err := loadAssets(); err != nil {
		return nil, err
	}
	return &App{scene: NewTitleScene()}, nil
}

// loadAssets carga todo lo que hace falta antes de poner cualquier escena
// en pantalla (fuentes, sprites, mapa, portada y pausa). Aparte de NewApp,
// la usa el backdoor de pruebas (RunAgent, agent.go): arranca directo en
// la escena que pida -scene, no siempre en la portada.
func loadAssets() error {
	for _, load := range []func() error{loadFonts, loadPixelFont, loadSprites, loadStretcher, loadMap, loadTitle, loadPause} {
		if err := load(); err != nil {
			return err
		}
	}
	boxes, err := loadHitboxes()
	if err != nil {
		return err
	}
	titleHitboxes = boxes
	return nil
}

// Update lo llama Ebitengine 60 veces por segundo. La escena devuelve cuál
// sigue; así se cambia de pantalla sin un switch con todas las escenas.
//
// El Esc se lee con keyJustPressed (agent_hooks.go) en vez de llamar directo
// a inpututil: así el backdoor de pruebas (-agent) lo puede reemplazar por
// un guion sin ventana real.
func (a *App) Update() error { return a.update(keyJustPressed(ebiten.KeyEscape)) }

// update hace el trabajo de Update; esc dice si se acaba de apretar Esc
// (los tests lo pasan a mano). La pausa la abre la App, así las escenas no
// tienen que saber de Esc: sobre una escena que se puede pausar, Esc la
// tapa con el menú de pausa en vez de darle el tick. Con la pausa abierta,
// Esc lo atiende la pausa (ella no es pausable).
func (a *App) update(esc bool) error {
	if p, ok := a.scene.(pausable); ok && (esc || blurred(a.scene)) {
		a.scene = newPauseScene(p)
		return nil
	}
	next, err := a.scene.Update()
	if err != nil {
		return err // Ebitengine se detiene y RunGame devuelve este error
	}
	a.scene = next
	return nil
}

// isFocused dice si la ventana tiene el foco (los tests la cambian).
var isFocused = ebiten.IsFocused

// blurred dice si la escena se tiene que pausar porque la ventana perdió el
// foco: solo el modo juego (pausesOnBlur), para que el turno no corra
// mientras el jugador no está mirando.
func blurred(s Scene) bool {
	b, ok := s.(interface{ pausesOnBlur() bool })
	return ok && b.pausesOnBlur() && !isFocused()
}

// Draw lo llama Ebitengine para pintar cada cuadro.
func (a *App) Draw(screen *ebiten.Image) {
	a.scene.Draw(screen)
}

// Layout fija la resolución lógica: 1280×720, la del mapa de fondo. Si la
// ventana es más grande (pantalla completa), Ebitengine la escala.
func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}
