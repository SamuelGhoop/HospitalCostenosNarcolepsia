package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// El backdoor de pruebas (GAME_DESIGN §10.9): el agente tester llega a
// cualquier escena en segundos, sin una persona jugando, con un guion de
// -input, saca capturas y un dump de estado, y sale. Nunca toca archivos
// del jugador (no hay partida guardada todavía) ni suena nada (hoy no hay
// sonido).
//
// La idea: las escenas ya leen el teclado y el mouse a través de tres
// variables de agent_hooks.go (keyJustPressed, clickJustPressed,
// readInputFn) en vez de llamar directo a Ebitengine. agentRunner las
// reemplaza por la entrada del guion, así ninguna escena se entera de que
// la está jugando un guion y no una persona.

// AgentConfig son los -flags del backdoor (gui/main.go).
type AgentConfig struct {
	Scene      string // title, demo, game, report, pause
	Step       int    // demo: paso al que salta (0 = desde el principio)
	Seed       int64  // semilla del azar de la partida/demo y del tráfico
	ScriptPath string // -input: guion de instrucciones; "" = ninguno
	Frames     int    // ticks de Update sin entrada, después del guion
	ShotPath   string // captura al final; "" = ninguna
	DumpPath   string // dump al final; "" = ninguno
	Quit       bool   // termina solo al acabar; si no, deja la ventana abierta
}

// RunAgent arma la escena pedida, instala los relojes de prueba (semilla
// fija, Tick a mano en vez del motor) y corre el guion, si hay. Es el único
// camino que abre una ventana real de Ebitengine: hace falta para poder
// capturar píxeles (shot). Los tests headless usan newAgentRunner directo
// (agent_test.go), sin RunGame.
func RunAgent(cfg AgentConfig) error {
	if err := loadAssets(); err != nil {
		return err
	}
	var instr []instruction
	if cfg.ScriptPath != "" {
		var err error
		if instr, err = ParseScript(cfg.ScriptPath); err != nil {
			return err
		}
	}

	installAgentDeterminism(cfg.Seed)

	scene, err := buildAgentScene(cfg)
	if err != nil {
		return err
	}

	ag := &agentGame{runner: newAgentRunner(&App{scene: scene}, instr), cfg: cfg}
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle("Hospital de los Costeños con Narcolepsia — agente de pruebas")
	if err := ebiten.RunGame(ag); err != nil && !errors.Is(err, ebiten.Termination) {
		return err
	}
	return nil
}

// installAgentDeterminism fija el azar (seedSource, que usan las escenas
// nuevas) y hace que el modo juego nunca arranque su goroutine motor: en
// cambio, agentRunner llama Game.Tick a mano, como los tests de game/
// (GAME_DESIGN §11). También fuerza isFocused a verdadero: sin ventana real
// con foco, GameScene.pausesOnBlur() abriría la pausa en todos los ticks.
func installAgentDeterminism(seed int64) {
	seedSource = func() int64 { return seed }
	isFocused = func() bool { return true }
	newGameSceneFn = func() (*GameScene, error) {
		g, err := game.New(rand.New(rand.NewSource(seed)), rand.New(rand.NewSource(seed+1)), rand.New(rand.NewSource(seed+2)))
		if err != nil {
			return nil, err
		}
		return newGameScene(g), nil
	}
}

// buildAgentScene arma la escena de arranque (-scene).
func buildAgentScene(cfg AgentConfig) (Scene, error) {
	switch cfg.Scene {
	case "", "title":
		return NewTitleScene(), nil
	case "demo":
		return buildDemoScene(cfg.Step)
	case "game":
		return newGameSceneFn()
	case "pause":
		under, err := newGameSceneFn()
		if err != nil {
			return nil, err
		}
		return newPauseScene(under), nil
	case "hiring":
		under, err := newGameSceneFn()
		if err != nil {
			return nil, err
		}
		return newHiringSceneOpen(under), nil
	case "report":
		return buildReportScene()
	default:
		return nil, fmt.Errorf("escena desconocida: %q (válidas: title, demo, game, report, pause, hiring)", cfg.Scene)
	}
}

// buildDemoScene arranca la demo y, si step > 0, la adelanta hasta ese
// paso (de 1 a game.DemoSteps), con cada personaje ya en su puesto final
// (sin reproducir las coreografías de los pasos intermedios: así -step
// lleva directo a la foto de ese paso).
func buildDemoScene(step int) (*DemoScene, error) {
	d := game.NewDemo()
	s := NewDemoScene(d)
	for i := 0; i < step && !d.Done(); i++ {
		if err := s.advance(); err != nil {
			return nil, fmt.Errorf("adelantando la demo al paso %d: %w", i+1, err)
		}
		s.finishChoreography()
	}
	return s, nil
}

