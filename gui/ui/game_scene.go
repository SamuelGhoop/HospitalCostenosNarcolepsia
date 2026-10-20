package ui

import (
	"context"
	"fmt"
	"image"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// GameScene es el modo juego (F3b, GAME_DESIGN §5 y §9.1). La partida vive
// en game/: la goroutine motor (Game.Start) llama Tick cada 100 ms, y esta
// escena, en la goroutine de Ebitengine, lee la foto (Snapshot) en cada
// tick y llama las acciones del jugador (Dispatch, WakeEarly). Las dos
// tocan la partida a la vez; el candado de Game lo hace seguro (§11).
type GameScene struct {
	world // dónde está y por dónde camina cada personaje (world.go)

	g      *game.Game
	cancel context.CancelFunc // detiene el motor (SALIR AL MENÚ)
	done   <-chan struct{}    // se cierra cuando el motor terminó
	snap   game.Snapshot

	actors          map[string]*actor // lo que la escena recuerda de cada uno (game_world.go)
	standing, lying spotBook          // quién ocupa cada puesto de pie y de piso

	hud     hud
	scenery scenery
	door    mainDoor
	wander  *rand.Rand // azar de interfaz del personal libre (staff_rest.go)

	selected     string // paciente elegido para mandarle a alguien ("" = ninguno)
	hover        string // personaje bajo el mouse
	shift        bool   // Shift apretado: todas las etiquetas
	message      string // aviso de la escena (errores, "Próximamente")
	messageTicks int
	noticeSeen   map[string]int // avisos del juego → tick en que aparecieron
	ticks        int

	// sound compara la foto de antes con la de ahora y toca lo que
	// corresponda (sound_events.go, pedido de Samuel el 2026-10-17: "añade
	// sonidos acordes, que la vida está ahí").
	sound soundTracker

	// Los paneles del fin del día y del Game Over abren y cierran animados,
	// como la bolsa de empleo (anim.go, pedido de Samuel el 2026-10-17):
	// nil mientras no hace falta ninguno; se crean la primera vez que
	// snap.DayOver o snap.GameOver es verdad y se vuelven nil cuando
	// terminan de cerrar (así, al día siguiente o en una partida nueva,
	// vuelven a abrir desde cero). gameOverNext es la acción pendiente
	// (NUEVA PARTIDA o MENÚ) que se hace al terminar de cerrar el Game
	// Over; el fin de día siempre termina en StartNextDay, así que no le
	// hace falta guardar nada.
	dayOverAnim  *animPanel
	gameOverAnim *animPanel
	gameOverNext func() (Scene, error)
}

// NewGameScene arranca una partida nueva (NUEVA PARTIDA): el azar de la
// partida, el cosmético y el de la bolsa de empleo salen de seedSource (la
// hora, salvo que el backdoor de pruebas la haya fijado), y el motor queda
// corriendo.
func NewGameScene() (*GameScene, error) {
	seed := seedSource()
	g, err := game.New(rand.New(rand.NewSource(seed)), rand.New(rand.NewSource(seed+1)), rand.New(rand.NewSource(seed+2)))
	if err != nil {
		return nil, err
	}
	s := newGameScene(g)
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel, s.done = cancel, g.Start(ctx)
	return s, nil
}

// newGameSceneFn es a quién llaman NUEVA PARTIDA (aquí y en title_scene.go)
// y "NUEVA PARTIDA" del Game Over. El backdoor de pruebas la reemplaza por
// una que no arranca el motor y usa Tick a mano, para que el guion sea
// determinista (agent.go).
var newGameSceneFn = NewGameScene

// newGameScene arma la escena para la partida g sin arrancar el motor: los
// tests la avanzan llamando g.Tick a mano.
func newGameScene(g *game.Game) *GameScene {
	s := &GameScene{world: newWorld(), g: g, actors: map[string]*actor{}, standing: spotBook{}, lying: spotBook{},
		noticeSeen: map[string]int{}, scenery: newScenery(rand.New(rand.NewSource(seedSource())))}
	s.snap = g.Snapshot()
	s.hud.update(s.snap)
	s.sync()
	return s
}

// La escena se puede pausar (Esc, PAUSA o perder el foco): mientras está la
// pausa la partida no avanza (Game.Pause congela reloj, cronómetros y
// llegadas). SALIR AL MENÚ detiene el motor.
func (s *GameScene) onPause()  { s.g.Pause() }
func (s *GameScene) onResume() { s.g.Resume() }
func (s *GameScene) onQuit() {
	if s.cancel != nil {
		s.cancel()
		<-s.done // espera a que el motor termine
	}
}

// pausesOnBlur: en el juego la pausa se abre sola si la ventana pierde el
// foco, para que el turno no corra mientras el jugador no mira (en la demo
// no hace falta).
func (s *GameScene) pausesOnBlur() bool { return true }

// Update lee la entrada y avanza la escena.
func (s *GameScene) Update() (Scene, error) { return s.update(readInput()) }

// update hace el trabajo de Update con la entrada ya leída (los tests la
// arman a mano).
func (s *GameScene) update(in input) (Scene, error) {
	s.ticks++
	s.snap = s.g.Snapshot()
	s.sound.update(s.snap)
	s.scenery.update()
	s.hud.update(s.snap)
	if s.messageTicks > 0 {
		s.messageTicks--
	}
	for _, n := range s.snap.Notices {
		if _, ok := s.noticeSeen[n]; !ok {
			s.noticeSeen[n] = s.ticks
		}
	}

	s.sync()
	s.animate()
	s.door.step(s.snap.DoorOpen)

	// Los paneles del fin del día y de la derrota se llevan la entrada.
	if s.snap.GameOver != game.NotOver {
		return s.updateGameOver(in)
	}
	if s.snap.DayOver {
		return s.updateDayOver(in)
	}

	s.shift = in.shift
	s.hover = ""
	if f, ok := s.figureAt(in.mouse); ok {
		s.hover = f.id
	}
	switch {
	case in.click && in.mouse.In(reportButton):
		playSound(sfxClick)
		s.g.Pause() // el reporte va con la partida en pausa
		return newReportScene(s.g.Report(), fmt.Sprintf("REPORTE DEL TURNO — DÍA %d", s.snap.Day), s), nil
	case in.click && in.mouse.In(pauseButton):
		playSound(sfxClick)
		return newPauseScene(s), nil
	case in.click && in.mouse.In(hireButton):
		if len(s.snap.Candidates) == 0 {
			s.say("Ya contrataste a los 3 candidatos de hoy")
		} else {
			playSound(sfxClick)
			return newHiringScene(s), nil
		}
	case in.click:
		s.click(in.mouse)
	case in.rightClick:
		s.rightClick(in.mouse)
	}
	return s, nil
}

// panelAnimFrom es desde dónde aparecen el panel del fin de día y el de
// Game Over: del botón REPORTE del HUD (son los dos "un reporte" de cómo
// quedó el turno o la partida; GAME_DESIGN §9.4), pedido de Samuel el
// 2026-10-17.
func panelAnimFrom() vec {
	c := reportButton.Min.Add(reportButton.Max).Div(2)
	return vec{float64(c.X), float64(c.Y)}
}

// Tiempos y escala mínima de estos dos paneles: los mismos que la bolsa de
// empleo y la pausa (hiringOpenTicks/hiringCloseTicks, pauseOpenTicks/
// pauseCloseTicks), con otro nombre porque cada escena guarda los suyos.
const (
	panelOpenTicks  = 13 // 0,22 s
	panelCloseTicks = 11 // 0,18 s
	panelMinScale   = 0.167
)

// dayOverOpenCenter y gameOverOpenCenter son el centro de cada panel ya
// abierto (el centro de su caja en drawDayOver/drawGameOver).
var (
	dayOverOpenCenter  = vec{640, 355}
	gameOverOpenCenter = vec{640, 365}
)

// updateDayOver: a las 20:00 termina el turno. El panel abre animado la
// primera vez que se ve DayOver, y SIGUIENTE DÍA (o Espacio/Enter) arranca
// el cierre; StartNextDay se llama al terminar de cerrar, no al hacer
// clic (igual que la bolsa de empleo con CONTRATAR, pero acá la acción
// sale de la escena, no del panel).
func (s *GameScene) updateDayOver(in input) (Scene, error) {
	if s.dayOverAnim == nil {
		// El primer tick que se ve DayOver arranca a abrir en 0: no avanza
		// la animación todavía (el tick que crea el panel no es el primero
		// que ya lo mueve, igual que la bolsa de empleo, donde abrir y
		// crear la escena son dos pasos separados).
		s.dayOverAnim = newAnimPanel(panelOpenTicks, panelCloseTicks, panelAnimFrom(), dayOverOpenCenter, panelMinScale)
		return s, nil
	}
	if !s.dayOverAnim.acceptsInput() {
		if s.dayOverAnim.step() {
			s.dayOverAnim = nil // listo para abrir de cero el día que vuelva a terminar
			if err := s.g.StartNextDay(); err != nil {
				s.say(err.Error())
			}
		}
		return s, nil
	}
	if in.confirm || in.space || (in.click && in.mouse.In(dayOverButton)) {
		playSound(sfxClick)
		s.dayOverAnim.startClose()
	}
	return s, nil
}

// updateGameOver: la partida se perdió. Mismo patrón que updateDayOver,
// pero la acción pendiente (NUEVA PARTIDA o MENÚ) sí depende de qué
// botón se usó, así que se guarda en gameOverNext.
func (s *GameScene) updateGameOver(in input) (Scene, error) {
	if s.gameOverAnim == nil {
		s.gameOverAnim = newAnimPanel(panelOpenTicks, panelCloseTicks, panelAnimFrom(), gameOverOpenCenter, panelMinScale)
		return s, nil // igual que updateDayOver: el tick que crea el panel no lo mueve todavía
	}
	if !s.gameOverAnim.acceptsInput() {
		if s.gameOverAnim.step() {
			return s.gameOverNext()
		}
		return s, nil
	}
	switch {
	case in.click && in.mouse.In(newGameButton), in.confirm:
		playSound(sfxClick)
		s.gameOverNext = func() (Scene, error) { s.onQuit(); return newGameSceneFn() }
		s.gameOverAnim.startClose()
	case in.click && in.mouse.In(menuButton), in.back:
		playSound(sfxClick)
		s.gameOverNext = func() (Scene, error) { s.onQuit(); return NewTitleScene(), nil }
		s.gameOverAnim.startClose()
	}
	return s, nil
}

// dayOverAnimOrOpen y gameOverAnimOrOpen dan la animación del panel, o una
// ya quieta y abierta si Draw se llama antes de que update la haya creado
// (no debería pasar en el juego real, donde Update siempre corre antes que
// Draw, pero sí en un test que dibuje sin pasar por update primero).
func (s *GameScene) dayOverAnimOrOpen() *animPanel {
	if s.dayOverAnim == nil {
		return &animPanel{phase: panelOpen, to: dayOverOpenCenter}
	}
	return s.dayOverAnim
}

func (s *GameScene) gameOverAnimOrOpen() *animPanel {
	if s.gameOverAnim == nil {
		return &animPanel{phase: panelOpen, to: gameOverOpenCenter}
	}
	return s.gameOverAnim
}

// Botones de abajo a la derecha (§9.1) y de los paneles (Figma, vistas
// aprobadas del Reporte del día y el Game Over, §9: "boton" de cada
// frame).
var (
	reportButton  = image.Rect(1106, 672, 1186, 708)
	pauseButton   = image.Rect(1194, 672, 1268, 708)
	dayOverButton = image.Rect(772, 606, 1212, 662)
	newGameButton = image.Rect(334, 520, 714, 578)
	menuButton    = image.Rect(726, 520, 946, 578)
)

// Draw dibuja por capas, como la demo: mapa → carros y humo → personajes y
// camillas → copas de las palmeras → cobijas → rótulos → barra, botones,
// avisos y ficha → paneles.
func (s *GameScene) Draw(screen *ebiten.Image) {
	drawMap(screen)
	drawMainDoor(screen, s.door.open)
	s.scenery.draw(screen)

	type drawable struct {
		y    float64
		draw func()
	}
	var items []drawable
	lyingIn := map[int]bool{}
	for _, f := range s.figures {
		if f.patient && s.onBed(f) {
			lyingIn[f.room] = true
		}
		y := s.pos[f.id].y + figHeight
		if c := s.carryOf(f.id); c != nil && c.visible() && f.id == c.order.patient {
			if c.riding() {
				continue // va sobre la camilla: se dibuja con ella
			}
			y = float64(c.fp.Max.Y) + 0.5
		}
		items = append(items, drawable{y, func() {
			drawFigureBody(screen, f, s.pos[f.id], s.anims[f.id], s.staff[f.id])
		}})
	}
	for _, c := range s.others {
		if !c.visible() {
			continue
		}
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
	drawNeedDoctorIcons(screen, s.snap.Patients, s.ticks)

	for _, o := range roomBadges(s.snap.Rooms) {
		drawOverlay(screen, o)
	}
	for _, f := range s.figures {
		if s.actorOf(f.id).exiting {
			continue
		}
		if f.id == s.selected { // el elegido, con un marco amarillo alrededor
			box := s.figureBox(f).Inset(-4)
			for _, edge := range []image.Rectangle{
				{box.Min, image.Pt(box.Max.X, box.Min.Y+3)}, {image.Pt(box.Min.X, box.Max.Y-3), box.Max},
				{box.Min, image.Pt(box.Min.X+3, box.Max.Y)}, {image.Pt(box.Max.X-3, box.Min.Y), box.Max},
			} {
				fillRect(screen, edge, colYellow)
			}
		}
		for _, o := range s.gameOverlays(f, s.shift || f.id == s.hover || f.id == s.selected) {
			drawOverlay(screen, o)
		}
	}

	// Los avisos del juego (errores, altas) van primero para saber dónde
	// terminan: los avisos de reputación con motivo (hud.go, §12.8) no se
	// pueden encimar con ellos (los dos salen pegados a la izquierda, bajo
	// la barra, y "P-xxx se fue enojado" deja un Notice Y un cambio de
	// reputación al mismo tiempo).
	noticesBottom := s.drawNotices(screen)
	s.hud.draw(screen, s.snap, noticesBottom)
	s.drawButtons(screen)
	if s.hover != "" && !s.shift {
		if f, ok := s.figure(s.hover); ok {
			s.drawTooltip(screen, f)
		}
	}
	switch {
	case s.snap.GameOver != game.NotOver:
		s.drawGameOver(screen)
	case s.snap.DayOver:
		s.drawDayOver(screen)
	}
}

// drawButtons: CONTRATAR (un portapapeles chiquito: "Próximamente" hasta
// F4), REPORTE y PAUSA, abajo a la derecha.
func (s *GameScene) drawButtons(dst *ebiten.Image) {
	s.drawHireClipboard(dst)
	for _, b := range []struct {
		r    image.Rectangle
		text string
	}{{reportButton, "REPORTE"}, {pauseButton, "PAUSA"}} {
		panel(dst, b.r, colPaper)
		drawText(dst, b.text, float64(b.r.Min.X+(b.r.Dx()-int(textWidth(b.text, smallFace)))/2), float64(b.r.Min.Y+9), smallFace, colInk)
	}
}

// noticeShowTicks: cuánto se ve un aviso del juego (6 s).
const noticeShowTicks = 360

// noticeLines son los avisos del juego que se ven ahora: hasta 3 de
// Snapshot.Notices (errores del modelo, altas) de los últimos 6 s, y el
// aviso de la escena, si hay.
func (s *GameScene) noticeLines() []string {
	var lines []string
	for _, n := range s.snap.Notices {
		if s.ticks-s.noticeSeen[n] < noticeShowTicks {
			lines = append(lines, n)
		}
	}
	if len(lines) > 3 {
		lines = lines[len(lines)-3:]
	}
	if s.messageTicks > 0 {
		lines = append(lines, s.message)
	}
	return lines
}

// drawNotices dibuja los avisos del juego en la esquina de arriba a la
// izquierda, debajo de la barra, apilados hacia abajo. Devuelve el Y libre
// justo debajo del último (o, si no hay ninguno, debajo de la barra): ahí
// empiezan los avisos de reputación con motivo (hud.go, §12.8), para que no
// se encimen con estos.
func (s *GameScene) drawNotices(dst *ebiten.Image) int {
	y := hudHeight + 2*hudBorder + 8
	for _, l := range s.noticeLines() {
		r := textBox(l, 12, y, smallFace)
		panel(dst, r, colPaper)
		drawText(dst, l, float64(r.Min.X+4), float64(r.Min.Y+2), smallFace, colInk)
		y = r.Max.Y + 4
	}
	return y
}

// drawDayOver: a las 20:00 termina el turno y se paga la nómina (§5.8,
// §9): el tablero de corcho con la ficha del título, el balance del día,
// la nómina, los contratados, las cuatro consultas de la Sección 5, el
// puntaje y el botón SIGUIENTE DÍA (vista aprobada en Figma, "Reporte A –
// día con ganancia"/"Reporte B – día con pérdida"; el dibujo está en
// report_panel.go, que no depende de GameScene: solo del DayReport del
// último día cerrado y del puntaje actual). El panel abre y cierra
// animado (drawAnimatedPanel, anim.go), igual que la bolsa de empleo.
//
// ⚠️ Pendiente (§12.9, "Mejor racha"): la vista aprobada de Figma no trae
// un renglón para la mejor racha del día; se dejó de mostrar acá. El dato
// sigue en DayReport.BestStreak por si Samuel lo quiere de vuelta en
// algún lado.
func (s *GameScene) drawDayOver(dst *ebiten.Image) {
	drawAnimatedPanel(dst, s.dayOverAnimOrOpen(), pauseDim.A, func(c *ebiten.Image) {
		drawDayOverPanel(c, s.g.DayReport(), s.snap.Score)
	})
}

// drawGameOver: la partida se perdió (§5.10, §9): el portapapeles con el
// ícono y el sello del motivo, los días sobrevividos, el desglose del
// puntaje final (plata ÷ 10, estrellas × 200) y NUEVA PARTIDA o MENÚ
// (vista aprobada en Figma, "Game Over C1/C2/C3"; el dibujo está en
// gameover_panel.go, que solo depende de Snapshot). El panel abre y
// cierra animado, igual que el fin de día.
func (s *GameScene) drawGameOver(dst *ebiten.Image) {
	drawAnimatedPanel(dst, s.gameOverAnimOrOpen(), pauseDim.A, func(c *ebiten.Image) {
		drawGameOverPanel(c, s.snap)
	})
}
