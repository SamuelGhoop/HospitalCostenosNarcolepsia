package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Puntos de entrada reemplazables para el backdoor de pruebas (agent.go,
// GAME_DESIGN §10.9): en vez de que cada escena llame directo a inpututil o
// a ebiten, pasan por estas tres variables. En producción son la lectura
// real del teclado y el mouse; -agent las reemplaza por la entrada de un
// guion, así ninguna escena tiene que saber si la está jugando una persona
// o un guion.
var (
	// keyJustPressed es cómo demo_scene.go (C, A, S, Espacio) y app.go (Esc)
	// leen una tecla que se acaba de apretar.
	keyJustPressed = inpututil.IsKeyJustPressed

	// clickJustPressed es cómo demo_scene.go lee el clic izquierdo (la demo
	// no necesita saber dónde: solo si hubo clic).
	clickJustPressed = func() bool { return inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) }

	// readInputFn es cómo las escenas de portapapeles y el juego
	// (TitleScene, PauseScene, GameScene, ReportScene) leen el teclado y el
	// mouse de un tick, a través de input.go.
	readInputFn = defaultReadInput
)
