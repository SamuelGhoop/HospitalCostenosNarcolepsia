// Interfaz gráfica del Hospital de los Costeños con Narcolepsia (bono +10 %).
//
// Es un módulo Go aparte (gui/go.mod): Ebitengine solo se importa aquí y
// el núcleo (hospital/, simulation/, game/) sigue siendo solo librería
// estándar. Se corre con:
//
//	cd gui && go run .
package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/ui"
)

func main() {
	app, err := ui.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// Ventana al doble de la resolución lógica (640×360 → 1280×720).
	ebiten.SetWindowSize(ui.ScreenWidth*2, ui.ScreenHeight*2)
	ebiten.SetWindowTitle("Hospital de los Costeños con Narcolepsia — Demostración")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}
