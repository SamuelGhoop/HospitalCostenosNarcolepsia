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

	selected     string // paciente elegido para mandarle a alguien ("" = ninguno)
	hover        string // personaje bajo el mouse
	shift        bool   // Shift apretado: todas las etiquetas
	message      string // aviso de la escena (errores, "Próximamente")
	messageTicks int
	noticeSeen   map[string]int // avisos del juego → tick en que aparecieron
	ticks        int
}

// NewGameScene arranca una partida nueva (NUEVA PARTIDA): el azar de la
// partida y el cosmético salen de seedSource (la hora, salvo que el
// backdoor de pruebas la haya fijado), y el motor queda corriendo.
func NewGameScene() (*GameScene, error) {
	seed := seedSource()
	g, err := game.New(rand.New(rand.NewSource(seed)), rand.New(rand.NewSource(seed+1)))
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
		return s.gameOverInput(in)
	}
	if s.snap.DayOver {
		if in.confirm || in.space || (in.click && in.mouse.In(dayOverButton)) {
			if err := s.g.StartNextDay(); err != nil {
				s.say(err.Error())
			}
		}
		return s, nil
	}

	s.shift = in.shift
	s.hover = ""
	if f, ok := s.figureAt(in.mouse); ok {
		s.hover = f.id
	}
	switch {
	case in.click && in.mouse.In(reportButton):
		s.g.Pause() // el reporte va con la partida en pausa
		return &ReportScene{report: s.g.Report(), title: fmt.Sprintf("REPORTE DEL TURNO — DÍA %d", s.snap.Day), back: s}, nil
	case in.click && in.mouse.In(pauseButton):
		return newPauseScene(s), nil
	case in.click && in.mouse.In(hireButton):
		s.say("Próximamente: la bolsa de empleo llega en F4")
	case in.click:
		s.click(in.mouse)
	case in.rightClick:
		s.rightClick(in.mouse)
	}
	return s, nil
}

// gameOverInput: con la partida perdida, NUEVA PARTIDA o MENÚ.
func (s *GameScene) gameOverInput(in input) (Scene, error) {
	switch {
	case in.click && in.mouse.In(newGameButton), in.confirm:
		s.onQuit()
		return newGameSceneFn()
	case in.click && in.mouse.In(menuButton), in.back:
		s.onQuit()
		return NewTitleScene(), nil
	}
	return s, nil
}

// Botones de abajo a la derecha (§9.1) y de los paneles.
var (
	hireButton    = image.Rect(990, 664, 1098, 708) // el portapapeles chiquito de CONTRATAR
	reportButton  = image.Rect(1106, 672, 1186, 708)
	pauseButton   = image.Rect(1194, 672, 1268, 708)
	dayOverButton = image.Rect(520, 400, 760, 448)
	newGameButton = image.Rect(380, 430, 620, 478)
	menuButton    = image.Rect(660, 430, 900, 478)
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

	s.hud.draw(screen, s.snap)
	s.drawButtons(screen)
	s.drawNotices(screen)
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
	panel(dst, hireButton, colWood)
	paper := hireButton.Inset(6)
	paper.Min.Y += 6
	panel(dst, paper, colPaper)
	fillRect(dst, image.Rect(hireButton.Min.X+30, hireButton.Min.Y-4, hireButton.Max.X-30, hireButton.Min.Y+8), colMetal) // el gancho
	drawText(dst, "CONTRATAR", float64(paper.Min.X+(paper.Dx()-int(textWidth("CONTRATAR", smallFace)))/2), float64(paper.Min.Y+8), smallFace, colInk)
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

// drawNotices: los avisos del juego (errores del modelo, altas) en la
// esquina de arriba a la izquierda, debajo de la barra, unos segundos; y
// el aviso de la escena, si hay.
func (s *GameScene) drawNotices(dst *ebiten.Image) {
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
	y := hudHeight + 2*hudBorder + 8
	for _, l := range lines {
		r := textBox(l, 12, y, smallFace)
		panel(dst, r, colPaper)
		drawText(dst, l, float64(r.Min.X+4), float64(r.Min.Y+2), smallFace, colInk)
		y = r.Max.Y + 4
	}
}

// drawDayOver: a las 20:00 termina el turno (la nómina y el reporte del día
// llegan en F4).
func (s *GameScene) drawDayOver(dst *ebiten.Image) {
	fillRect(dst, image.Rect(0, 0, ScreenWidth, ScreenHeight), pauseDim)
	box := image.Rect(400, 240, 880, 470)
	panel(dst, box, colPaper)
	title := fmt.Sprintf("FIN DEL DÍA %d", s.snap.Day)
	drawPixelText(dst, title, ScreenWidth/2-pixelTextWidth(title, 24)/2, 280, 24, []textLayer{{0, 0, colInk}}, 1)
	score := fmt.Sprintf("Puntaje: %d", s.snap.Score)
	drawText(dst, score, float64(ScreenWidth/2)-textWidth(score, textFace)/2, 340, textFace, colMuted)
	panel(dst, dayOverButton, colBlue)
	drawText(dst, "SIGUIENTE DÍA", float64(dayOverButton.Min.X+(dayOverButton.Dx()-int(textWidth("SIGUIENTE DÍA", boldFace)))/2), float64(dayOverButton.Min.Y+12), boldFace, colWhite)
}

// drawGameOver: la partida se perdió (§5.10): el motivo, el puntaje final y
// los días; NUEVA PARTIDA o MENÚ.
func (s *GameScene) drawGameOver(dst *ebiten.Image) {
	fillRect(dst, image.Rect(0, 0, ScreenWidth, ScreenHeight), pauseDim)
	box := image.Rect(340, 220, 940, 510)
	panel(dst, box, colPaper)
	title := s.snap.GameOver.String()
	drawPixelText(dst, title, ScreenWidth/2-pixelTextWidth(title, 24)/2, 260, 24, append(outlined(2, colInk), textLayer{0, 0, colHUDRed}), 1)
	for i, l := range []string{fmt.Sprintf("Puntaje final: %d", s.snap.FinalScore), fmt.Sprintf("Días: %d", s.snap.Day)} {
		drawText(dst, l, float64(ScreenWidth/2)-textWidth(l, textFace)/2, float64(320+i*34), textFace, colInk)
	}
	for _, b := range []struct {
		r    image.Rectangle
		text string
	}{{newGameButton, "NUEVA PARTIDA"}, {menuButton, "MENÚ"}} {
		panel(dst, b.r, colBlue)
		drawText(dst, b.text, float64(b.r.Min.X+(b.r.Dx()-int(textWidth(b.text, boldFace)))/2), float64(b.r.Min.Y+12), boldFace, colWhite)
	}
}
