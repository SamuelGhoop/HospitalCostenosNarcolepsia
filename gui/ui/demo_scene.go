package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"slices"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// Cuántos píxeles avanza por tick un personaje que camina (a 60 ticks por
// segundo): el personal a 120 px/s y los pacientes a 90 px/s (decisión de
// Samuel, 2026-10-11). La posición es float y se redondea al dibujar.
const (
	staffSpeed   = 2.0
	patientSpeed = 1.5
)

// speedOf es la velocidad a la que camina un personaje.
func speedOf(f figure) float64 {
	if f.patient {
		return patientSpeed
	}
	return staffSpeed
}

// DemoScene muestra la demostración de la Sección 6 paso a paso.
//
// No decide nada: llama demo.Next() o demo.Skip() y dibuja la foto que
// devuelve demo.Snapshot(). Lo propio es la animación: a dónde camina cada
// personaje (la coreografía del paso) y lo que el usuario ve (HUD,
// subtítulos).
type DemoScene struct {
	world    // dónde está y por dónde camina cada personaje (world.go)
	demo     *game.Demo
	snap     game.DemoSnapshot
	choreo   *choreography // coreografía del último paso; nil si no hay o ya terminó
	carried  string        // paciente que el último paso pasó del pasillo a una cama
	arrivals []string      // pacientes que llegaron en el último paso, en orden
	auto     autoAdvance
	showSubs bool
	scenery  scenery  // los carros de la calle y el humo de la olla
	door     mainDoor // la puerta principal, que se abre y se cierra suave
}

// NewDemoScene prepara la escena para una demo recién creada.
func NewDemoScene(d *game.Demo) *DemoScene {
	s := &DemoScene{world: newWorld(), demo: d, showSubs: true,
		scenery: newScenery(rand.New(rand.NewSource(seedSource())))} // el tráfico es decorado: cada vez distinto (o fijo, con -agent -seed)
	s.refresh()
	return s
}

// Update lee el teclado y el ratón y avanza la animación.
//
//	Espacio o clic: siguiente paso (o adelantar la coreografía en curso)
//	A: avance automático cada 3 s · S: saltar al final · C: subtítulos
//	(Esc abre la pausa, pero eso lo atiende la App: ver app.go)
func (s *DemoScene) Update() (Scene, error) {
	s.scenery.update()
	if keyJustPressed(ebiten.KeyC) {
		s.showSubs = !s.showSubs
	}
	if keyJustPressed(ebiten.KeyA) {
		s.auto.toggle()
	}
	if keyJustPressed(ebiten.KeyS) {
		if err := s.demo.Skip(); err != nil {
			return s, err
		}
		s.refresh()
		s.choreo = nil // al saltar no se reproducen coreografías
	}

	pressed := keyJustPressed(ebiten.KeySpace) || clickJustPressed()
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

// La demo se puede pausar con Esc (cumple pausable). No tiene que hacer
// nada: mientras está la pausa no recibe Update, así que todo se queda
// quieto (el avance automático, las animaciones, los carros y el humo) y al
// reanudar sigue exactamente donde iba.
func (s *DemoScene) onPause()  {}
func (s *DemoScene) onResume() {}
func (s *DemoScene) onQuit()   {}

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
	s.beginTick() // congela las posiciones de arranque: la reserva de celdas no depende del orden de la lista
	// El traslado en camilla de la fase actual, si hay: lo mueve stepCarry.
	var order *carryOrder
	if s.choreo != nil {
		order = s.choreo.carrying()
	}
	switch {
	case order == nil:
		s.carry, s.dodge = nil, nil // la camilla ya pasó: vuelven a su puesto
	case s.carry == nil || s.carry.order != order:
		s.carry = &carrying{order: order}
		s.dodge = s.dodgesFor(order)
	}
	before := map[string]vec{}
	for id, p := range s.pos {
		before[id] = p
	}
	if s.carry != nil {
		s.stepCarry(s.carry)
	}

	for _, f := range s.figures {
		old := before[f.id]
		a, isPatient := s.anims[f.id]
		if !s.inCarry(f.id) && !(isPatient && a.holdsStill()) { // despertándose se queda donde está
			s.pos[f.id] = moveToward(old, s.nextWaypoint(f.id, old, s.destination(f)), speedOf(f))
		}
		now := s.pos[f.id]
		if isPatient {
			a.step(now.x-old.x, now.y-old.y, s.onBed(f))
		}
		if a, ok := s.staff[f.id]; ok {
			c := s.carryOf(f.id)
			// Solo el camillero tiene fila de empujar; un médico que empuja
			// (no había camillero libre) sigue con la animación de caminar.
			pushing := !f.doctor && c != nil && c.visible() && c.order.pusher == f.id
			a.step(now.x-old.x, now.y-old.y, s.choreo != nil && s.choreo.attending() == f.id, pushing)
		}
	}

	s.door.step(s.someoneNearDoor())

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

// destination es a dónde tiene que caminar un personaje ahora mismo.
func (s *DemoScene) destination(f figure) vec {
	if to, ok := s.dodge[f.id]; ok {
		return to // se hizo a un lado para dejar pasar la camilla
	}
	if s.choreo != nil {
		if to, ok := s.choreo.target(f.id); ok {
			return to
		}
	}
	return f.target
}

// finishChoreography adelanta la coreografía en curso: cada personaje
// queda de una vez en su puesto final, con su animación de reposo.
func (s *DemoScene) finishChoreography() {
	s.choreo, s.carry, s.dodge = nil, nil, nil
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
		hold := 0
		if attender.doctor {
			hold = attendTicks // el médico atiende antes de llevárselo
		}
		return attendChoreography(patient.id, attender.id, attackAt, attender.target, roomZone(e.Room), e.Room != 0, hold, s.stillFeet(patient.id, attender.id))
	}

	if len(s.arrivals) > 0 { // paso 3: entran desde la acera, uno cada 1,5 s
		targets := map[string]vec{}
		for _, id := range s.arrivals {
			f, _ := s.figure(id)
			targets[id] = f.target
		}
		return arrivalChoreography(s.arrivals, targets, arrivalSpawn())
	}

	if s.carried != "" {
		patient, _ := s.figure(s.carried)
		for _, f := range s.figures {
			if !f.patient && !f.doctor { // el camillero
				return carryChoreography(patient.id, f.id, s.pos[patient.id], f.target, roomZone(patient.room), s.stillFeet(patient.id, f.id))
			}
		}
	}
	return nil
}

