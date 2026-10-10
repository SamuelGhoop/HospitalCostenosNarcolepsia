package ui

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// walkSpeed: cuántos píxeles avanza por tick un personaje que cambia de
// puesto (a 60 ticks por segundo, unos 360 px por segundo en la pantalla
// de 1280×720: la misma rapidez que se veía a 640×360).
const walkSpeed = 6.0

// DemoScene muestra la demostración de la Sección 6 paso a paso.
//
// No decide nada: llama demo.Next() o demo.Skip() y dibuja la foto que
// devuelve demo.Snapshot(). Lo propio es la animación: a dónde camina cada
// personaje (la coreografía del paso) y lo que el usuario ve (HUD,
// subtítulos).
type DemoScene struct {
	demo     *game.Demo
	snap     game.DemoSnapshot
	figures  []figure
	pos      map[string]vec          // posición dibujada de cada personaje, por ID
	paths    map[string][]vec        // lo que le falta de ruta a cada personaje (ver routes.go)
	pathTo   map[string]vec          // el destino para el que se calculó esa ruta
	anims    map[string]*patientAnim // animación de cada paciente, por ID
	choreo   *choreography           // coreografía del último paso; nil si no hay o ya terminó
	carried  string                  // paciente que el último paso pasó del pasillo a una cama
	auto     autoAdvance
	showSubs bool
	scenery  scenery // el taxi y el humo de la olla
}

// NewDemoScene prepara la escena para una demo recién creada.
func NewDemoScene(d *game.Demo) *DemoScene {
	s := &DemoScene{demo: d, pos: map[string]vec{}, paths: map[string][]vec{}, pathTo: map[string]vec{},
		anims: map[string]*patientAnim{}, showSubs: true}
	s.refresh()
	return s
}

// Update lee el teclado y el ratón y avanza la animación.
//
//	Espacio o clic: siguiente paso (o adelantar la coreografía en curso)
//	A: avance automático cada 3 s · S: saltar al final · C: subtítulos
func (s *DemoScene) Update() (Scene, error) {
	s.scenery.update()
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
		s.choreo = nil // al saltar no se reproducen coreografías
	}

	pressed := inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	switch {
	case s.choreo != nil && pressed:
		s.finishChoreography() // adelanta: cada uno queda en su sitio final
	case pressed || (s.choreo == nil && s.auto.tick()): // el automático cuenta cuando termina la coreografía
		if s.demo.Done() {
			return NewReportScene(s.demo.Report()), nil // fin: Shift Report
		}
		if err := s.advance(); err != nil {
			return s, err // error inesperado: el escenario está hecho para no fallar
		}
		s.auto.reset()
	}

	s.animate()
	return s, nil
}

// advance da el siguiente paso de la demo y arma su coreografía.
func (s *DemoScene) advance() error {
	step, err := s.demo.Next()
	if err != nil {
		return err
	}
	s.refresh()
	s.choreo = s.choreographyFor(step)
	return nil
}

// animate hace un tick de animación: cada personaje camina hacia su destino
// (el de la coreografía si lo hay; si no, su puesto final) siguiendo su
// ruta por las puertas, la animación de cada paciente avanza según cuánto
// se movió, y la coreografía pasa a la siguiente fase cuando todos llegaron.
func (s *DemoScene) animate() {
	for _, f := range s.figures {
		old := s.pos[f.id]
		now := moveToward(old, s.nextWaypoint(f.id, old, s.destination(f)), walkSpeed)
		s.pos[f.id] = now
		if a, ok := s.anims[f.id]; ok {
			a.step(now.x-old.x, now.y-old.y, s.onBed(f))
		}
	}

	if s.choreo == nil {
		return
	}
	s.choreo.step(
		func(id string) bool { // ¿ya llegó?
			f, _ := s.figure(id)
			return s.pos[id] == s.destination(f)
		},
		func(id string) bool { // ¿ya está acostado?
			a, ok := s.anims[id]
			return ok && a.showsBubble()
		},
	)
	if s.choreo.done() {
		s.choreo = nil
	}
}

