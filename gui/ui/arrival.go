package ui

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// La llegada de los pacientes (paso 3 de la demo; pedido de Samuel,
// 2026-10-11): aparecen fuera de la pantalla, a la derecha, y entran
// caminando por la acera y por la puerta principal, uno cada 1,5 s.

// arrivalGap: cada paciente nuevo sale 1,5 s después del anterior.
const arrivalGap = 90

// arrivalChoreography: cada uno espera fuera de la pantalla (spawn) hasta
// que le toca y camina a su puesto. Termina cuando llega el último.
func arrivalChoreography(ids []string, targets map[string]vec, spawn vec) *choreography {
	ph := phase{moves: map[string]vec{}, delays: map[string]int{}, waitAt: map[string]vec{}}
	for i, id := range ids {
		ph.moves[id], ph.delays[id], ph.waitAt[id] = targets[id], i*arrivalGap, spawn
	}
	return &choreography{phases: []phase{ph}}
}

// doorReach: la puerta principal está abierta mientras haya pies a menos de
// esto de su hueco (alguien entrando o saliendo); si no, se cierra sola.
const doorReach = 48

// doorOpen dice si la puerta principal está abierta. En el modo juego lo
// dirá Snapshot.DoorOpen (F3b).
func (s *DemoScene) doorOpen() bool {
	door := doorways[len(doorways)-1].gap
	for _, f := range s.figures {
		if distToRect(s.pos[f.id].add(feetOffset), door) <= doorReach {
			return true
		}
	}
	return false
}

// distToRect es la distancia de p al rectángulo r (0 si está adentro).
func distToRect(p vec, r image.Rectangle) float64 {
	dx := math.Max(math.Max(float64(r.Min.X)-p.x, 0), p.x-float64(r.Max.X))
	dy := math.Max(math.Max(float64(r.Min.Y)-p.y, 0), p.y-float64(r.Max.Y))
	return math.Hypot(dx, dy)
}

// Colores de la puerta abierta (los de la maqueta: el umbral de las puertas
// y la hoja de la principal).
var (
	colDoorFloor = color.RGBA{0xd7, 0xdc, 0xe2, 0xff}
	colDoorLeaf  = color.RGBA{0x5a, 0x61, 0x6b, 0xff}
)

// drawMainDoor dibuja la puerta principal abierta encima del fondo (que la
// trae cerrada): el umbral despejado y las dos hojas corridas hacia el
// marco.
func drawMainDoor(dst *ebiten.Image, open bool) {
	if !open {
		return
	}
	g := doorways[len(doorways)-1].gap
	fillRect(dst, g, colDoorFloor)
	w := g.Dx() / 6
	for _, x := range []int{g.Min.X, g.Max.X - w} {
		leaf := image.Rect(x, g.Min.Y, x+w, g.Max.Y)
		fillRect(dst, leaf, colInk)
		fillRect(dst, leaf.Inset(1), colDoorLeaf)
	}
}