// stillFeet son las huellas de los pies de los que se quedan quietos en su
// puesto mientras se hace un traslado (todos menos except): la camilla no
// los pisa. Los que duermen en su cama no cuentan.
func (s *DemoScene) stillFeet(except ...string) []image.Rectangle {
	var out []image.Rectangle
	for _, f := range s.figures {
		if slices.Contains(except, f.id) || (f.patient && f.room != 0 && f.target == bedCell(roomZone(f.room))) {
			continue
		}
		feet := f.target.add(feetOffset)
		out = append(out, feetBox.Add(image.Pt(int(math.Round(feet.x)), int(math.Round(feet.y)))))
	}
	return out
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
	s.carried, s.arrivals = "", nil
	standing := map[zone]int{} // puestos de pie usados en cada zona
	lying := map[zone]int{}    // puestos de piso usados en cada zona

	for _, p := range s.snap.Patients {
		f := figure{id: p.ID, patient: true, state: p.State, level: p.Level, room: p.Room, look: demoLook(p.ID)}
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
			s.pos[p.ID] = arrivalSpawn() // los pacientes nuevos llegan por la acera, desde fuera de la pantalla
			s.arrivals = append(s.arrivals, p.ID)
		}
		was := a.state
		a.observe(p.State, p.Room) // si se despertó en la cama, se levanta ahí y camina al costado
		if was == hospital.AsleepInHallway && p.State == hospital.AsleepInBed {
			s.carried = p.ID // pasó del pasillo a una cama (paso 11)
		}
		s.figures = append(s.figures, f)
	}

	for i, st := range s.snap.Staff {
		f := figure{id: st.ID, doctor: st.IsDoctor, target: toVec(slot(zoneStaffRoom, i)), look: demoLook(st.ID)}
		if _, seen := s.pos[st.ID]; !seen {
			s.pos[st.ID] = f.target // el personal aparece directo en su sala
			s.staff[st.ID] = &staffAnim{}
		}
		s.figures = append(s.figures, f)
	}
}

