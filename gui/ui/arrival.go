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

// doorReach: la puerta principal se abre cuando hay pies a menos de esto de
// su hueco (alguien entrando o saliendo).
const doorReach = 48

// La puerta principal se abre y se cierra suave (pedido de Samuel,
// 2026-10-11): las hojas se corren en doorSlideTicks y se cierra
// doorCloseDelay ticks (0,5 s) después de que no haya nadie cerca.
const (
	doorSlideTicks = 12
	doorCloseDelay = 30
)

// mainDoor es la puerta principal: lo abierta que está (de 0, cerrada, a 1)
// y cuántos ticks más se queda abierta aunque ya no haya nadie cerca.
type mainDoor struct {
	open float64
	hold int
}

// step avanza la puerta un tick. someoneNear dice si hay alguien cerca: en
// la demo lo mira la escena; en el modo juego lo dirá Snapshot.DoorOpen.
func (d *mainDoor) step(someoneNear bool) {
	if someoneNear {
		d.hold = doorCloseDelay
	} else if d.hold > 0 {
		d.hold--
	}
	if d.hold > 0 {
		d.open = math.Min(1, d.open+1.0/doorSlideTicks)
	} else {
		d.open = math.Max(0, d.open-1.0/doorSlideTicks)
	}
}

// someoneNearDoor dice si hay pies a menos de doorReach de la puerta principal.
func (s *DemoScene) someoneNearDoor() bool {
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

// drawMainDoor dibuja la puerta principal encima del fondo (que la trae
// cerrada), abierta en la fracción open: el umbral y las dos hojas del
// fondo corridas hacia el marco (se meten en la pared: lo que sale del
// hueco no se dibuja). Abierta del todo, de cada hoja se ve una sexta parte.
func drawMainDoor(dst *ebiten.Image, open float64) {
	if open <= 0 {
		return
	}
	g := doorways[len(doorways)-1].gap
	half := g.Dx() / 2
	shift := open * float64(half-g.Dx()/6)
	inGap := dst.SubImage(g).(*ebiten.Image) // lo que se dibuja fuera del hueco no se ve
	fillRect(inGap, g, colDoorFloor)
	for _, leaf := range []struct {
		r  image.Rectangle
		dx float64
	}{
		{image.Rect(g.Min.X, g.Min.Y, g.Min.X+half, g.Max.Y), -shift}, // la de la izquierda, hacia la izquierda
		{image.Rect(g.Min.X+half, g.Min.Y, g.Max.X, g.Max.Y), shift},
	} {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(leaf.r.Min.X)+math.Round(leaf.dx), float64(leaf.r.Min.Y))
		inGap.DrawImage(backgroundImg.SubImage(leaf.r).(*ebiten.Image), op)
	}
}
