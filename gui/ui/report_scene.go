package ui

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// ReportScene es el Shift Report: las cuatro consultas de la Sección 5 en
// un portapapeles, como el menú de la pantalla de inicio de la maqueta.
//
// Desde el 2026-10-17 (§10.10, "hazle así animados los menús") abre y
// cierra con la misma animación de portapapeles que la pausa y la bolsa
// de empleo: en el modo juego desde el botón REPORTE del HUD, en la demo
// desde el centro de la pantalla (que no tiene HUD). Mientras anima no
// acepta Espacio ni clic, y la vuelta a la portada o a la partida se hace
// al terminar de cerrar, no al apretar la tecla.
type ReportScene struct {
	report game.Report
	title  string // "" = el de la demo
	back   Scene  // a dónde vuelve; nil = la portada (la demo)
	anim   *animPanel
}

// Tiempos y centro abierto del Shift Report: los mismos de la pausa y los
// paneles del juego (panelOpenTicks/panelCloseTicks, game_scene.go), con
// su propio nombre porque cada escena guarda los suyos.
const (
	reportOpenTicks  = 13 // 0,22 s
	reportCloseTicks = 11 // 0,18 s
	reportMinScale   = 0.167
)

// reportOpenCenter es el centro del portapapeles abierto (el board de
// Draw: 48..1232 × 28..704).
var reportOpenCenter = vec{640, 366}

// reportAnimFrom: desde el botón REPORTE del HUD si vuelve al modo juego
// (back no es nil); desde el centro de la pantalla en la demo, igual que
// pauseAnimFrom.
func reportAnimFrom(back Scene) vec {
	if back != nil {
		c := reportButton.Min.Add(reportButton.Max).Div(2)
		return vec{float64(c.X), float64(c.Y)}
	}
	return vec{640, 360}
}

// NewReportScene muestra un Report ya calculado por el paquete game (la
// demo y el backdoor de pruebas: no vuelven a ningún lado más que a la
// portada).
func NewReportScene(r game.Report) *ReportScene {
	return newReportScene(r, "", nil)
}

// newReportScene arma el reporte con back (nil en la demo) y arranca
// abriendo desde reportAnimFrom.
func newReportScene(r game.Report, title string, back Scene) *ReportScene {
	return &ReportScene{report: r, title: title, back: back,
		anim: newAnimPanel(reportOpenTicks, reportCloseTicks, reportAnimFrom(back), reportOpenCenter, reportMinScale)}
}

// Update: Espacio o clic arrancan el cierre; al terminar, vuelve a la
// portada o, en el modo juego, a la partida (que sigue: el reporte la
// tenía en pausa).
func (s *ReportScene) Update() (Scene, error) { return s.update(readInput()) }

func (s *ReportScene) update(in input) (Scene, error) {
	if !s.anim.acceptsInput() { // abriendo o cerrando: no se le hace caso a Espacio ni al clic
		if s.anim.step() { // terminó de cerrar: ahí es cuando se vuelve de verdad
			if s.back == nil {
				return NewTitleScene(), nil
			}
			if p, ok := s.back.(pausable); ok {
				p.onResume()
			}
			return s.back, nil
		}
		return s, nil
	}
	if in.space || in.click {
		s.anim.startClose()
	}
	return s, nil
}

// Draw dibuja, debajo, la partida en pausa (o un fondo liso en la demo, que
// no tiene nada detrás) y, encima, el portapapeles con las cuatro
// consultas, oscurecido y escalado según la animación (drawAnimatedPanel,
// anim.go).
func (s *ReportScene) Draw(screen *ebiten.Image) {
	if s.back != nil {
		s.back.Draw(screen)
	} else {
		screen.Fill(colSand)
	}
	drawAnimatedPanel(screen, s.anim, pauseDim.A, s.drawPanel)
}

