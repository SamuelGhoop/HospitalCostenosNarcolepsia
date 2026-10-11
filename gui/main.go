// Interfaz gráfica del Hospital de los Costeños con Narcolepsia (bono +10 %).
//
// Es un módulo Go aparte (gui/go.mod): Ebitengine solo se importa aquí y
// el núcleo (hospital/, simulation/, game/) sigue siendo solo librería
// estándar. Se corre con:
//
//	cd gui && go run .
package main

import (
	"errors"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/ui"
)

func main() {
	app, err := ui.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// Ventana del tamaño de la resolución lógica: 1280×720.
	ebiten.SetWindowSize(ui.ScreenWidth, ui.ScreenHeight)
	ebiten.SetWindowTitle("Hospital de los Costeños con Narcolepsia")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// SALIR de la portada devuelve ebiten.Termination: es un cierre normal,
	// no un error.
	if err := ebiten.RunGame(app); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
}