// buildReportScene corre la demostración completa y muestra su Shift
// Report (para revisar esa pantalla sin jugar los 12 pasos a mano).
func buildReportScene() (*ReportScene, error) {
	d := game.NewDemo()
	for !d.Done() {
		if _, err := d.Next(); err != nil {
			return nil, err
		}
	}
	return NewReportScene(d.Report()), nil
}

// ─── El driver: lee el guion y empuja la App tick a tick ──────────────────

// agentRunner ejecuta un guion sobre una App ya armada, instalando en cada
// tick los hooks de agent_hooks.go con la entrada que toca. No sabe nada de
// ventanas ni de píxeles: shot solo deja pendingShot listo para que lo
// capture agentGame.Draw con el cuadro real que sigue.
type agentRunner struct {
	app   *App
	instr []instruction
	idx   int // próxima instrucción del guion

	appTicks int // ticks de Update dados en total (para el dump)

	waitLeft int // ticks que le quedan a un "wait" en curso

	holdKey   ebiten.Key // 0 = ninguna tecla sostenida ahora
	holdName  string
	holdLeft  int  // ticks que le quedan a un "hold" en curso
	holdFired bool // ya se dio el tick en que se "acaba de apretar"

	click, rightClick bool
	mouse, prevMouse  image.Point

	pendingShot string // ruta pendiente de capturar en el próximo Draw; "" = ninguna
	dump        func(path string, v any) error

	quit    bool // se acabó el guion (con o sin "quit")
	lastErr error
}

// newAgentRunner arma el driver con dump de producción (writeDump); los
// tests lo pueden reemplazar para no tocar disco.
func newAgentRunner(app *App, instr []instruction) *agentRunner {
	return &agentRunner{app: app, instr: instr, dump: writeDump}
}

// writeDump guarda v como JSON legible en path.
func writeDump(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}

// step da un cuadro real de guion: cada llamada corresponde a un Update()
// de Ebitengine. Devuelve seguir = false cuando ya no queda nada del
// guion (lo demás lo hacen los -flags finales, en agentGame.Update).
func (r *agentRunner) step() (seguir bool, err error) {
	switch {
	case r.waitLeft > 0:
		r.driveFrame()
		r.waitLeft--
		return true, r.lastErr
	case r.holdLeft > 0:
		r.driveFrame()
		r.holdLeft--
		if r.holdLeft == 0 {
			r.holdKey, r.holdName = 0, ""
		}
		return true, r.lastErr
	}
	if r.idx >= len(r.instr) {
		r.quit = true
		return false, nil
	}
	in := r.instr[r.idx]
	r.idx++
	switch in.kind {
	case insWait:
		if in.ticks <= 0 {
			return true, nil
		}
		r.driveFrame()
		r.waitLeft = in.ticks - 1
		return true, r.lastErr
	case insClick:
		r.mouse, r.click = image.Pt(in.x, in.y), true
		r.driveFrame()
		return true, r.lastErr
	case insRClick:
		r.mouse, r.rightClick = image.Pt(in.x, in.y), true
		r.driveFrame()
		return true, r.lastErr
	case insMouse:
		r.mouse = image.Pt(in.x, in.y)
		r.driveFrame()
		return true, r.lastErr
	case insKey:
		r.holdKey, r.holdName, r.holdFired = in.key, in.keyName, false
		r.driveFrame()
		r.holdKey, r.holdName = 0, ""
		return true, r.lastErr
	case insHold:
		if in.ticks <= 0 {
			return true, nil
		}
		r.holdKey, r.holdName, r.holdFired = in.key, in.keyName, false
		r.driveFrame()
		r.holdLeft = in.ticks - 1
		if r.holdLeft == 0 {
			r.holdKey, r.holdName = 0, ""
		}
		return true, r.lastErr
	case insShot:
		r.pendingShot = in.path // lo captura agentGame.Draw con el cuadro que sigue
		return true, nil
	case insDump:
		if err := r.dump(in.path, r.dumpState()); err != nil {
			return false, fmt.Errorf("dump %s: %w", in.path, err)
		}
		return true, nil
	case insQuit:
		r.quit = true
		return false, nil
	}
	return true, nil
}

