package ui

import (
	"image"
	"testing"

	"golang.org/x/image/font/sfnt"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// §10.5: en Press Start 2P la Ó, É, Í, Ú, Á y Ñ mayúsculas vienen encogidas.
// accentMarks las cambia por su letra normal y devuelve los rectángulos de
// la tilde, que van ENCIMA de la letra (por arriba del borde de la línea,
// porque las mayúsculas llenan la línea entera).
func TestAccentMarks_ReplaceUppercaseAccentsAndDrawTheMarkAbove(t *testing.T) {
	cases := []struct {
		in, plain string
		marked    []int // posición (en letras) de cada letra con tilde
	}{
		{"SECCIÓN", "SECCION", []int{5}},
		{"COSTEÑOS", "COSTENOS", []int{5}},
		{"CRÉDITOS", "CREDITOS", []int{2}},
		{"ÁÉÍÓÚÑ", "AEIOUN", []int{0, 1, 2, 3, 4, 5}},
		{"HOSPITAL DE LOS", "HOSPITAL DE LOS", nil},
		{"con Narcolepsia · Costeños", "con Narcolepsia · Costeños", nil}, // las minúsculas no se tocan
	}
	for _, c := range cases {
		const size = 16 // cada píxel de la fuente mide 2 px
		plain, marks := accentMarks(c.in, size)
		if plain != c.plain {
			t.Errorf("accentMarks(%q): texto %q; se esperaba %q", c.in, plain, c.plain)
		}
		// Las marcas de cada letra: dentro de su columna y arriba de la línea.
		perLetter := map[int]int{}
		for _, m := range marks {
			col := m.Min.X / size // cada letra ocupa `size` px de ancho
			perLetter[col]++
			if m.Max.X > (col+1)*size || m.Max.Y > 0 {
				t.Errorf("accentMarks(%q): la marca %v se sale de la columna %d o baja a la letra (y > 0)", c.in, m, col)
			}
		}
		if len(perLetter) != len(c.marked) {
			t.Errorf("accentMarks(%q): marcas en %v; se esperaban sobre las letras %v", c.in, perLetter, c.marked)
		}
		for _, i := range c.marked {
			if perLetter[i] == 0 {
				t.Errorf("accentMarks(%q): falta la tilde sobre la letra %d", c.in, i)
			}
		}
	}
}

// Las marcas crecen con la fuente: a 72 px cada píxel de la fuente son 9.
func TestAccentMarks_ScaleWithTheFontSize(t *testing.T) {
	_, small := accentMarks("Ó", 8)
	_, big := accentMarks("Ó", 72)
	if len(small) == 0 || len(small) != len(big) {
		t.Fatalf("Ó: %d marcas a 8 px y %d a 72 px", len(small), len(big))
	}
	for i := range small {
		if want := image.Rect(small[i].Min.X*9, small[i].Min.Y*9, small[i].Max.X*9, small[i].Max.Y*9); big[i] != want {
			t.Errorf("marca %d a 72 px = %v; se esperaba %v (la de 8 px ×9)", i, big[i], want)
		}
	}
}

// Todos los caracteres que se escriben con Press Start 2P (ya sin las
// tildes mayúsculas, que se dibujan a mano) existen en la fuente: así no
// aparece ningún cuadrito vacío.
func TestPixelFont_HasEveryCharacterWeUse(t *testing.T) {
	data, err := assets.Fonts.ReadFile(pixelFontPath)
	if err != nil {
		t.Fatal(err)
	}
	f, err := sfnt.Parse(data)
	if err != nil {
		t.Fatalf("la fuente no se pudo leer: %v", err)
	}
	var buf sfnt.Buffer
	for _, s := range pixelTexts() {
		plain, _ := accentMarks(s, 8)
		for _, r := range plain {
			if gi, err := f.GlyphIndex(&buf, r); err != nil || gi == 0 {
				t.Errorf("%q: Press Start 2P no tiene %q", s, r)
			}
		}
	}
}
