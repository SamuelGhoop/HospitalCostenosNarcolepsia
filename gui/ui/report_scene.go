package ui

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// ReportScene es el Shift Report: las cuatro consultas de la Sección 5 en
// un portapapeles, como el menú de la pantalla de inicio de la maqueta.
type ReportScene struct {
	report game.Report
}

// NewReportScene muestra un Report ya calculado por el paquete game.
func NewReportScene(r game.Report) *ReportScene {
	return &ReportScene{report: r}
}

// Update: Espacio o clic vuelve a empezar la demostración, con un hospital
// nuevo. (El botón "¡AHORA TE TOCA!" se activa cuando exista el modo juego.)
func (s *ReportScene) Update() (Scene, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return NewDemoScene(game.NewDemo()), nil
	}
	return s, nil
}

// Draw dibuja el portapapeles con las cuatro consultas.
func (s *ReportScene) Draw(screen *ebiten.Image) {
	screen.Fill(colSand)

	board := image.Rect(48, 28, 1232, 704)
	fillRect(screen, board.Add(image.Pt(6, 6)), colWoodDark) // sombra
	panel(screen, board, colWood)
	panel(screen, image.Rect(540, 12, 740, 44), colMetal) // gancho
	paper := image.Rect(72, 56, 1208, 680)
	fillRect(screen, paper.Add(image.Pt(4, 4)), colPaperShade)
	panel(screen, paper, colPaper)

	c := column{dst: screen, x: 92, y: 68}
	r := s.report
	c.line("REPORTE DEL TURNO — DEMOSTRACIÓN DE LA SECCIÓN 6", boldFace, colBlue)
	c.gap(8)

	c.line("5.1  Pacientes dormidos en el pasillo", boldFace, colInk)
	if len(r.Hallway) == 0 {
		c.indented("(nadie: todos los dormidos tienen cama)", textFace, colMuted)
	}
	for _, p := range r.Hallway {
		c.indented(fmt.Sprintf("%s %s (%s)", p.ID, p.Name, p.Location), textFace, colInk)
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
			c.indented(fmt.Sprintf("   %s %s  %s %s · %s → %s", e.ID, e.Time, e.PatientID, e.PatientName, e.Location, destination), textFace, colInk)
		}
	}
	c.gap(8)

	c.line("5.3  Habitaciones", boldFace, colInk)
	for _, room := range r.Rooms {
		occupants := "vacía"
		if len(room.Occupants) > 0 {
			occupants = strings.Join(room.Occupants, ", ")
		}
		c.indented(fmt.Sprintf("%d  %-10v %s", room.Number, room.State, occupants), textFace, colInk)
	}
	if r.LastAssignError != "" {
		c.indented("Último error de AssignRoom:", textFace, colMuted)
		c.indented("   "+r.LastAssignError, textFace, colRedDark)
	}
	c.gap(8)

	c.line("5.4  Reporte de severidad (pacientes Severe)", boldFace, colInk)
	for _, sv := range r.Severe {
		c.indented(fmt.Sprintf("%-6s %-18s %d episodio(s) hoy", sv.PatientID, sv.PatientName, sv.Episodes), textFace, colInk)
	}

	// Pie: el botón del modo juego todavía deshabilitado.
	button := image.Rect(92, 612, 372, 648)
	panel(screen, button, colLightGray)
	drawText(screen, "¡AHORA TE TOCA!", float64(button.Min.X+16), float64(button.Min.Y+6), boldFace, colMuted)
	drawText(screen, "(modo juego: próximamente)", float64(button.Max.X+16), float64(button.Min.Y+10), textFace, colMuted)
	hint := "ESPACIO o clic: ver la demostración otra vez"
	drawText(screen, hint, float64(paper.Max.X-20)-textWidth(hint, smallFace), 656, smallFace, colMuted)
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

// gap deja un espacio vertical extra.
func (c *column) gap(px float64) { c.y += px }
