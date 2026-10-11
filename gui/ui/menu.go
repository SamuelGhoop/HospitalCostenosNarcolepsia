package ui

import (
	"encoding/json"
	"fmt"
	"image"
	"slices"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// Los menús de portapapeles: el de la portada, inclinado y con 6 opciones
// (TITLE_ASSETS.md), y el de la pausa, derecho y con 3 (PAUSE_ASSETS.md).
// Aquí solo está la lógica (qué opción está seleccionada, el cursor, la
// confirmación y el mouse); los dibuja su escena con la imagen que dice
// image().

// option es una opción de un menú.
type option int

const (
	optContinue option = iota // CONTINUAR (en gris hasta que haya partida guardada, F6)
	optNew                    // NUEVA PARTIDA
	optDemo                   // DEMO SECCIÓN 6
	optSettings               // AJUSTES (en la portada y en la pausa)
	optCredits                // CRÉDITOS
	optExit                   // SALIR (cierra el programa)
	optResume                 // REANUDAR (pausa)
	optToTitle                // SALIR AL MENÚ (pausa)

	optionCount // no es una opción: cuántas hay, para recorrerlas en orden
)

// Las opciones de cada portapapeles, de arriba abajo.
var (
	titleOptions = []option{optContinue, optNew, optDemo, optSettings, optCredits, optExit}
	pauseOptions = []option{optResume, optSettings, optToTitle}
)

// optionFile es cómo se llama cada opción en los archivos de los
// portapapeles.
var optionFile = map[option]string{
	optContinue: "continuar", optNew: "nueva", optDemo: "demo",
	optSettings: "ajustes", optCredits: "creditos", optExit: "salir",
	optResume: "reanudar", optToTitle: "salir",
}

func (o option) String() string { return optionFile[o] }

// Tiempos del cursor (TITLE_ASSETS.md y PAUSE_ASSETS.md).
const (
	cursorTicks  = 21 // los 2 cuadros del cursor se alternan cada 0,35 s
	confirmTicks = 27 // al confirmar, el cursor se queda quieto 0,45 s antes de la acción
)

// imagePath es la imagen de un portapapeles con la opción o seleccionada y
// el cursor en el cuadro frame: prefix + opción + cuadro.
func imagePath(prefix string, o option, frame int) string {
	return fmt.Sprintf("%s%s_%d.png", prefix, optionFile[o], frame)
}

// clipboardPrefix: sin partida guardada, la portada usa las imágenes que
// tienen CONTINUAR en gris (clipboard_sincontinuar_…).
func clipboardPrefix(hasSave bool) string {
	if hasSave {
		return "title/clipboard/clipboard_"
	}
	return "title/clipboard/clipboard_sincontinuar_"
}

// pausePrefix: las imágenes de la pausa son pause/pause_<opción>_<cuadro>.png.
const pausePrefix = "pause/pause_"

// clipboardPath es la imagen del portapapeles de la portada.
func clipboardPath(o option, hasSave bool, frame int) string {
	return imagePath(clipboardPrefix(hasSave), o, frame)
}

// pausePath es la imagen del portapapeles de la pausa.
func pausePath(o option, frame int) string { return imagePath(pausePrefix, o, frame) }

// menu es el estado de un portapapeles.
type menu struct {
	options    []option // de arriba abajo
	prefix     string   // con qué empiezan sus imágenes
	selected   option
	hasSave    bool // portada: hay partida guardada (siempre false hasta F6)
	ticks      int  // para alternar el cursor
	confirming int  // ticks que faltan para la acción; 0 = no se está confirmando
}

// newTitleMenu es el menú de la portada: arranca en la primera opción que
// se puede elegir.
func newTitleMenu(hasSave bool) menu {
	m := menu{options: titleOptions, prefix: clipboardPrefix(hasSave), selected: optContinue, hasSave: hasSave}
	if !m.enabled(optContinue) {
		m.selected = optNew
	}
	return m
}

// newPauseMenu es el menú de la pausa: arranca en REANUDAR.
func newPauseMenu() menu {
	return menu{options: pauseOptions, prefix: pausePrefix, selected: optResume}
}

// enabled dice si una opción se puede elegir: todas, menos CONTINUAR sin
// partida guardada.
func (m *menu) enabled(o option) bool { return o != optContinue || m.hasSave }

// move sube (delta −1) o baja (+1) la selección, dando la vuelta y
// saltándose las opciones que no se pueden elegir.
func (m *menu) move(delta int) {
	n := len(m.options)
	i := slices.Index(m.options, m.selected)
	for {
		i = (i + delta + n) % n
		if m.enabled(m.options[i]) {
			m.selected = m.options[i]
			return
		}
	}
}

// update avanza el menú un tick con lo que hizo el jugador. Cuando termina
// la espera de una confirmación, devuelve la opción elegida y done = true.
func (m *menu) update(in input, boxes hitboxes) (chosen option, done bool) {
	m.ticks++
	if m.confirming > 0 { // confirmando: el cursor quieto y no se escucha nada más
		m.confirming--
		if m.confirming == 0 {
			return m.selected, true
		}
		return 0, false
	}

	if o, over := boxes.at(in.mouse); over && m.enabled(o) && (in.mouseMoved || in.click) {
		m.selected = o
	}
	switch {
	case in.up:
		m.move(-1)
	case in.down:
		m.move(+1)
	case in.confirm:
		m.confirming = confirmTicks
	case in.click:
		if o, over := boxes.at(in.mouse); over && m.enabled(o) {
			m.confirming = confirmTicks
		}
	}
	return 0, false
}

// cursorFrame es el cuadro del cursor que toca: alterna cada 0,35 s y se
// queda en el 0 mientras se confirma.
func (m *menu) cursorFrame() int {
	if m.confirming > 0 {
		return 0
	}
	return m.ticks / cursorTicks % 2
}

// image es la imagen del portapapeles que toca dibujar.
func (m *menu) image() string { return imagePath(m.prefix, m.selected, m.cursorFrame()) }

// ─── Mouse ─────────────────────────────────────────────────────────────

// hitboxes son los renglones de un portapapeles para el mouse: los 4
// vértices de cada uno en píxeles de pantalla (en la portada, ya
// inclinados).
type hitboxes map[option][]image.Point

// loadHitboxes lee title/clipboard/hitboxes.json: los 4 vértices de cada
// renglón.
func loadHitboxes() (hitboxes, error) {
	data, err := assets.Title.ReadFile("title/clipboard/hitboxes.json")
	if err != nil {
		return nil, err
	}
	var raw map[string][][2]int
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("leyendo hitboxes.json: %w", err)
	}
	boxes := hitboxes{}
	for _, o := range titleOptions {
		vertices, ok := raw[optionFile[o]]
		if !ok {
			return nil, fmt.Errorf("hitboxes.json no trae la opción %q", optionFile[o])
		}
		for _, v := range vertices {
			boxes[o] = append(boxes[o], image.Pt(v[0], v[1]))
		}
	}
	return boxes, nil
}