// nextWaypoint es el próximo punto al que camina id para llegar a dest.
// Si dest cambió (otra fase de la coreografía, otro puesto), calcula la
// ruta de nuevo; cuando llega a un punto de paso, sigue con el siguiente.
func (s *DemoScene) nextWaypoint(id string, from, dest vec) vec {
	if s.pathTo[id] != dest || len(s.paths[id]) == 0 {
		s.paths[id], s.pathTo[id] = route(from, dest), dest
	}
	path := s.paths[id]
	for len(path) > 1 && from == path[0] {
		path = path[1:]
	}
	s.paths[id] = path
	return path[0]
}

// destination es a dónde tiene que caminar un personaje ahora mismo.
func (s *DemoScene) destination(f figure) vec {
	if s.choreo != nil {
		if to, ok := s.choreo.target(f.id); ok {
			return to
		}
	}
	return f.target
}

// onBed dice si un paciente ya está SOBRE su cama (no solo asignada: la
// cama se asigna en el modelo antes de que lo terminen de llevar).
func (s *DemoScene) onBed(f figure) bool {
	return f.room != 0 && s.pos[f.id] == bedCell(roomZone(f.room))
}

// figure busca un personaje por ID.
func (s *DemoScene) figure(id string) (figure, bool) {
	for _, f := range s.figures {
		if f.id == id {
			return f, true
		}
	}
	return figure{}, false
}

// finishChoreography adelanta la coreografía en curso: cada personaje
// queda de una vez en su puesto final, con su animación de reposo.
func (s *DemoScene) finishChoreography() {
	s.choreo = nil
	for _, f := range s.figures {
		s.pos[f.id] = f.target
		delete(s.paths, f.id) // saltó a su puesto: la ruta vieja ya no sirve
		if a, ok := s.anims[f.id]; ok {
			a.settle(s.onBed(f))
		}
	}
}

// choreographyFor arma la coreografía del paso que se acaba de dar:
//
//   - pasos 5–8 (crearon un episodio): quien lo atendió, según el modelo,
//     camina hasta el paciente y lo lleva a su cama, o lo acompaña si no hubo;
//   - paso 11 (un paciente pasó del pasillo a una cama): lo lleva el camillero.
func (s *DemoScene) choreographyFor(step game.DemoStep) *choreography {
	if e := step.Episode; e != nil {
		patient, okP := s.figure(e.PatientID)
		attender, okA := s.figure(e.AttendedByID)
		if !okP || !okA {
			return nil
		}
		attackAt := patient.target // sin cama: el puesto del piso donde se va a quedar
		if e.Room != 0 {
			attackAt = toVec(floorSpot(zoneFor(e.Location), 0)) // con cama: se desploma donde le dio el ataque
		}
		return attendChoreography(patient.id, attender.id, attackAt, attender.target, roomZone(e.Room), e.Room != 0)
	}

	if s.carried != "" {
		patient, _ := s.figure(s.carried)
		for _, f := range s.figures {
			if !f.patient && !f.doctor { // el camillero
				return carryChoreography(patient.id, f.id, s.pos[patient.id], f.target, roomZone(patient.room))
			}
		}
	}
	return nil
}

