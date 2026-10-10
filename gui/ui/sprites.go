package ui

import (
	"fmt"
	"image"
	"image/png"
	"io/fs"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// patientSheetPath: la hoja del cuerpo base del paciente (GAME_DESIGN §10.3).
const patientSheetPath = "sprites/patient_body_skin1.png"

// patientSheet es la hoja ya cargada como imagen de Ebitengine.
var patientSheet *ebiten.Image

// animation es una fila de la hoja de sprites: una animación.
type animation int

// Las filas, en el orden de la hoja (y de la tabla de GAME_DESIGN §10.3).
const (
	animIdle       animation = iota // quieto respirando
	animWalkDown                    // caminar hacia abajo
	animWalkUp                      // caminar hacia arriba
	animWalkLeft                    // caminar a la izquierda (a la derecha: volteada)
	animDrowsy                      // aviso de sueño (cabeceo)
	animCollapse                    // desplomarse
	animSleepFloor                  // dormido en el piso
	animSleepBed                    // dormido en cama
	animWakeUp                      // despertarse
	animWave                        // alta (saluda)
)

// animSpec dice cuántos cuadros tiene una animación, cuántos ticks dura
// cada cuadro (60 ticks = 1 s) y si vuelve a empezar al terminar.
type animSpec struct {
	frames        int
	ticksPerFrame int
	loop          bool
}

// animSpecs sale de la tabla de GAME_DESIGN §10.3:
// 500 ms = 30 ticks · 150 ms = 9 · 250 ms = 15 · 120 ms = 7 · 600 ms = 36.
// Es un arreglo indexado por animation: animSpecs[animIdle] es la de quieto.
var animSpecs = [...]animSpec{
	animIdle:       {2, 30, true},
	animWalkDown:   {4, 9, true},
	animWalkUp:     {4, 9, true},
	animWalkLeft:   {4, 9, true},
	animDrowsy:     {4, 15, true},
	animCollapse:   {4, 7, false},
	animSleepFloor: {2, 36, true},
	animSleepBed:   {2, 36, true},
	animWakeUp:     {4, 9, false},
	animWave:       {4, 9, false}, // en la demo saluda una sola vez
}

// frameRect es el recorte de la hoja para el cuadro i de una animación:
// columna i, fila de la animación, de 16×24.
func frameRect(a animation, i int) image.Rectangle {
	x, y := i*cellWidth, int(a)*cellHeight
	return image.Rect(x, y, x+cellWidth, y+cellHeight)
}

// frameAt dice qué cuadro toca después de ticks ticks de animación. Si la
// animación no se repite, se queda en el último cuadro.
func frameAt(a animation, ticks int) int {
	spec := animSpecs[a]
	i := ticks / spec.ticksPerFrame
	if spec.loop {
		return i % spec.frames
	}
	if i >= spec.frames {
		return spec.frames - 1
	}
	return i
}

// finished dice si una animación que no se repite ya mostró todos sus cuadros.
func finished(a animation, ticks int) bool {
	spec := animSpecs[a]
	return !spec.loop && ticks >= spec.frames*spec.ticksPerFrame
}

// walkAnimation elige la animación de caminar según hacia dónde se movió
// más (dx, dy). Caminar a la derecha es caminar a la izquierda volteado.
func walkAnimation(dx, dy float64) (anim animation, flip bool) {
	if math.Abs(dx) > math.Abs(dy) {
		return animWalkLeft, dx > 0
	}
	if dy < 0 {
		return animWalkUp, false
	}
	return animWalkDown, false
}

// spritePlacement calcula la GeoM (la transformación) con la que se dibuja
// el cuadro de un paciente, y dice si va rotado.
//
//   - En la cama (dormido en cama, o el cuadro 1 de despertarse) el cuadro
//     horizontal se rota 90° en sentido horario, para que la cabeza quede
//     hacia la cabecera, y se pone sobre la cama.
//   - En cualquier otro caso se dibuja tal cual en at, sin rotar (en el
//     piso el cuerpo horizontal va así). Si flip, se voltea en su cuadro.
//
// Cuando lleguen las capas de camisa, pelo y sombrero, se dibujan con esta
// misma GeoM, así rotan y se voltean junto con el cuerpo.
func spritePlacement(a animation, frame int, bed image.Rectangle, hasBed bool, at vec, flip bool) (ebiten.GeoM, bool) {
	var g ebiten.GeoM
	inBedFrame := a == animSleepBed || (a == animWakeUp && frame == 0)
	if hasBed && inBedFrame {
		g.Rotate(math.Pi / 2)      // 90° horario: lo que estaba a la izquierda (la cabeza) queda arriba
		g.Translate(cellHeight, 0) // al rotar, el cuadro queda en x negativas: se devuelve (ahora mide 24×16)
		g.Translate(float64(bed.Min.X+(bed.Dx()-cellHeight)/2), float64(bed.Min.Y+4))
		return g, true
	}
	if flip {
		g.Scale(-1, 1)            // espejo horizontal…
		g.Translate(cellWidth, 0) // …y de vuelta a su cuadro
	}
	g.Translate(math.Round(at.x), math.Round(at.y))
	return g, false
}

// decodePNG lee un PNG de un sistema de archivos (por ejemplo, el embebido).
func decodePNG(fsys fs.FS, path string) (image.Image, error) {
	f, err := fsys.Open(path)
	if err != nil {
		return nil, fmt.Errorf("abriendo %s: %w", path, err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decodificando %s: %w", path, err)
	}
	return img, nil
}

// loadSprites carga las hojas embebidas como imágenes de Ebitengine.
func loadSprites() error {
	img, err := decodePNG(assets.Sprites, patientSheetPath)
	if err != nil {
		return err
	}
	patientSheet = ebiten.NewImageFromImage(img)
	return nil
}
