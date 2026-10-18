package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// Paleta tomada de las maquetas de Claude Design (hospital.dc.html,
// Personaje.dc.html y Title Screen.dc.html). Cada comentario es el color
// tal como aparece en la maqueta, para poder buscarlo allá.
var (
	colInk        = color.RGBA{0x14, 0x14, 0x14, 0xff} // #141414 contornos y texto
	colWhite      = color.RGBA{0xff, 0xff, 0xff, 0xff} // #ffffff
	colPaper      = color.RGBA{0xfb, 0xf8, 0xef, 0xff} // #fbf8ef papel, etiquetas, bata del médico
	colPaperShade = color.RGBA{0xe3, 0xdd, 0xcc, 0xff} // #e3ddcc sombra del papel
	colMuted      = color.RGBA{0x5a, 0x61, 0x6b, 0xff} // #5a616b texto secundario
	colLightGray  = color.RGBA{0xc9, 0xce, 0xd6, 0xff} // #c9ced6 muebles, botón deshabilitado
	colMetal      = color.RGBA{0x8a, 0x90, 0x9a, 0xff} // #8a909a cabecera de la cama, gancho

	colSand     = color.RGBA{0xf3, 0xd6, 0x8e, 0xff} // #f3d68e arena (exterior)
	colWall     = color.RGBA{0xb9, 0xbf, 0xc8, 0xff} // #b9bfc8 muros del edificio
	colSidewalk = color.RGBA{0xe3, 0xe5, 0xe8, 0xff} // #e3e5e8 acera
	colStreet   = color.RGBA{0x4a, 0x4f, 0x57, 0xff} // #4a4f57 calle

	colRoomFloor      = color.RGBA{0xec, 0xea, 0xe4, 0xff} // #eceae4 habitaciones
	colRadiologyFloor = color.RGBA{0xdf, 0xe6, 0xee, 0xff} // #dfe6ee radiología
	colHallFloor      = color.RGBA{0xdd, 0xe1, 0xe6, 0xff} // #dde1e6 pasillos
	colCafeFloor      = color.RGBA{0xc6, 0xec, 0xe6, 0xff} // #c6ece6 cafetería
	colLobbyFloor     = color.RGBA{0xf6, 0xe7, 0xbd, 0xff} // #f6e7bd recepción (lobby)
	colStaffFloor     = color.RGBA{0xdc, 0xae, 0x78, 0xff} // #dcae78 sala del personal

	colBlue      = color.RGBA{0x2f, 0x6f, 0xd6, 0xff} // #2f6fd6
	colBlueDark  = color.RGBA{0x21, 0x52, 0xa8, 0xff} // #2152a8
	colLightBlue = color.RGBA{0x9d, 0xdc, 0xf7, 0xff} // #9ddcf7 vidrios, sábana
	colGreen     = color.RGBA{0x2f, 0xb8, 0x4f, 0xff} // #2fb84f
	colGreenDark = color.RGBA{0x1f, 0x8a, 0x3a, 0xff} // #1f8a3a
	colRed       = color.RGBA{0xe5, 0x48, 0x4d, 0xff} // #e5484d
	colRedDark   = color.RGBA{0xc4, 0x30, 0x2b, 0xff} // #c4302b
	colYellow    = color.RGBA{0xff, 0xd2, 0x3f, 0xff} // #ffd23f
	colOrange    = color.RGBA{0xf5, 0xa6, 0x23, 0xff} // #f5a623
	colTeal      = color.RGBA{0x22, 0xb8, 0xb0, 0xff} // #22b8b0 pantalón del médico

	colSkin  = color.RGBA{0xa8, 0x6b, 0x45, 0xff} // #a86b45 piel
	colPants = color.RGBA{0x3a, 0x3f, 0x47, 0xff} // #3a3f47 pantalón / zapatos

	colWood     = color.RGBA{0xb9, 0x77, 0x3f, 0xff} // #b9773f portapapeles
	colWoodDark = color.RGBA{0x8f, 0x5a, 0x2c, 0xff} // #8f5a2c sombra del portapapeles

	// Letreros de las habitaciones y burbujas "Zzz" (Personaje.dc.html).
	colBadgeOccupied  = colRedDark   // #c4302b OCUPADA
	colBadgeAvailable = colGreenDark // #1f8a3a DISPONIBLE
	colBubbleHallway  = colRedDark   // #c4302b "Zzz!" dormido sin cama
	colBubbleBed      = colBlue      // #2f6fd6 "Zzz" dormido en cama
)