// driveFrame instala los hooks de este tick (la tecla sostenida, el clic,
// el mouse) y le da un Update a la App. Si la escena actual es el modo
// juego, antes le da a Game un Tick de 100 ms cada 6 Updates (60 Updates
// por segundo de Ebitengine ÷ 6 = 10 por segundo = uno cada 100 ms), igual
// que la goroutine motor real (GAME_DESIGN §11) y que los tests de game/.
func (r *agentRunner) driveFrame() {
	justNow := r.holdKey != 0 && !r.holdFired
	moved := r.mouse != r.prevMouse
	holdKey, click, rightClick, mouse := r.holdKey, r.click, r.rightClick, r.mouse

	keyJustPressed = func(k ebiten.Key) bool { return justNow && holdKey == k }
	clickJustPressed = func() bool { return click }
	readInputFn = func() input {
		return input{
			up:         justNow && (holdKey == ebiten.KeyArrowUp || holdKey == ebiten.KeyW),
			down:       justNow && (holdKey == ebiten.KeyArrowDown || holdKey == ebiten.KeyS),
			left:       justNow && holdKey == ebiten.KeyArrowLeft,
			right:      justNow && holdKey == ebiten.KeyArrowRight,
			confirm:    justNow && (holdKey == ebiten.KeyEnter || holdKey == ebiten.KeyNumpadEnter),
			space:      justNow && holdKey == ebiten.KeySpace,
			back:       justNow && holdKey == ebiten.KeyEscape,
			click:      click,
			rightClick: rightClick,
			shift:      holdKey == ebiten.KeyShift,
			mouse:      mouse,
			mouseMoved: moved,
		}
	}

	if gs, ok := r.app.scene.(*GameScene); ok && gs.ticks%6 == 0 {
		gs.g.Tick(100 * time.Millisecond)
	}

	r.appTicks++
	r.lastErr = r.app.Update()

	r.holdFired = true
	r.prevMouse = r.mouse
	r.click, r.rightClick = false, false
}

// ─── agentGame: el ebiten.Game real, con ventana y capturas ───────────────

// agentGame implementa ebiten.Game para el backdoor: cada cuadro real le
// pide al runner que siga el guion; cuando se acaba, da los -frames que
// falten y hace -shot/-dump una sola vez, y termina si -quit (si no, deja
// la ventana quieta para mirarla a mano).
type agentGame struct {
	runner *agentRunner
	cfg    AgentConfig

	readyFinal               bool // el guion ya terminó: entramos a los -flags finales
	framesLeft               int
	finalShotDone, finalDump bool
}

func (a *agentGame) Update() error {
	if !a.readyFinal {
		seguir, err := a.runner.step()
		if err != nil {
			return err
		}
		if seguir {
			return nil
		}
		a.readyFinal = true
		a.framesLeft = a.cfg.Frames
	}
	if a.framesLeft > 0 {
		a.runner.driveFrame()
		a.framesLeft--
		return a.runner.lastErr
	}
	if a.cfg.ShotPath != "" && !a.finalShotDone {
		a.finalShotDone = true
		a.runner.pendingShot = a.cfg.ShotPath
		return nil // lo captura Draw con este mismo cuadro
	}
	if a.cfg.DumpPath != "" && !a.finalDump {
		a.finalDump = true
		return a.runner.dump(a.cfg.DumpPath, a.runner.dumpState())
	}
	if a.cfg.Quit {
		return ebiten.Termination
	}
	return nil
}

func (a *agentGame) Draw(screen *ebiten.Image) {
	a.runner.app.Draw(screen)
	if a.runner.pendingShot != "" {
		path := a.runner.pendingShot
		a.runner.pendingShot = ""
		if err := saveScreenshot(screen, path); err != nil {
			// No hay cómo devolver el error desde Draw: lo dejamos en
			// lastErr, que Update() ya está revisando en el próximo cuadro.
			a.runner.lastErr = fmt.Errorf("shot %s: %w", path, err)
		}
	}
}

func (a *agentGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

// saveScreenshot guarda lo que hay en screen como PNG a 1× (la pantalla ya
// es 1280×720: a diferencia de los sprites, no hace falta escalar).
func saveScreenshot(screen *ebiten.Image, path string) error {
	b := screen.Bounds()
	out := image.NewRGBA(b)
	draw.Draw(out, b, screen, b.Min, draw.Src)
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, out)
}

// ─── El dump: lo que un tester tendría que adivinar de los píxeles ────────

// dumpState es el contenido de "dump ruta.json".
type dumpState struct {
	Scene string    `json:"scene"`
	Tick  int       `json:"tick"`
	Demo  *dumpDemo `json:"demo,omitempty"`
	Game  *dumpGame `json:"game,omitempty"`
}

// dumpDemo: en la demostración, solo el paso actual (lo demás ya sale en
// GAME_DESIGN y en el Shift Report).
type dumpDemo struct {
	Step       int `json:"step"`
	TotalSteps int `json:"totalSteps"`
}

