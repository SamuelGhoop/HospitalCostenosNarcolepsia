package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// App es el juego para Ebitengine: cumple la interfaz ebiten.Game
// (Update, Draw y Layout) y le delega todo a la escena actual.
type App struct {
	scene Scene
}

// Chequeo en compilación: si App dejara de cumplir ebiten.Game, no compila.
var _ ebiten.Game = (*App)(nil)

// NewApp carga las fuentes y arranca en la demostración de la Sección 6.
func NewApp() (*App, error) {
	if err := loadFonts(); err != nil {
		return nil, err
	}
	return &App{scene: NewDemoScene(game.NewDemo())}, nil
}

// Update lo llama Ebitengine 60 veces por segundo. La escena devuelve cuál
// sigue; así se cambia de pantalla sin un switch con todas las escenas.
func (a *App) Update() error {
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

// Layout fija la resolución lógica. Ebitengine la escala a la ventana; con
// escala entera (×2) usa el filtro nearest, así el pixel art se ve nítido.
func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}