// Fuentes. Desde el 2026-10-17 (prueba de Samuel, §12.8) TODA la interfaz
// usa Press Start 2P (antes solo la portada, los créditos y el aviso, y el
// resto Go Mono): los rótulos del mapa, los IDs, los tooltips, los avisos,
// los subtítulos de la demo, los botones, la pausa, el reporte y el Game
// Over.
//
// Press Start 2P es un tipo "de letrero": cada letra ocupa un cuadro
// cuadrado de su tamaño completo (un carácter a 16 px mide 16 px de
// ancho), mientras que Go Mono, aun siendo monoespaciada, deja la letra
// más angosta que su cuadro (una X a 16 px medía unos 9-10 px). Por eso no
// basta con copiar los tamaños de Go Mono: con el mismo número de puntos,
// Press Start 2P sale ~60 % más ancha.
//
// El 2026-10-08 textFace había bajado a 8 px para que los textos largos
// (la llamada al modelo de la barra de la demo, las filas del reporte)
// cupieran enteros en una sola línea. Prueba de Samuel el 2026-10-17: a
// 1280×720 eso se lee muy chico. textFace vuelve a 16 px (de nuevo igual
// de ancho que boldFace, como antes del 10-08) y los textos largos que no
// cabían ya se parten en dos renglones (wrapTwoLines, demo_scene.go) o se
// recortan con "…" (truncateWithEllipsis, report_scene.go) en vez de
// seguir achicando toda la letra.
//   - smallFace 8 (rótulos del mapa y del HUD, que ya se miden con su
//     propia fuente: caben igual o con más aire).
//   - textFace 16 (el texto "normal").
//   - boldFace 16 (títulos y botones cortos y ya centrados con
//     textWidth).
//
// Los tres siguen en múltiplos de 8 px, como pide Press Start 2P para
// verse nítida.
var (
	textFace  *text.GoTextFace // texto normal (16 px)
	smallFace *text.GoTextFace // rótulos pequeños (8 px)
	boldFace  *text.GoTextFace // títulos y botones (16 px; ya no hay un peso "negrita" propio)
)

// loadFonts prepara las fuentes. Devuelve error en vez de hacer panic: así
// main decide qué hacer si algo falla.
func loadFonts() error {
	if err := loadPixelFont(); err != nil {
		return err
	}
	textFace = pixelFace(16)
	smallFace = pixelFace(8)
	boldFace = pixelFace(16)
	return nil
}

// levelStyle devuelve el color de los "rayitos" de nivel y cuántos van:
// 1 amarillo (leve), 2 naranja (moderada), 3 rojos (severa).
func levelStyle(level hospital.NarcolepsyLevel) (color.RGBA, int) {
	switch level {
	case hospital.Moderate:
		return colOrange, 2
	case hospital.Severe:
		return colRed, 3
	default:
		return colYellow, 1
	}
}

// bubbleFor dice qué burbuja lleva un paciente según su estado en el
// modelo: roja "Zzz!" si duerme sin cama, azul "Zzz" si duerme en cama.
// ok es false si está despierto (no lleva burbuja).
func bubbleFor(state hospital.PatientState) (txt string, bg color.RGBA, ok bool) {
	switch state {
	case hospital.AsleepInHallway:
		return "Zzz!", colBubbleHallway, true
	case hospital.AsleepInBed:
		return "Zzz", colBubbleBed, true
	default:
		return "", color.RGBA{}, false
	}
}

// ─── Ayudantes de dibujo ───────────────────────────────────────────────

// fillRect pinta un rectángulo sólido (sin suavizado: pixel art nítido).
func fillRect(dst *ebiten.Image, r image.Rectangle, c color.Color) {
	vector.FillRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
}

// outlineWidth: el borde de los rótulos mide 2 px, el doble que a 640×360.
const outlineWidth = 2

// outline dibuja un borde POR FUERA del rectángulo, como la función edge()
// de la maqueta.
func outline(dst *ebiten.Image, r image.Rectangle, c color.Color) {
	const w = outlineWidth
	fillRect(dst, image.Rect(r.Min.X-w, r.Min.Y-w, r.Max.X+w, r.Min.Y), c) // arriba
	fillRect(dst, image.Rect(r.Min.X-w, r.Max.Y, r.Max.X+w, r.Max.Y+w), c) // abajo
	fillRect(dst, image.Rect(r.Min.X-w, r.Min.Y, r.Min.X, r.Max.Y), c)     // izquierda
	fillRect(dst, image.Rect(r.Max.X, r.Min.Y, r.Max.X+w, r.Max.Y), c)     // derecha
}

// panel es un rectángulo con fondo y borde negro.
func panel(dst *ebiten.Image, r image.Rectangle, bg color.Color) {
	fillRect(dst, r, bg)
	outline(dst, r, colInk)
}

// drawText escribe s con su esquina superior izquierda en (x, y). Como toda
// la interfaz usa Press Start 2P, si face es una de sus caras (pixelFace)
// pasa por drawPixelText para que las mayúsculas con tilde (Á, É, Í, Ó, Ú,
// Ñ) lleven su tilde dibujada a mano encima (pixelfont.go, GAME_DESIGN
// §10.5): una sola pasada, sin contorno ni sombra.
func drawText(dst *ebiten.Image, s string, x, y float64, face text.Face, c color.Color) {
	if pf, ok := face.(*text.GoTextFace); ok {
		drawPixelText(dst, s, int(x), int(y), pf.Size, []textLayer{{0, 0, c}}, 1)
		return
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, face, op)
}

// textWidth mide cuántos píxeles de ancho ocupa s con esa fuente.
func textWidth(s string, face text.Face) float64 {
	w, _ := text.Measure(s, face, 0)
	return w
}
