package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// input es lo que el jugador hizo en este tick, ya leído del teclado y del
// mouse. Las escenas de menú trabajan con este struct y no con Ebitengine
// directo: así los tests pueden "apretar teclas" armando un input.
type input struct {
	up, down    bool // ↑ o W, ↓ o S
	left, right bool // ← → (la bolsa de empleo: cambiar de hoja)
	confirm     bool // Enter
	space       bool // Espacio
	back        bool // Esc
	click       bool // clic izquierdo
	rightClick  bool // clic derecho
	shift       bool // Shift apretado (en el juego, muestra todas las etiquetas)
	mouse       image.Point
	mouseMoved  bool // el mouse se movió desde el tick anterior
}

// lastMouse es dónde estaba el mouse en el tick anterior.
var lastMouse image.Point

// readInput lee el teclado y el mouse de este tick: delega en readInputFn
// (agent_hooks.go), que en producción es defaultReadInput y que el backdoor
// de pruebas (-agent) reemplaza por la entrada de un guion.
func readInput() input { return readInputFn() }

// defaultReadInput es la lectura real de Ebitengine.
func defaultReadInput() input {
	key := inpututil.IsKeyJustPressed
	x, y := ebiten.CursorPosition()
	mouse := image.Pt(x, y)
	in := input{
		up:         key(ebiten.KeyArrowUp) || key(ebiten.KeyW),
		down:       key(ebiten.KeyArrowDown) || key(ebiten.KeyS),
		left:       key(ebiten.KeyArrowLeft),
		right:      key(ebiten.KeyArrowRight),
		confirm:    key(ebiten.KeyEnter) || key(ebiten.KeyNumpadEnter),
		space:      key(ebiten.KeySpace),
		back:       key(ebiten.KeyEscape),
		click:      inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
		rightClick: inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight),
		shift:      ebiten.IsKeyPressed(ebiten.KeyShift),
		mouse:      mouse,
		mouseMoved: mouse != lastMouse,
	}
	lastMouse = mouse
	return in
}