// refresh pide una foto nueva a la demo y recalcula el puesto FINAL de
// cada personaje:
//
//   - con cama: acostado en su cama;
//   - dormido sin cama: en un puesto del piso, en la parte de abajo de su zona;
//   - despierto en una habitación: de pie a la izquierda de la cama;
//   - despierto en otra zona: en la fila de arriba, uno al lado del otro.
func (s *DemoScene) refresh() {
	s.snap = s.demo.Snapshot()
	s.figures = s.figures[:0]
	s.carried = ""
	standing := map[zone]int{} // puestos de pie usados en cada zona
	lying := map[zone]int{}    // puestos de piso usados en cada zona

	for _, p := range s.snap.Patients {
		f := figure{id: p.ID, patient: true, state: p.State, level: p.Level, room: p.Room}
		z := zoneFor(p.Location)
		_, inRoomZone := roomMockupX[z] // ¿la zona es una habitación?
		switch {
		case p.Room != 0:
			f.target = bedCell(roomZone(p.Room))
		case p.State != hospital.Awake:
			f.target = toVec(floorSpot(z, lying[z]))
			lying[z]++
		case inRoomZone && standing[z] == 0:
			f.target = toVec(wakeSpot(z)) // se despertó en su habitación
			standing[z]++
		default:
			f.target = toVec(slot(z, standing[z]))
			standing[z]++
		}

		a, seen := s.anims[p.ID]
		if !seen {
			a = &patientAnim{state: p.State, room: p.Room}
			s.anims[p.ID] = a
			s.pos[p.ID] = toVec(entrance) // los pacientes nuevos entran desde la acera
		}
		was := a.state
		if leftBed := a.observe(p.State, p.Room); leftBed {
			s.pos[p.ID] = f.target // se levanta y queda de una al lado de la cama, sin deslizarse
			delete(s.paths, p.ID)
		}
		if was == hospital.AsleepInHallway && p.State == hospital.AsleepInBed {
			s.carried = p.ID // pasó del pasillo a una cama (paso 11)
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

// Draw dibuja por capas: mapa (con las camas libres) → taxi y humo →
// personajes → cobijas → rótulos (letreros, burbujas, etiquetas) → HUD y
// subtítulos.
func (s *DemoScene) Draw(screen *ebiten.Image) {
	drawMap(screen)
	s.scenery.draw(screen)

	lyingIn := map[int]bool{} // habitaciones con alguien ya acostado en la cama
	for _, f := range s.figures {
		drawFigureBody(screen, f, s.pos[f.id], s.anims[f.id]) // anims[id] es nil para el personal
		if f.patient && s.onBed(f) {
			lyingIn[f.room] = true
		}
	}
	drawBlankets(screen, s.snap.Rooms, lyingIn)

	for _, o := range roomBadges(s.snap.Rooms) {
		drawOverlay(screen, o)
	}
	for _, f := range s.figures {
		showBubble := false
		if a, ok := s.anims[f.id]; ok {
			showBubble = a.showsBubble()
		}
		for _, o := range figureOverlays(f, s.pos[f.id], showBubble) {
			drawOverlay(screen, o)
		}
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
	drawText(dst, title, 12, 6, boldFace, colYellow)
	if s.auto.on {
		drawText(dst, "[AUTOMÁTICO]", 24+textWidth(title, boldFace), 10, smallFace, colGreen)
	}

	hint := "ESPACIO siguiente · A automático · S saltar · C subtítulos"
	drawText(dst, hint, ScreenWidth-12-textWidth(hint, smallFace), 10, smallFace, colLightGray)
}

// drawSubtitles: la barra de abajo con lo que pasó en el último paso, la
// llamada real al modelo y lo que respondió (en rojo si fue un error).
func (s *DemoScene) drawSubtitles(dst *ebiten.Image) {
	box := image.Rect(12, 594, ScreenWidth-12, ScreenHeight-10)
	panel(dst, box, colPaper)
	x := float64(box.Min.X + 12)
	top := float64(box.Min.Y)

	step := s.snap.Last
	if step.Number == 0 {
		drawText(dst, "Demostración del escenario obligatorio de la Sección 6 del enunciado.", x, top+10, boldFace, colInk)
		drawText(dst, "Cada paso hace la llamada REAL al modelo (paquete hospital) y muestra lo que respondió.", x, top+44, textFace, colMuted)
		drawText(dst, "Presiona ESPACIO o haz clic para dar el primer paso.", x, top+72, textFace, colBlueDark)
		return
	}

	drawText(dst, fmt.Sprintf("%d. %s", step.Number, step.Subtitle), x, top+8, boldFace, colInk)
	drawText(dst, step.Call, x, top+42, textFace, colBlueDark)
	result := colGreenDark
	if step.Err != nil {
		result = colRedDark // el error previsto del paso 8
	}
	drawText(dst, "→ "+step.Result, x, top+72, textFace, result)

	if s.demo.Done() {
		hint := "ESPACIO: ver el Shift Report"
		drawText(dst, hint, float64(box.Max.X-12)-textWidth(hint, smallFace), top+12, smallFace, colMuted)
	}
}