// loadPauseHitboxes lee pause/hitboxes.json, que trae un rectángulo [x, y,
// ancho, alto] por renglón, y lo pasa a sus 4 vértices para usar la misma
// prueba que la portada.
func loadPauseHitboxes() (hitboxes, error) {
	data, err := assets.Pause.ReadFile("pause/hitboxes.json")
	if err != nil {
		return nil, err
	}
	var raw map[string][4]int
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("leyendo pause/hitboxes.json: %w", err)
	}
	boxes := hitboxes{}
	for _, o := range pauseOptions {
		r, ok := raw[optionFile[o]]
		if !ok {
			return nil, fmt.Errorf("pause/hitboxes.json no trae la opción %q", optionFile[o])
		}
		x, y, w, h := r[0], r[1], r[2], r[3]
		boxes[o] = []image.Point{{x, y}, {x + w, y}, {x + w, y + h}, {x, y + h}}
	}
	return boxes, nil
}

// at dice sobre qué renglón está el punto p, si está sobre alguno. Recorre
// las opciones en orden para que el resultado no dependa del orden del map.
func (h hitboxes) at(p image.Point) (option, bool) {
	for o := option(0); o < optionCount; o++ {
		if poly, ok := h[o]; ok && insidePolygon(p, poly) {
			return o, true
		}
	}
	return 0, false
}

// insidePolygon es la prueba del rayo: desde p se traza una línea hacia la
// derecha y se cuentan los lados que cruza. Si son impares, p está adentro.
func insidePolygon(p image.Point, poly []image.Point) bool {
	inside := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		a, b := poly[i], poly[j]
		if (a.Y > p.Y) != (b.Y > p.Y) {
			// x donde el lado a–b cruza la altura de p.
			x := float64(a.X) + float64(p.Y-a.Y)*float64(b.X-a.X)/float64(b.Y-a.Y)
			if float64(p.X) < x {
				inside = !inside
			}
		}
	}
	return inside
}
