package ui

import (
	"fmt"
	"image"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// La entrada del modo juego (GAME_DESIGN §10.4): clic en un paciente que
// necesita a alguien (desplomado, en el pasillo o esperando revisión) y
// después en alguien del personal: Game.Dispatch. Clic derecho en un
// paciente revisado en su cama: Game.WakeEarly. Pasar el mouse muestra su
// ficha; Shift, todas las etiquetas. Los errores del juego salen como aviso.

// figureBox es lo que se ve de un personaje en la pantalla (para saber si el
// mouse está encima): en la cama, la cama; acostado en el piso, su cuerpo
// acostado; de pie, su cuerpo de pie.
func (s *GameScene) figureBox(f figure) image.Rectangle {
	at := s.pos[f.id]
	p := image.Pt(int(math.Round(at.x)), int(math.Round(at.y)))
	switch {
	case f.patient && s.onBed(f):
		return bedRect(roomZone(f.room))
	case f.patient && s.anims[f.id] != nil && (s.anims[f.id].showsBubble() || s.anims[f.id].anim == animCollapse):
		return floorBody.Add(p)
	default:
		return standingBody.Add(p)
	}
}

// clickableFigures son los personajes que cuentan para el clic, el hover y
// el imán: los mismos que antes (nadie en camilla).
func (s *GameScene) clickableFigures() []figure {
	var out []figure
	for _, f := range s.figures {
		if s.actorOf(f.id).exiting || (s.carryOf(f.id) != nil && f.id == s.carryOf(f.id).order.patient) {
			continue
		}
		out = append(out, f)
	}
	return out
}

// figureAt es el personaje que está bajo p: si hay varios, el que se dibuja
// encima (el de los pies más abajo). Los que van en camilla no cuentan.
//
// BLOQUE 1 del pase de juice (GAME_DESIGN §12.10, pedido de Samuel el
// 2026-10-19, "clics más fáciles"): la hitbox de cada uno crece +50 %
// (hitboxGrowFactor, juice.go) alrededor de su caja actual, y si el clic no
// cae en nadie así agrandado, se usa el imán (magnetFigureAt): el personaje
// clicable más cercano por distancia al centro de su caja, dentro de
// clickMagnetRadius píxeles de pantalla. No cambia la regla de despacho
// (sigue siendo clic paciente → clic personal): esto solo decide A QUIÉN
// apunta el clic, igual que antes.
func (s *GameScene) figureAt(p image.Point) (figure, bool) {
	figs := s.clickableFigures()
	var hits []figure
	for _, f := range figs {
		if p.In(growRect(s.figureBox(f), hitboxGrowFactor)) {
			hits = append(hits, f)
		}
	}
	if len(hits) == 0 {
		return s.magnetFigureAt(p, figs)
	}
	sort.SliceStable(hits, func(i, j int) bool { return s.pos[hits[i].id].y > s.pos[hits[j].id].y })
	return hits[0], true
}

// magnetFigureAt es el imán (§12.10 BLOQUE 1): entre figs, el más cercano a
// p por distancia al centro de su caja ACTUAL (sin agrandar: el imán ya hace
// más fácil llegar a alguien, no hace falta agrandar dos veces), si queda a
// clickMagnetRadius píxeles o menos. Sin nadie tan cerca, no hay imán.
func (s *GameScene) magnetFigureAt(p image.Point, figs []figure) (figure, bool) {
	best := math.Inf(1)
	var closest figure
	found := false
	for _, f := range figs {
		cx, cy := boxCenter(s.figureBox(f))
		d := math.Hypot(float64(p.X)-cx, float64(p.Y)-cy)
		if d <= clickMagnetRadius && d < best {
			best, closest, found = d, f, true
		}
	}
	return closest, found
}

// patientView y staffView buscan en la foto.
func (s *GameScene) patientView(id string) (game.GamePatientView, bool) {
	for _, p := range s.snap.Patients {
		if p.ID == id {
			return p, true
		}
	}
	return game.GamePatientView{}, false
}

func (s *GameScene) staffView(id string) (game.StaffMemberView, bool) {
	for _, st := range s.snap.Staff {
		if st.ID == id {
			return st, true
		}
	}
	return game.StaffMemberView{}, false
}

// needsSomeone dice si a un paciente se le puede mandar a alguien.
func needsSomeone(st game.Stage) bool {
	return st == game.Collapsed || st == game.InHallway || st == game.AwaitingReview
}

// click es el clic izquierdo en p: elige un paciente que necesita a alguien
// o, con uno elegido, manda a quien se clicó. Clic en otro sitio, nada.
//
// BLOQUE 2 del pase de juice (§12.10): todo clic que de verdad le pega a
// alguien deja su squash&stretch + destello blanco (s.fx.punch). Si el clic
// manda a alguien (Dispatch), un despacho exitoso deja además la línea
// punteada hasta el paciente; uno inválido tiñe el destello de rojo y
// tiembla la pantalla (s.fx.invalidClick).
func (s *GameScene) click(p image.Point) {
	f, ok := s.figureAt(p)
	switch {
	case !ok:
		s.selected = ""
	case f.patient:
		if pv, _ := s.patientView(f.id); needsSomeone(pv.Stage) && s.selected != f.id {
			s.selected = f.id
			s.fx.punch(f.id, colWhite)
		} else {
			s.selected = ""
		}
	case s.selected != "":
		from, _ := s.figure(s.selected)
		fx0, fy0 := boxCenter(s.figureBox(from))
		tx0, ty0 := boxCenter(s.figureBox(f))
		if err := s.g.Dispatch(s.selected, f.id); err != nil {
			s.say(err.Error())
			s.fx.punch(f.id, colRed)
			s.fx.invalidClick()
			playSound(sfxInvalid)
		} else {
			s.fx.punch(f.id, colWhite)
			s.fx.dispatchSuccess(vec{fx0, fy0}, vec{tx0, ty0})
			playSound(sfxDispatch)
		}
		s.selected = ""
	default:
		s.say("Primero haz clic en un paciente que necesite a alguien")
	}
}

// rightClick es el clic derecho en p: despierta antes de tiempo al
// paciente de su cama (cuesta reputación: GAME_DESIGN §5.7).
func (s *GameScene) rightClick(p image.Point) {
	f, ok := s.figureAt(p)
	if !ok || !f.patient {
		return
	}
	if err := s.g.WakeEarly(f.id); err != nil {
		s.say(err.Error())
		s.fx.punch(f.id, colRed)
		s.fx.invalidClick()
		playSound(sfxInvalid)
		return
	}
	s.fx.punch(f.id, colWhite)
}

// say muestra un aviso de la escena (los errores del juego, "Próximamente").
func (s *GameScene) say(msg string) { s.message, s.messageTicks = msg, messageTicks }

// messageTicks: lo que dura un aviso de la escena (3 s).
const messageTicks = 180

// clock escribe el cronómetro de espera como en la maqueta: "0:42".
func clock(d time.Duration) string {
	secs := int(d.Seconds())
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}

// bubbleBlinkAfter: la burbuja roja titila después de 15 s de espera.
const bubbleBlinkAfter = 15 * time.Second

// gameOverlays son los rótulos de un personaje en el juego (§9.1): siempre,
// la burbuja de los dormidos (roja con el cronómetro si nadie lo ha
// recogido o está en el pasillo, y titila después de 15 s; azul en la cama)
// con sus rayitos, y el punto del personal (verde libre, rojo ocupado). La
// etiqueta con el ID, solo si showLabel (el mouse encima o Shift).
func (s *GameScene) gameOverlays(f figure, showLabel bool) []overlay {
	at := s.pos[f.id]
	x, y := int(math.Round(at.x)), int(math.Round(at.y))
	var out []overlay
	if f.patient {
		pv, _ := s.patientView(f.id)
		pa := s.anims[f.id]
		if pa != nil && pa.showsBubble() {
			txt, bg := "Zzz", colBubbleBed
			if pv.Stage == game.Collapsed || pv.Stage == game.InHallway {
				txt, bg = "Zzz! "+clock(pv.Waited), colBubbleHallway
			}
			blinkOff := (pv.Stage == game.Collapsed || pv.Stage == game.InHallway) && pv.Waited > bubbleBlinkAfter && s.ticks/30%2 == 1
			if !blinkOff {
				box := centeredBox(txt, x+figWidth/2, y-20, smallFace)
				if s.onBed(f) {
					bed := bedRect(roomZone(f.room))
					box = textBox(txt, bed.Max.X+4, bed.Min.Y+4, smallFace)
				}
				out = append(out, bubbleOverlays(txt, bg, f.level, box)...)
			}
		}
	} else {
		dot := colGreen
		if st, _ := s.staffView(f.id); st.Activity != game.Free || s.actorOf(f.id).attend > 0 {
			dot = colRed
		}
		out = append(out, overlay{rect: image.Rect(x+figWidth/2-4, y-10, x+figWidth/2+4, y-2), bg: dot})
	}
	if showLabel {
		box := centeredBox(f.id, x+figWidth/2, y+figHeight+2, smallFace)
		if f.patient && s.onBed(f) {
			bed := bedRect(roomZone(f.room))
			box = textBox(f.id, bed.Max.X+4, bed.Min.Y+28, smallFace)
		}
		out = append(out, overlay{rect: box, text: f.id, fg: colInk, bg: colPaper})
	}
	return out
}

// tooltipLines es la ficha de un personaje (§9.1): ID y nombre, nivel y
// estado si es paciente; rol y lo que hace si es del personal.
func (s *GameScene) tooltipLines(f figure) []string {
	if f.patient {
		pv, ok := s.patientView(f.id)
		if !ok {
			return nil
		}
		status := pv.Stage.String()
		if pv.Stage == game.Collapsed || pv.Stage == game.InHallway || pv.Stage == game.AwaitingReview {
			status += " (" + clock(pv.Waited) + ")"
		}
		return []string{pv.ID + " " + pv.Name, "Nivel: " + levelName(pv.Level), "Estado: " + status}
	}
	st, ok := s.staffView(f.id)
	if !ok {
		return nil
	}
	role := fmt.Sprintf("Camillero · rapidez %d", st.Speed)
	if st.IsDoctor {
		role = fmt.Sprintf("Médico · rapidez %d · pericia %d", st.Speed, st.Skill)
	}
	return []string{st.ID + " " + st.Name, role, "Estado: " + st.Activity.String()}
}

// levelName es el nivel de narcolepsia en español, como lo escribe el modelo.
func levelName(l hospital.NarcolepsyLevel) string { return strings.ToLower(l.String()) }

// drawTooltip dibuja la ficha junto al personaje f.
func (s *GameScene) drawTooltip(dst *ebiten.Image, f figure) {
	lines := s.tooltipLines(f)
	if len(lines) == 0 {
		return
	}
	w := 0
	for _, l := range lines {
		w = max(w, int(textWidth(l, smallFace)))
	}
	box := s.figureBox(f)
	r := image.Rect(box.Max.X+8, box.Min.Y, box.Max.X+8+w+16, box.Min.Y+len(lines)*20+10)
	if r.Max.X > ScreenWidth-4 { // no cabe a la derecha: a la izquierda
		r = r.Add(image.Pt(box.Min.X-8-r.Max.X, 0))
	}
	if r.Min.Y < hudHeight+2*hudBorder+4 {
		r = r.Add(image.Pt(0, hudHeight+2*hudBorder+4-r.Min.Y))
	}
	panel(dst, r, colPaper)
	for i, l := range lines {
		c := colInk
		if i > 0 {
			c = colMuted
		}
		drawText(dst, l, float64(r.Min.X+8), float64(r.Min.Y+5+i*20), smallFace, c)
	}
}