// drawPanel dibuja el contenido del portapapeles en su sitio final; lo
// escala y lo centra drawAnimatedPanel.
func (s *ReportScene) drawPanel(dst *ebiten.Image) {
	board := image.Rect(48, 28, 1232, 704)
	fillRect(dst, board.Add(image.Pt(6, 6)), colWoodDark) // sombra
	panel(dst, board, colWood)
	panel(dst, image.Rect(540, 12, 740, 44), colMetal) // gancho
	paper := image.Rect(72, 56, 1208, 680)
	fillRect(dst, paper.Add(image.Pt(4, 4)), colPaperShade)
	panel(dst, paper, colPaper)

	c := column{dst: dst, x: 92, y: 68}
	const paperRight = 1208 // paper.Max.X: hasta ahí llega una fila antes de recortarse
	r := s.report
	title := s.title
	if title == "" {
		title = "REPORTE DEL TURNO — DEMOSTRACIÓN DE LA SECCIÓN 6"
	}
	c.line(title, boldFace, colBlue)
	c.gap(8)

	c.line("5.1  Pacientes dormidos en el pasillo", boldFace, colInk)
	if len(r.Hallway) == 0 {
		c.indented("(nadie: todos los dormidos tienen cama)", textFace, colMuted)
	}
	for _, p := range r.Hallway {
		c.indentedFit(fmt.Sprintf("%s %s (%s)", p.ID, p.Name, p.Location), textFace, colInk, paperRight)
	}
	c.gap(8)

	c.line("5.2  Episodios atendidos por cada médico", boldFace, colInk)
	for _, d := range r.Episodes {
		c.indented(d.DoctorName+":", textFace, colBlueDark)
		if len(d.Episodes) == 0 {
			c.indented("   (ninguno)", textFace, colMuted)
		}
		for _, e := range d.Episodes {
			destination := "pasillo (sin cama)"
			if e.Room != 0 {
				destination = fmt.Sprintf("habitación %d", e.Room)
			}
			c.indentedFit(fmt.Sprintf("   %s %s  %s %s · %s → %s", e.ID, e.Time, e.PatientID, e.PatientName, e.Location, destination), textFace, colInk, paperRight)
		}
	}
	c.gap(8)

	c.line("5.3  Habitaciones", boldFace, colInk)
	for _, room := range r.Rooms {
		occupants := "vacía"
		if len(room.Occupants) > 0 {
			occupants = strings.Join(room.Occupants, ", ")
		}
		c.indentedFit(fmt.Sprintf("%d  %-10v %s", room.Number, room.State, occupants), textFace, colInk, paperRight)
	}
	if r.LastAssignError != "" {
		c.indented("Último error de AssignRoom:", textFace, colMuted)
		c.indented("   "+r.LastAssignError, textFace, colRedDark)
	}
	c.gap(8)

	c.line("5.4  Reporte de severidad (pacientes Severe)", boldFace, colInk)
	for _, sv := range r.Severe {
		c.indentedFit(fmt.Sprintf("%-6s %-18s %d episodio(s) hoy", sv.PatientID, sv.PatientName, sv.Episodes), textFace, colInk, paperRight)
	}

	// Pie: el botón del modo juego todavía deshabilitado.
	button := image.Rect(92, 612, 372, 648)
	panel(dst, button, colLightGray)
	drawText(dst, "¡AHORA TE TOCA!", float64(button.Min.X+16), float64(button.Min.Y+6), boldFace, colMuted)
	drawText(dst, "(modo juego: próximamente)", float64(button.Max.X+16), float64(button.Min.Y+10), textFace, colMuted)
	hint := "ESPACIO o clic: volver a la portada"
	drawText(dst, hint, float64(paper.Max.X-20)-textWidth(hint, smallFace), 656, smallFace, colMuted)
}

// column escribe líneas de texto una debajo de otra.
type column struct {
	dst  *ebiten.Image
	x, y float64
}

// line escribe una línea y baja el cursor.
func (c *column) line(s string, face *text.GoTextFace, clr color.Color) {
	drawText(c.dst, s, c.x, c.y, face, clr)
	m := face.Metrics()
	c.y += m.HAscent + m.HDescent + 2
}

// indented escribe una línea con sangría.
func (c *column) indented(s string, face *text.GoTextFace, clr color.Color) {
	c.x += 20
	c.line(s, face, clr)
	c.x -= 20
}

// indentedFit escribe una línea con sangría, recortada con "…"
// (truncateWithEllipsis) si no cabe entre su x y right: con textFace a
// 16 px, las filas largas del reporte (episodios, habitaciones,
// severidad) ya no entran siempre enteras, y acá no hay cómo partirlas en
// dos renglones sin desordenar la columna.
func (c *column) indentedFit(s string, face *text.GoTextFace, clr color.Color, right int) {
	avail := right - int(c.x) - 20
	c.indented(truncateWithEllipsis(s, avail, face.Size), face, clr)
}

// gap deja un espacio vertical extra.
func (c *column) gap(px float64) { c.y += px }
