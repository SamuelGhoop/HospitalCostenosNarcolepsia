package ui

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// walkSpeed: cuántos píxeles avanza por tick un personaje que cambia de
// puesto (a 60 ticks por segundo, unos 180 px por segundo).
const walkSpeed = 3.0

// DemoScene muestra la demostración de la Sección 6 paso a paso.
//
// No decide nada: llama demo.Next() o demo.Skip() y dibuja la foto que
// devuelve demo.Snapshot(). Lo único propio es la animación (a dónde se
// desliza cada personaje) y lo que el usuario ve (subtítulos, HUD).
type DemoScene struct {
	demo     *game.Demo
	snap     game.DemoSnapshot
	figures  []figure
	pos      map[string]vec // posición dibujada de cada personaje, por ID
	auto     autoAdvance
	showSubs bool
}

// NewDemoScene prepara la escena para una demo recién creada.
func NewDemoScene(d *game.Demo) *DemoScene {
	s := &DemoScene{demo: d, pos: map[string]vec{}, showSubs: true}
	s.refresh()
	return s
}

// Update lee el teclado y el ratón y avanza la animación.
//
//	Espacio o clic: siguiente paso · A: avance automático cada 3 s
//	S: saltar al final · C: mostrar u ocultar los subtítulos
func (s *DemoScene) Update() (Scene, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		s.showSubs = !s.showSubs
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		s.auto.toggle()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if err := s.demo.Skip(); err != nil {
			return s, err
		}
		s.refresh()
	}

	advance := inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		s.auto.tick()
	if advance {
		if s.demo.Done() {
			return NewReportScene(s.demo.Report()), nil // fin: Shift Report
		}
		if _, err := s.demo.Next(); err != nil {
			return s, err // error inesperado: el escenario está hecho para no fallar
		}
		s.auto.reset()
		s.refresh()
	}

	// Cada personaje se desliza un poco hacia su puesto en cada tick.
	for _, f := range s.figures {
		s.pos[f.id] = moveToward(s.pos[f.id], f.target, walkSpeed)
	}
	return s, nil
}

// refresh pide una foto nueva a la demo y recalcula el puesto de cada
// personaje: en su cama si tiene habitación; si no, en la zona de su
// ubicación (zoneFor), uno al lado del otro.
func (s *DemoScene) refresh() {
	s.snap = s.demo.Snapshot()
	s.figures = s.figures[:0]
	used := map[zone]int{} // cuántos puestos van ocupados en cada zona

	for i, p := range s.snap.Patients {
		f := figure{id: p.ID, patient: true, state: p.State, level: p.Level, shirt: shirts[i%len(shirts)]}
		if p.Room != 0 {
			f.target = bedCell(roomZone(p.Room))
		} else {
			z := zoneFor(p.Location)
			f.target = toVec(slot(z, used[z]))
			used[z]++
		}
		if _, seen := s.pos[p.ID]; !seen {
			s.pos[p.ID] = toVec(entrance) // los pacientes nuevos entran desde la acera
		}
		s.figures = append(s.figures, f)
	}

	for i, st := range s.snap.Staff {
		f := figure{id: st.ID, doctor: st.IsDoctor, target: toVec(slot(zoneStaffRoom, i))}
		if _, seen := s.pos[st.ID]; !seen {
			s.pos[st.ID] = f.target // el personal aparece directo en su sala
		}
		s.figures = append(s.figures, f)
	}
}

// Draw dibuja el mapa, las camas, los personajes, el HUD y los subtítulos.
func (s *DemoScene) Draw(screen *ebiten.Image) {
	drawMap(screen)
	drawRooms(screen, s.snap.Rooms)
	for _, f := range s.figures {
		drawFigure(screen, f, s.pos[f.id])
	}
	s.drawHUD(screen)
	if s.showSubs {
		s.drawSubtitles(screen)
	}
}

// drawHUD: franja de arriba con el paso actual y los controles.
func (s *DemoScene) drawHUD(dst *ebiten.Image) {
	fillRect(dst, image.Rect(0, 0, ScreenWidth, buildingRect.Min.Y-1), colInk)

	title := fmt.Sprintf("MODO DEMOSTRACIÓN — PASO %d/%d", s.snap.Step, s.snap.TotalSteps)
	drawText(dst, title, 6, 3, boldFace, colYellow)
	if s.auto.on {
		drawText(dst, "[AUTOMÁTICO]", 12+textWidth(title, boldFace), 5, smallFace, colGreen)
	}

	hint := "ESPACIO siguiente · A automático · S saltar · C subtítulos"
	drawText(dst, hint, ScreenWidth-6-textWidth(hint, smallFace), 5, smallFace, colLightGray)
}

// drawSubtitles: la barra de abajo con lo que pasó en el último paso, la
// llamada real al modelo y lo que respondió (en rojo si fue un error).
func (s *DemoScene) drawSubtitles(dst *ebiten.Image) {
	box := image.Rect(6, 297, ScreenWidth-6, ScreenHeight-5)
	panel(dst, box, colPaper)
	x := float64(box.Min.X + 6)
	top := float64(box.Min.Y)

	step := s.snap.Last
	if step.Number == 0 {
		drawText(dst, "Demostración del escenario obligatorio de la Sección 6 del enunciado.", x, top+5, boldFace, colInk)
		drawText(dst, "Cada paso hace la llamada REAL al modelo (paquete hospital) y muestra lo que respondió.", x, top+22, textFace, colMuted)
		drawText(dst, "Presiona ESPACIO o haz clic para dar el primer paso.", x, top+36, textFace, colBlueDark)
		return
	}

	drawText(dst, fmt.Sprintf("%d. %s", step.Number, step.Subtitle), x, top+4, boldFace, colInk)
	drawText(dst, step.Call, x, top+21, textFace, colBlueDark)
	result := colGreenDark
	if step.Err != nil {
		result = colRedDark // el error previsto del paso 8
	}
	drawText(dst, "→ "+step.Result, x, top+36, textFace, result)

	if s.demo.Done() {
		hint := "ESPACIO: ver el Shift Report"
		drawText(dst, hint, float64(box.Max.X-6)-textWidth(hint, smallFace), top+6, smallFace, colMuted)
	}
}
