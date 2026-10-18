package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Press Start 2P es un tipo "de letrero" (cada letra ocupa su cuadro
// completo), mucho más ancha que Go Mono con el mismo tamaño en puntos.
// Estos tests comprueban, con el ancho REAL medido (textWidth, la misma
// función que usa el dibujo), que ningún texto fijo de la interfaz se sale
// de su caja. GAME_DESIGN §9.3 y §12.8.

// TestButtonLabels_FitTheirBox: los botones de la escena de juego centran
// su texto con textWidth, pero si el texto fuera más ancho que el botón,
// el centrado daría un margen negativo y el texto saldría por los bordes.
func TestButtonLabels_FitTheirBox(t *testing.T) {
	needFonts(t)
	cases := []struct {
		text string
		face text.Face
		w    int // ancho del botón en px
	}{
		{"REPORTE", smallFace, reportButton.Dx()},
		{"PAUSA", smallFace, pauseButton.Dx()},
		{"SIGUIENTE DÍA", boldFace, dayOverButton.Dx()},
		{"NUEVA PARTIDA", boldFace, newGameButton.Dx()},
		{"MENÚ", boldFace, menuButton.Dx()},
	}
	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			if w := textWidth(c.text, c.face); w >= float64(c.w) {
				t.Errorf("%q mide %.0f px; el botón mide %d px de ancho", c.text, w, c.w)
			}
		})
	}
}

// TestZoneLabels_FitTheirZone: zoneLabelBox busca un sitio donde el rótulo
// quepa dentro de su zona (box.In(info.rect)); si ninguno sirve, se queda
// con el primer candidato aunque se salga. El test pide que SÍ haya
// quepa: que el rótulo de cada zona quede dentro de su rectángulo.
func TestZoneLabels_FitTheirZone(t *testing.T) {
	needFonts(t)
	for z, info := range zones {
		box := zoneLabelBox(z)
		if !box.In(info.rect) {
			t.Errorf("zona %q: su rótulo (%v) no cabe dentro de la zona (%v)", info.label, box, info.rect)
		}
	}
}

// TestIDTags_FitUnderTheirFigure: la etiqueta de ID (P-001, D-02...) va
// centrada bajo el personaje; a smallFace debe quedar más angosta que el
// ancho de un cuadro de personaje más el margen que le da centeredBox.
func TestIDTags_FitUnderTheirFigure(t *testing.T) {
	needFonts(t)
	ids := []string{"P-001", "P-005", "D-01", "D-02", "C-01"}
	for _, id := range ids {
		if w := textWidth(id, smallFace); w >= float64(figWidth*3) {
			t.Errorf("la etiqueta %q mide %.0f px; se esperaba menos de %d (3 cuadros de personaje)", id, w, figWidth*3)
		}
	}
}

// TestRoomBadges_FitTheRoom: el letrero OCUPADA/DISPONIBLE se centra sobre
// el borde de abajo de la habitación más angosta (101, 102 o 103).
func TestRoomBadges_FitTheRoom(t *testing.T) {
	needFonts(t)
	narrowest := zones[zoneRoom101].rect.Dx()
	for _, txt := range []string{"OCUPADA", "DISPONIBLE"} {
		if w := textWidth(txt, smallFace); w >= float64(narrowest) {
			t.Errorf("%q mide %.0f px; la habitación más angosta mide %d px", txt, w, narrowest)
		}
	}
}

// TestDemoSubtitles_FitTheBox: la barra de abajo de la demo (drawSubtitles)
// muestra, con textFace a 16 px, la llamada real al modelo y su resultado;
// algunas son las más largas de toda la interfaz (varias llamadas
// encadenadas con "·"). Desde el 2026-10-17 se parten en hasta dos
// renglones (wrapTwoLines, pixelfont.go); el test pide que CADA renglón
// quepa en el ancho de la barra, no la frase entera de una sola vez.
func TestDemoSubtitles_FitTheBox(t *testing.T) {
	needFonts(t)
	box := demoSubtitleBoxWidth()
	longest := []string{
		"karen.DiagnosePatient(P-001, P-002, P-005) · efrain.DiagnosePatient(P-003, P-004)",
		"h.PatientsInHallway() · d.MyEpisodes() · h.Rooms() · h.SevereReport()",
		"ESPACIO siguiente · A automático · S saltar · C subtítulos · Esc pausa",
		"Cada paso hace la llamada REAL al modelo (paquete hospital) y muestra lo que respondió.",
	}
	for _, s := range longest {
		face := textFace
		if strings.HasPrefix(s, "ESPACIO") {
			face = smallFace // el hint de la esquina usa smallFace
		}
		for _, line := range wrapTwoLines(s, box, face.Size) {
			if w := textWidth(line, face); w >= float64(box) {
				t.Errorf("%q (de %q) mide %.0f px; la barra de la demo mide %d px de ancho útil", line, s, w, box)
			}
		}
	}
}

// TestReportRows_FitThePaper: las filas del Shift Report (report_scene.go)
// van con textFace desde x=92 hasta el borde derecho del papel (1208). La
// fila de episodios es la más larga: ID, hora, paciente, ubicación y
// destino de un extremo a otro del hospital. Desde el 2026-10-17, si no
// cabe entera se recorta con "…" (truncateWithEllipsis) en vez de
// achicarse: el test pide que el resultado quepa.
func TestReportRows_FitThePaper(t *testing.T) {
	needFonts(t)
	const paperRight = 1208
	const rowX = 92 + 20 // c.x inicial (92) más la sangría de column.indented (20)
	avail := paperRight - rowX
	row := fmt.Sprintf("%s %s  %s %s · %s → %s",
		"EP-01", "2026-10-20 14:15", "P-001", "Wilfrido Berrío", "pasillo 2, segundo piso", "habitación 103")
	fitted := truncateWithEllipsis(row, avail, textFace.Size)
	if w := textWidth(fitted, textFace); w >= float64(avail) {
		t.Errorf("la fila de episodio recortada (%q) mide %.0f px; el papel deja %d px de ese punto al borde", fitted, w, avail)
	}
}

// TestDayOverLines_FitTheBox: "Puntaje: N" y "Mejor racha: N" (§12.9),
// centradas en el panel del fin de día (drawDayOver, game_scene.go), de
// x=400 a x=880: con textFace a 16 px tienen que caber enteras.
func TestDayOverLines_FitTheBox(t *testing.T) {
	needFonts(t)
	const boxWidth = 880 - 400
	for _, s := range []string{"Puntaje: 123456", "Mejor racha: 99"} {
		if w := textWidth(s, textFace); w >= float64(boxWidth) {
			t.Errorf("%q mide %.0f px; el panel del fin de día mide %d px de ancho", s, w, boxWidth)
		}
	}
}
