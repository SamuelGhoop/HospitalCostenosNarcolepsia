package ui

import (
	"encoding/json"
	"fmt"
	"image"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// El menú de la portada: el portapapeles inclinado con sus 6 opciones
// (TITLE_ASSETS.md). Aquí solo está la lógica (qué opción está
// seleccionada, el cursor, la confirmación y el mouse); lo dibuja la
// portada con la imagen que dice image().

// option es una opción del menú.
type option int

const (
	optContinue option = iota // CONTINUAR (en gris hasta que haya partida guardada, F6)
	optNew                    // NUEVA PARTIDA
	optDemo                   // DEMO SECCIÓN 6
	optSettings               // AJUSTES
	optCredits                // CRÉDITOS
	optExit                   // SALIR
)

// menuOptions en el orden del portapapeles, de arriba abajo.
var menuOptions = []option{optContinue, optNew, optDemo, optSettings, optCredits, optExit}

// optionFile es cómo se llama cada opción en los archivos del portapapeles.
var optionFile = map[option]string{
	optContinue: "continuar", optNew: "nueva", optDemo: "demo",
	optSettings: "ajustes", optCredits: "creditos", optExit: "salir",
}

func (o option) String() string { return optionFile[o] }

// Tiempos del cursor (TITLE_ASSETS.md).
const (
	cursorTicks  = 21 // los 2 cuadros del cursor se alternan cada 0,35 s
	confirmTicks = 27 // al confirmar, el cursor se queda quieto 0,45 s antes de la acción
)

// clipboardPath es la imagen del portapapeles con la opción o seleccionada
// y el cursor en el cuadro frame. Sin partida guardada se usan las que
// tienen CONTINUAR en gris (clipboard_sincontinuar_…).
func clipboardPath(o option, hasSave bool, frame int) string {
	prefix := "clipboard_sincontinuar_"
	if hasSave {
		prefix = "clipboard_"
	}
	return fmt.Sprintf("title/clipboard/%s%s_%d.png", prefix, optionFile[o], frame)
}

// menu es el estado del menú.
type menu struct {
	selected   option
	hasSave    bool // hay partida guardada (siempre false hasta F6)
	ticks      int  // para alternar el cursor
	confirming int  // ticks que faltan para la acción; 0 = no se está confirmando
}

// newMenu arranca en la primera opción que se puede elegir.
func newMenu(hasSave bool) menu {
	m := menu{selected: optContinue, hasSave: hasSave}
	if !m.enabled(optContinue) {
		m.selected = optNew
	}
	return m
}

// enabled dice si una opción se puede elegir: todas, menos CONTINUAR sin
// partida guardada.
func (m *menu) enabled(o option) bool { return o != optContinue || m.hasSave }

// move sube (delta −1) o baja (+1) la selección, dando la vuelta y
// saltándose las opciones que no se pueden elegir.
func (m *menu) move(delta int) {
	n := len(menuOptions)
	i := int(m.selected)
	for {
		i = (i + delta + n) % n
		if m.enabled(option(i)) {
			m.selected = option(i)
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
func (m *menu) image() string { return clipboardPath(m.selected, m.hasSave, m.cursorFrame()) }

// ─── Mouse ─────────────────────────────────────────────────────────────

// hitboxes son los renglones del portapapeles para el mouse: los 4
// vértices de cada uno, ya inclinados, en píxeles de pantalla.
type hitboxes map[option][]image.Point

// loadHitboxes lee title/clipboard/hitboxes.json.
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
	for _, o := range menuOptions {
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

// at dice sobre qué renglón está el punto p, si está sobre alguno.
func (h hitboxes) at(p image.Point) (option, bool) {
	for _, o := range menuOptions {
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
