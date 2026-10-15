// Interfaz gráfica del Hospital de los Costeños con Narcolepsia (bono +10 %).
//
// Es un módulo Go aparte (gui/go.mod): Ebitengine solo se importa aquí y
// el núcleo (hospital/, simulation/, game/) sigue siendo solo librería
// estándar. Se corre con:
//
//	cd gui && go run .
//
// El backdoor de pruebas (-agent, GAME_DESIGN §10.9) deja que el agente
// tester llegue a cualquier escena sin una persona jugando: arranca
// directo en -scene, con -seed fija, y sigue un guion de -input. Por
// ejemplo, para ver el modo juego a los 30 s y guardar una captura y un
// dump de estado:
//
//	cd gui && go run . -agent -scene game -seed 7 -input guion.txt -quit
//
// donde guion.txt tiene, una instrucción por línea:
//
//	wait 300
//	shot /ruta/captura.png
//	dump /ruta/estado.json
//	quit
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/ui"
)

func main() {
	agent := flag.Bool("agent", false, "modo de prueba: sin ventana de juego normal, arranca directo en -scene")
	scene := flag.String("scene", "title", "con -agent: escena de arranque (title, demo, game, report, pause)")
	step := flag.Int("step", 0, "con -agent -scene demo: paso al que salta (1 a 12); 0 = desde el principio")
	seed := flag.Int64("seed", 1, "con -agent: semilla del azar de la partida/demo y del tráfico")
	input := flag.String("input", "", "con -agent: guion de instrucciones (una por línea); ver gui/main.go")
	frames := flag.Int("frames", 0, "con -agent: ticks de Update sin entrada, después del guion")
	shot := flag.String("shot", "", "con -agent: captura (PNG, 1280×720, a 1×) al terminar")
	dump := flag.String("dump", "", "con -agent: dump de estado (JSON) al terminar")
	quit := flag.Bool("quit", false, "con -agent: termina solo al acabar (si no, deja la ventana abierta)")
	flag.Parse()

	if *agent {
		cfg := ui.AgentConfig{
			Scene: *scene, Step: *step, Seed: *seed, ScriptPath: *input,
			Frames: *frames, ShotPath: *shot, DumpPath: *dump, Quit: *quit,
		}
		if err := ui.RunAgent(cfg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	// El modo normal, igual que siempre: la portada, con ventana y mouse de
	// verdad.
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