// Draw dibuja por capas: mapa (con las camas libres) → carros y humo →
// personajes → copas de las palmeras → cobijas → rótulos (letreros,
// burbujas, etiquetas) → HUD y subtítulos.
func (s *DemoScene) Draw(screen *ebiten.Image) {
	drawMap(screen)
	drawMainDoor(screen, s.door.open)
	s.scenery.draw(screen)

	// Los personajes y la camilla, de atrás para adelante: el que tiene los
	// pies más abajo se dibuja encima (así quien empuja queda detrás de la
	// camilla al subir y delante al bajar).
	type drawable struct {
		y    float64
		draw func()
	}
	var items []drawable
	lyingIn := map[int]bool{} // habitaciones con alguien ya acostado en la cama
	c := s.carry
	for _, f := range s.figures {
		if f.patient && s.onBed(f) {
			lyingIn[f.room] = true
		}
		y := s.pos[f.id].y + figHeight
		if c != nil && c.visible() && f.id == c.order.patient {
			if c.riding() {
				continue // va sobre la camilla: se dibuja con ella
			}
			y = float64(c.fp.Max.Y) + 0.5 // pasándose a la camilla o a la cama: encima de la camilla
		}
		items = append(items, drawable{y, func() {
			drawFigureBody(screen, f, s.pos[f.id], s.anims[f.id], s.staff[f.id]) // nil si no es paciente o médico
		}})
	}
	if c != nil && c.visible() {
		items = append(items, drawable{float64(c.fp.Max.Y), func() {
			p, _ := s.figure(c.order.patient)
			drawStretcher(screen, c.fp, c.dir)
			if c.riding() {
				drawRider(screen, p, s.anims[p.id], c.fp, c.dir)
			}
		}})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].y < items[j].y })
	for _, it := range items {
		it.draw()
	}
	drawPalmCrowns(screen)
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

	hint := "ESPACIO siguiente · A automático · S saltar · C subtítulos · Esc pausa"
	drawText(dst, hint, ScreenWidth-12-textWidth(hint, smallFace), 10, smallFace, colLightGray)
}

// demoSubtitleBox es la caja de la barra de abajo de la demo: la usan el
// dibujo y el test que comprueba que la llamada y el resultado de cada
// paso (los textos más largos de toda la interfaz) caben en su ancho, ya
// partidos en dos renglones (drawSubtitleLines). Creció hacia arriba desde
// la medida de antes del 2026-10-17 (y 594→710) para dejar sitio a textFace
// a 16 px en dos renglones por bloque, sin tapar más mapa del necesario.
func demoSubtitleBox() image.Rectangle {
	return image.Rect(12, 520, ScreenWidth-12, ScreenHeight-10)
}

// demoSubtitleBoxWidth: el ancho útil de la barra de subtítulos de la demo
// (drawSubtitles), de x hasta el borde derecho de la caja.
func demoSubtitleBoxWidth() int {
	box := demoSubtitleBox()
	return box.Max.X - 12 - (box.Min.X + 12)
}

// demoSubtitleLineGap: cuánto baja el segundo renglón de un bloque (llamada
// o resultado) respecto del primero.
const demoSubtitleLineGap = 20

// drawSubtitleLines dibuja s partido en hasta dos renglones (wrapTwoLines)
// con la esquina de arriba a la izquierda en (x, y) y devuelve el y donde
// seguiría el próximo bloque.
func drawSubtitleLines(dst *ebiten.Image, s string, x, y float64, maxWidth int, face *text.GoTextFace, c color.Color) float64 {
	lines := wrapTwoLines(s, maxWidth, face.Size)
	for i, line := range lines {
		drawText(dst, line, x, y+float64(i)*demoSubtitleLineGap, face, c)
	}
	return y + float64(len(lines))*demoSubtitleLineGap
}

// drawSubtitles: la barra de abajo con lo que pasó en el último paso, la
// llamada real al modelo y lo que respondió (en rojo si fue un error).
func (s *DemoScene) drawSubtitles(dst *ebiten.Image) {
	box := demoSubtitleBox()
	panel(dst, box, colPaper)
	x := float64(box.Min.X + 12)
	top := float64(box.Min.Y)
	maxWidth := demoSubtitleBoxWidth()

	step := s.snap.Last
	if step.Number == 0 {
		drawText(dst, "Demostración del escenario obligatorio de la Sección 6 del enunciado.", x, top+10, boldFace, colInk)
		y := drawSubtitleLines(dst, "Cada paso hace la llamada REAL al modelo (paquete hospital) y muestra lo que respondió.", x, top+44, maxWidth, textFace, colMuted)
		drawSubtitleLines(dst, "Presiona ESPACIO o haz clic para dar el primer paso.", x, y+10, maxWidth, textFace, colBlueDark)
		return
	}

	y := top + 8
	drawText(dst, fmt.Sprintf("%d. %s", step.Number, step.Subtitle), x, y, boldFace, colInk)
	y += 32
	y = drawSubtitleLines(dst, step.Call, x, y, maxWidth, textFace, colBlueDark)
	result := colGreenDark
	if step.Err != nil {
		result = colRedDark // el error previsto del paso 8
	}
	drawSubtitleLines(dst, "→ "+step.Result, x, y+10, maxWidth, textFace, result)

	if s.demo.Done() {
		hint := "ESPACIO: ver el Shift Report"
		drawText(dst, hint, float64(box.Max.X-12)-textWidth(hint, smallFace), top+12, smallFace, colMuted)
	}
}
