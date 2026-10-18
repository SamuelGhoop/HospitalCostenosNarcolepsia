package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// Press Start 2P (OFL): la fuente pixel de la portada y los créditos
// (GAME_DESIGN §10.5). Cada letra es un cuadro de 8×8 "píxeles de fuente";
// se usa en tamaños múltiplos de 8 para que cada uno caiga en píxeles
// enteros y se vea nítida. El resto de la interfaz sigue en Go Mono.

// pixelFontPath: el TTF dentro de assets.Fonts.
const pixelFontPath = "fonts/PressStart2P-Regular.ttf"

var (
	pixelSource *text.GoTextFaceSource
	pixelFaces  = map[float64]*text.GoTextFace{} // una cara por tamaño
)

// loadPixelFont lee el TTF embebido.
func loadPixelFont() error {
	data, err := assets.Fonts.ReadFile(pixelFontPath)
	if err != nil {
		return fmt.Errorf("leyendo %s: %w", pixelFontPath, err)
	}
	src, err := text.NewGoTextFaceSource(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("cargando Press Start 2P: %w", err)
	}
	pixelSource = src
	return nil
}

// pixelFace devuelve la cara de Press Start 2P de ese tamaño.
func pixelFace(size float64) *text.GoTextFace {
	if f, ok := pixelFaces[size]; ok {
		return f
	}
	f := &text.GoTextFace{Source: pixelSource, Size: size}
	pixelFaces[size] = f
	return f
}

// ─── Tildes dibujadas a mano ───────────────────────────────────────────

// Las tildes en píxeles de fuente (cada letra es un cuadro de 8×8 y las
// mayúsculas llenan las filas 0 a 6). Van por ENCIMA de la línea: y < 0.
var (
	// acute: la tilde ´, una diagonal de dos escalones que sube a la derecha.
	acute = []image.Rectangle{image.Rect(4, -4, 6, -3), image.Rect(3, -3, 5, -2)}
	// tildeN: la virgulilla de la Ñ, la misma de la maqueta (tilde-n).
	tildeN = []image.Rectangle{
		image.Rect(1, -4, 3, -3), image.Rect(5, -4, 6, -3),
		image.Rect(0, -3, 1, -2), image.Rect(3, -3, 5, -2),
	}
)

// plainLetter dice qué letra normal reemplaza a una mayúscula con tilde y
// qué marca lleva encima. ok es false si la letra no se cambia.
func plainLetter(r rune) (plain rune, mark []image.Rectangle, ok bool) {
	switch r {
	case 'Á':
		return 'A', acute, true
	case 'É':
		return 'E', acute, true
	case 'Í':
		return 'I', acute, true
	case 'Ó':
		return 'O', acute, true
	case 'Ú':
		return 'U', acute, true
	case 'Ñ':
		return 'N', tildeN, true
	}
	return r, nil, false
}

// accentMarks prepara un texto para Press Start 2P: cambia cada mayúscula
// con tilde por su letra normal y devuelve, en píxeles de pantalla y
// relativos a la esquina del texto, los rectángulos de las tildes que hay
// que dibujar encima. size es el tamaño de la fuente (múltiplo de 8).
func accentMarks(s string, size float64) (plain string, marks []image.Rectangle) {
	u := int(size) / 8 // píxeles de pantalla por píxel de fuente
	var out []rune
	for i, r := range []rune(s) {
		p, mark, ok := plainLetter(r)
		out = append(out, p)
		if !ok {
			continue
		}
		x0 := i * 8 * u // cada letra avanza 8 píxeles de fuente
		for _, m := range mark {
			marks = append(marks, image.Rect(x0+m.Min.X*u, m.Min.Y*u, x0+m.Max.X*u, m.Max.Y*u))
		}
	}
	return string(out), marks
}

// ─── Dibujo ────────────────────────────────────────────────────────────

// textLayer es una pasada del texto: un color corrido (dx, dy). El logo se
// arma con varias: sombra, contorno en 8 direcciones y el relleno encima.
// Cada pasada dibuja las letras y sus tildes.
type textLayer struct {
	dx, dy int
	c      color.Color
}

// outlined devuelve las 8 capas de un contorno de w px alrededor del texto.
func outlined(w int, c color.Color) []textLayer {
	var out []textLayer
	for _, d := range []image.Point{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}} {
		out = append(out, textLayer{d.X * w, d.Y * w, c})
	}
	return out
}

// drawPixelText escribe s con Press Start 2P (y sus tildes a mano) con la
// esquina en (x, y), pasada por pasada: las primeras quedan debajo.
func drawPixelText(dst *ebiten.Image, s string, x, y int, size float64, layers []textLayer, alpha float32) {
	plain, marks := accentMarks(s, size)
	for _, l := range layers {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+l.dx), float64(y+l.dy))
		op.ColorScale.ScaleWithColor(l.c)
		op.ColorScale.ScaleAlpha(alpha)
		text.Draw(dst, plain, pixelFace(size), op)
		for _, m := range marks {
			r := m.Add(image.Pt(x+l.dx, y+l.dy))
			cr, cg, cb, ca := l.c.RGBA()
			faded := color.NRGBA{uint8(cr >> 8), uint8(cg >> 8), uint8(cb >> 8), uint8(float32(ca>>8) * alpha)}
			fillRect(dst, r, faded)
		}
	}
}

// pixelTextWidth es lo que mide s con Press Start 2P: 8 píxeles de fuente
// por letra (es monoespaciada).
func pixelTextWidth(s string, size float64) int {
	return len([]rune(s)) * int(size)
}

// wrapTwoLines parte s en dos renglones por palabras, para que cada uno
// quepa en maxWidth: el primero se llena con todas las palabras que
// entren, y el resto queda en el segundo (que puede seguir sin caber, si
// una sola palabra ya es más ancha que maxWidth: no hay por dónde
// partirla). Si s ya cabe entera, devuelve un solo renglón. La usan los
// subtítulos de la demo (demo_scene.go) y el nombre del candidato de la
// bolsa de empleo (hiring_scene.go), que antes tenían cada uno su propia
// copia de este mismo partido por palabras.
func wrapTwoLines(s string, maxWidth int, size float64) []string {
	if pixelTextWidth(s, size) <= maxWidth {
		return []string{s}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{s}
	}
	line1 := words[0]
	i := 1
	for i < len(words) {
		grown := line1 + " " + words[i]
		if pixelTextWidth(grown, size) > maxWidth {
			break
		}
		line1 = grown
		i++
	}
	if i >= len(words) {
		return []string{line1}
	}
	return []string{line1, strings.Join(words[i:], " ")}
}

// truncateWithEllipsis recorta s a lo que quepa en maxWidth, cambiando el
// final por "…" si hizo falta cortarla. La usan las filas del reporte
// (report_scene.go), que no tienen cómo partirse en dos renglones sin
// desordenar la columna.
func truncateWithEllipsis(s string, maxWidth int, size float64) string {
	if pixelTextWidth(s, size) <= maxWidth {
		return s
	}
	runes := []rune(s)
	for n := len(runes) - 1; n > 0; n-- {
		cut := string(runes[:n]) + "…"
		if pixelTextWidth(cut, size) <= maxWidth {
			return cut
		}
	}
	return "…"
}
