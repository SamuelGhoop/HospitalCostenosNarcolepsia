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
	up, down   bool // ↑ o W, ↓ o S
	confirm    bool // Enter
	space      bool // Espacio
	back       bool // Esc
	click      bool // clic izquierdo
	mouse      image.Point
	mouseMoved bool // el mouse se movió desde el tick anterior
}

// lastMouse es dónde estaba el mouse en el tick anterior.
var lastMouse image.Point

// readInput lee el teclado y el mouse de este tick.
func readInput() input {
	key := inpututil.IsKeyJustPressed
	x, y := ebiten.CursorPosition()
	mouse := image.Pt(x, y)
	in := input{
		up:         key(ebiten.KeyArrowUp) || key(ebiten.KeyW),
		down:       key(ebiten.KeyArrowDown) || key(ebiten.KeyS),
		confirm:    key(ebiten.KeyEnter) || key(ebiten.KeyNumpadEnter),
		space:      key(ebiten.KeySpace),
		back:       key(ebiten.KeyEscape),
		click:      inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
		mouse:      mouse,
		mouseMoved: mouse != lastMouse,
	}
	lastMouse = mouse
	return in
}
