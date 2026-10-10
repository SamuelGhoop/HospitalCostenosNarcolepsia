package ui

import (
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// overlay es un rótulo que se dibuja encima del mapa: una burbuja "Zzz",
// un rayito de nivel, el punto del personal, una etiqueta con el ID o un
// letrero de habitación.
//
// Las posiciones se calculan aquí UNA sola vez: las usan el dibujo y el
// test que comprueba que ningún rótulo se monta sobre otro.
type overlay struct {
	rect   image.Rectangle
	text   string // "" para rayitos y puntos
	fg, bg color.Color
}

// textBox es la caja de un rótulo de texto con su esquina en (left, top).
func textBox(s string, left, top int, face text.Face) image.Rectangle {
	m := face.Metrics()
	w := int(textWidth(s, face)) + 8
	h := int(m.HAscent+m.HDescent) + 4
	return image.Rect(left, top, left+w, top+h)
}

// centeredBox es la caja de un rótulo centrado horizontalmente en centerX.
func centeredBox(s string, centerX, top int, face text.Face) image.Rectangle {
	r := textBox(s, 0, top, face)
	return r.Add(image.Pt(centerX-r.Dx()/2, 0))
}

// figureOverlays calcula los rótulos de un personaje que está en at.
// showBubble dice si ya toca la burbuja Zzz (solo cuando está acostado).
func figureOverlays(f figure, at vec, showBubble bool) []overlay {
	x, y := int(math.Round(at.x)), int(math.Round(at.y))
	var out []overlay

	// Acostado en su cama: los rótulos van a la DERECHA de la cama, dentro
	// de la habitación (encima de la cama se montaban con los demás).
	if f.patient && f.room != 0 && at == bedCell(roomZone(f.room)) {
		bed := bedRect(roomZone(f.room))
		if txt, bg, ok := bubbleFor(f.state); ok && showBubble {
			out = append(out, bubbleOverlays(txt, bg, f.level, textBox(txt, bed.Max.X+4, bed.Min.Y+4, smallFace))...)
		}
		return append(out, overlay{rect: textBox(f.id, bed.Max.X+4, bed.Min.Y+28, smallFace), text: f.id, fg: colInk, bg: colPaper})
	}

	if f.patient {
		if txt, bg, ok := bubbleFor(f.state); ok && showBubble {
			out = append(out, bubbleOverlays(txt, bg, f.level, centeredBox(txt, x+figWidth/2, y-20, smallFace))...)
		}
	} else {
		// Punto verde = libre. En la demo el personal nunca queda ocupado.
		out = append(out, overlay{rect: image.Rect(x+12, y-10, x+20, y-2), bg: colGreen})
	}
	// En la demo los IDs se muestran siempre, para seguir los subtítulos.
	return append(out, overlay{rect: centeredBox(f.id, x+figWidth/2, y+figHeight+2, smallFace), text: f.id, fg: colInk, bg: colPaper})
}

// bubbleOverlays: la burbuja Zzz en box y, a su derecha, los rayitos de
// nivel (1 amarillo leve, 2 naranja moderada, 3 rojos severa).
func bubbleOverlays(txt string, bg color.RGBA, level hospital.NarcolepsyLevel, box image.Rectangle) []overlay {
	out := []overlay{{rect: box, text: txt, fg: colWhite, bg: bg}}
	c, n := levelStyle(level)
	for i := 0; i < n; i++ {
		bolt := image.Rect(box.Max.X+4+i*8, box.Min.Y+4, box.Max.X+10+i*8, box.Max.Y-4)
		out = append(out, overlay{rect: bolt, bg: c})
	}
	return out
}

// roomBadges: el letrero OCUPADA / DISPONIBLE de cada habitación, centrado
// sobre el borde de abajo como en la maqueta. El texto sale del String()
// del modelo.
func roomBadges(rooms []game.RoomView) []overlay {
	var out []overlay
	for _, r := range rooms {
		bg := colBadgeAvailable
		if r.State == hospital.Occupied {
			bg = colBadgeOccupied
		}
		room := zones[roomZone(r.Number)].rect
		txt := strings.ToUpper(r.State.String())
		out = append(out, overlay{rect: centeredBox(txt, room.Min.X+room.Dx()/2, room.Max.Y-10, smallFace), text: txt, fg: colWhite, bg: bg})
	}
	return out
}

// zoneLabelBox es dónde va el rótulo de una zona: abajo a la izquierda,
// donde no estorban las camas ni los que están de pie, salvo el pasillo 1,
// que es muy bajito y lo lleva arriba.
func zoneLabelBox(z zone) image.Rectangle {
	info := zones[z]
	top := info.rect.Max.Y - 22
	if z == zoneHallway1 {
		top = info.rect.Min.Y + 2
	}
	return textBox(info.label, info.rect.Min.X+2, top, smallFace)
}

// drawOverlay dibuja un rótulo: fondo con borde negro y, si tiene, su texto.
func drawOverlay(dst *ebiten.Image, o overlay) {
	panel(dst, o.rect, o.bg)
	if o.text != "" {
		drawText(dst, o.text, float64(o.rect.Min.X+4), float64(o.rect.Min.Y+2), smallFace, o.fg)
	}
}
