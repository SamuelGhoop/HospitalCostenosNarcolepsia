package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
	for _, load := range []func() error{loadFonts, loadPixelFont, loadSprites, loadStretcher, loadMap, loadTitle, loadPause} {
		if err := load(); err != nil {
			return nil, err
		}
	}
	boxes, err := loadHitboxes()
	if err != nil {
		return nil, err
	}
	titleHitboxes = boxes
	return &App{scene: NewTitleScene()}, nil
}

// Update lo llama Ebitengine 60 veces por segundo. La escena devuelve cuál
// sigue; así se cambia de pantalla sin un switch con todas las escenas.
func (a *App) Update() error { return a.update(inpututil.IsKeyJustPressed(ebiten.KeyEscape)) }

// update hace el trabajo de Update; esc dice si se acaba de apretar Esc
// (los tests lo pasan a mano). La pausa la abre la App, así las escenas no
// tienen que saber de Esc: sobre una escena que se puede pausar, Esc la
// tapa con el menú de pausa en vez de darle el tick. Con la pausa abierta,
// Esc lo atiende la pausa (ella no es pausable).
func (a *App) update(esc bool) error {
	if p, ok := a.scene.(pausable); ok && esc {
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

// Draw lo llama Ebitengine para pintar cada cuadro.
func (a *App) Draw(screen *ebiten.Image) {
	a.scene.Draw(screen)
}

// Layout fija la resolución lógica: 1280×720, la del mapa de fondo. Si la
// ventana es más grande (pantalla completa), Ebitengine la escala.
func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}