// dumpGame es la foto del modo juego, con la posición en pantalla que usa
// la interfaz (la misma con la que dibuja, redondeada igual que al dibujar).
type dumpGame struct {
	Clock      string          `json:"clock"`
	Day        int             `json:"day"`
	Money      int             `json:"money"`
	Reputation int             `json:"reputation"`
	Notices    []string        `json:"notices"`
	Patients   []dumpPatient   `json:"patients"`
	Staff      []dumpStaff     `json:"staff"`
	Stretchers []dumpStretcher `json:"stretchers"`
	Candidates []dumpCandidate `json:"candidates,omitempty"` // solo con la bolsa de empleo abierta
}

// dumpCandidate es una ficha de la bolsa de empleo en el dump.
type dumpCandidate struct {
	Name       string `json:"name"`
	IsDoctor   bool   `json:"isDoctor"`
	Cost       int    `json:"cost"`
	Affordable bool   `json:"affordable"`
}

type dumpPatient struct {
	ID    string `json:"id"`
	State string `json:"state"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
}

type dumpStaff struct {
	ID       string `json:"id"`
	Role     string `json:"role"`
	Activity string `json:"activity"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
}

// dumpStretcher: una camilla activa, su huella en el piso y hacia dónde
// apunta (ver gui/ui/carry.go).
type dumpStretcher struct {
	X   int    `json:"x"`
	Y   int    `json:"y"`
	W   int    `json:"w"`
	H   int    `json:"h"`
	Dir string `json:"dir"`
}

// sceneName es cómo sale cada escena en el dump; coincide con los nombres
// de -scene (GAME_DESIGN §10.9).
func sceneName(s Scene) string {
	switch s.(type) {
	case *TitleScene:
		return "title"
	case *DemoScene:
		return "demo"
	case *GameScene:
		return "game"
	case *ReportScene:
		return "report"
	case *PauseScene:
		return "pause"
	case *HiringScene:
		return "hiring"
	case *CreditsScene:
		return "credits"
	default:
		return fmt.Sprintf("%T", s)
	}
}

// dumpState arma el dump de la escena actual.
func (r *agentRunner) dumpState() dumpState {
	d := dumpState{Scene: sceneName(r.app.scene), Tick: r.appTicks}
	switch s := r.app.scene.(type) {
	case *DemoScene:
		d.Demo = &dumpDemo{Step: s.snap.Step, TotalSteps: s.snap.TotalSteps}
	case *GameScene:
		d.Game = s.dumpGame()
	case *HiringScene: // la bolsa deja ver el juego de abajo tal como quedó, más sus candidatos
		d.Game = s.under.dumpGame()
		for _, c := range s.snap.Candidates {
			d.Game.Candidates = append(d.Game.Candidates, dumpCandidate{Name: c.Name, IsDoctor: c.IsDoctor, Cost: c.Cost, Affordable: c.Affordable})
		}
	case *PauseScene: // la pausa deja ver la escena de abajo tal como quedó
		switch under := s.under.(type) {
		case *GameScene:
			d.Game = under.dumpGame()
		case *DemoScene:
			d.Demo = &dumpDemo{Step: under.snap.Step, TotalSteps: under.snap.TotalSteps}
		}
	}
	return d
}

// dumpGame es la foto de GameScene para el dump: la posición en pantalla
// de cada uno sale de s.pos, la misma que usa Draw.
func (s *GameScene) dumpGame() *dumpGame {
	g := &dumpGame{
		Clock:      s.snap.Clock,
		Day:        s.snap.Day,
		Money:      s.snap.Money,
		Reputation: s.snap.Reputation,
		Notices:    append([]string(nil), s.snap.Notices...),
	}
	for _, p := range s.snap.Patients {
		at := s.pos[p.ID]
		g.Patients = append(g.Patients, dumpPatient{
			ID: p.ID, State: p.Stage.String(),
			X: int(math.Round(at.x)), Y: int(math.Round(at.y)),
		})
	}
	for _, st := range s.snap.Staff {
		at := s.pos[st.ID]
		role := "camillero"
		if st.IsDoctor {
			role = "médico"
		}
		g.Staff = append(g.Staff, dumpStaff{
			ID: st.ID, Role: role, Activity: st.Activity.String(),
			X: int(math.Round(at.x)), Y: int(math.Round(at.y)),
		})
	}
	for _, c := range s.allCarries() {
		if !c.visible() {
			continue
		}
		g.Stretchers = append(g.Stretchers, dumpStretcher{
			X: c.fp.Min.X, Y: c.fp.Min.Y, W: c.fp.Dx(), H: c.fp.Dy(), Dir: c.dir.String(),
		})
	}
	return g
}
