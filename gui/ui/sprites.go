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
	animAttend                      // atender (solo el médico: fila 5 de su hoja, ver staffRect)
	animPush                        // empujar la camilla de lado (solo el camillero: fila 5 de SU hoja)
	animPushDown                    // empujar la camilla hacia abajo (fila 6 de SU hoja, 2026-10-17)
	animPushUp                      // empujar la camilla hacia arriba (fila 7 de SU hoja, 2026-10-17)
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
// Caminar va a 8 ticks por cuadro (unos 133 ms), acorde con la velocidad.
// Es un arreglo indexado por animation: animSpecs[animIdle] es la de quieto.
var animSpecs = [...]animSpec{
	animIdle:       {2, 30, true},
	animWalkDown:   {4, 8, true}, // unos 8 cuadros por segundo, acorde con el paso
	animWalkUp:     {4, 8, true},
	animWalkLeft:   {4, 8, true},
	animDrowsy:     {4, 15, true},
	animCollapse:   {4, 7, false},
	animSleepFloor: {2, 36, true},
	animSleepBed:   {2, 36, true},
	animWakeUp:     {4, 9, false},
	animWave:       {4, 9, false}, // en la demo saluda una sola vez
	animAttend:     {4, 9, false}, // atiende una vez al llegar junto al paciente
	animPush:       {4, 8, true},  // empuja la camilla al mismo ritmo que camina
	animPushDown:   {4, 8, true},
	animPushUp:     {4, 8, true},
}

// attendTicks: lo que dura atender (4 cuadros × 9 ticks).
const attendTicks = 36

// frameRect es el recorte de la hoja para el cuadro i de una animación:
// columna i, fila de la animación, de 16×24.
func frameRect(a animation, i int) image.Rectangle {
	x, y := i*cellWidth, int(a)*cellHeight
	return image.Rect(x, y, x+cellWidth, y+cellHeight)
}

// staffRect es el recorte en una hoja del personal: las 4 primeras filas
// son las mismas del paciente (quieto y caminar) y de la 5 en adelante son
// propias de cada hoja: atender en la del médico (fila 5), y en la del
// camillero empujar de lado (fila 5), hacia abajo (fila 6) y hacia arriba
// (fila 7; GAME_DESIGN §10.3, 2026-10-17).
func staffRect(a animation, i int) image.Rectangle {
	row := int(a)
	switch a {
	case animAttend, animPush:
		row = 4
	case animPushDown:
		row = 5
	case animPushUp:
		row = 6
	}
	x, y := i*cellWidth, row*cellHeight
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

// pushAnimation elige la fila de empujar la camilla según hacia dónde se
// movió más quien empuja (dx, dy): igual que walkAnimation, pero con las
// filas propias de empujar (Samuel, 16-10: antes siempre salía la lateral,
// aunque el tramo fuera vertical).
func pushAnimation(dx, dy float64) (anim animation, flip bool) {
	if math.Abs(dx) > math.Abs(dy) {
		return animPush, dx > 0
	}
	if dy < 0 {
		return animPushUp, false
	}
	return animPushDown, false
}

// spritePlacement calcula la GeoM (la transformación) con la que se dibuja
// el cuadro de un paciente, y dice si va rotado. Las transformaciones se
// aplican en orden, como en el código: primero voltear o rotar en el
// cuadro de la hoja, después escalar ×2 y al final mover a su sitio.
//
//   - En la cama (dormido en cama, o el cuadro 1 de despertarse) el cuadro
//     horizontal se rota 90° en sentido horario, para que la cabeza quede
//     hacia la cabecera, y se pone en bedAt (ver bedCell).
//   - En cualquier otro caso se dibuja tal cual en at, sin rotar (en el
//     piso el cuerpo horizontal va así). Si flip, se voltea en su cuadro.
//
// Las capas de camisa, pelo y sombrero se dibujan con esta misma GeoM, así
// rotan, se voltean y se escalan junto con el cuerpo (ver layers.go).
func spritePlacement(a animation, frame int, bedAt vec, hasBed bool, at vec, flip bool) (ebiten.GeoM, bool) {
	var g ebiten.GeoM
	inBedFrame := a == animSleepBed || (a == animWakeUp && frame == 0)
	if hasBed && inBedFrame {
		g.Rotate(math.Pi / 2)      // 90° horario: lo que estaba a la izquierda (la cabeza) queda arriba
		g.Translate(cellHeight, 0) // al rotar, el cuadro queda en x negativas: se devuelve (ahora mide 24×16)
		g.Scale(spriteScale, spriteScale)
		g.Translate(bedAt.x, bedAt.y)
		return g, true
	}
	if flip {
		g.Scale(-1, 1)            // espejo horizontal…
		g.Translate(cellWidth, 0) // …y de vuelta a su cuadro
	}
	g.Scale(spriteScale, spriteScale)
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

// loadSprites carga como imágenes de Ebitengine todas las hojas de las
// capas (las de piel repetidas se cargan una vez).
func loadSprites() error {
	paths := append(bodySheets[:], doctorSheets[:]...) // el arreglo está lleno: append copia
	paths = append(paths, orderlySheets[:]...)
	for _, p := range shirtSheets {
		paths = append(paths, p)
	}
	for _, p := range hairSheets {
		paths = append(paths, p)
	}
	for _, p := range hatSheets {
		paths = append(paths, p)
	}
	for _, p := range paths {
		if _, done := sheetImgs[p]; done {
			continue
		}
		img, err := decodePNG(assets.Sprites, p)
		if err != nil {
			return err
		}
		sheetImgs[p] = ebiten.NewImageFromImage(img)
	}
	return nil
}
