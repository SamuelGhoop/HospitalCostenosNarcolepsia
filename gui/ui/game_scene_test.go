package ui

import (
	"image"
	"math"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// El modo juego (F3b). Los tests no arrancan el motor: avanzan la partida
// con g.Tick a mano, un tick de 100 ms cada 6 ticks de la escena (a 60 por
// segundo), como pasaría con el motor corriendo.

func newTestGameScene(t *testing.T) *GameScene {
	t.Helper()
	needFonts(t)
	// seedSource da la semilla del cosmético (scenery, wanderRNG...): sin
	// fijarla acá salía de time.Now().UnixNano() y el test de deambular del
	// personal libre (staff_rest.go) fallaba de vez en cuando. Se restaura
	// al terminar el test para no afectar a los demás (ni al juego normal
	// ni al backdoor -seed, que la fija por su cuenta en agent.go).
	old := seedSource
	seedSource = func() int64 { return 42 }
	t.Cleanup(func() { seedSource = old })
	g, err := game.New(rand.New(rand.NewSource(7)), rand.New(rand.NewSource(8)), rand.New(rand.NewSource(9)))
	if err != nil {
		t.Fatal(err)
	}
	return newGameScene(g)
}

// play avanza la escena n ticks (y la partida, 1 de cada 6) sin entrada.
func play(s *GameScene, n int) {
	for i := 0; i < n; i++ {
		if s.ticks%6 == 0 {
			s.g.Tick(100 * time.Millisecond)
		}
		s.update(input{mouse: image.Pt(-100, -100)})
	}
}

// playUntil avanza hasta que cond se cumpla (o falla a los limit ticks).
func playUntil(t *testing.T, s *GameScene, limit int, what string, cond func() bool) {
	t.Helper()
	for i := 0; i < limit; i++ {
		if cond() {
			return
		}
		play(s, 1)
	}
	t.Fatalf("en %d ticks no pasó: %s", limit, what)
}

// staffNow es la foto de un miembro del personal pedida ahora al juego (la
// de la escena es la del principio del tick, antes del clic).
func staffNow(s *GameScene, id string) game.StaffMemberView {
	for _, st := range s.g.Snapshot().Staff {
		if st.ID == id {
			return st
		}
	}
	return game.StaffMemberView{}
}

// stageOf es la etapa de un paciente en la foto ("" si no está).
func stageOf(s *GameScene, id string) game.Stage {
	p, _ := s.patientView(id)
	return p.Stage
}

// clickOn hace clic (o clic derecho) en el centro de lo que se ve de id.
func clickOn(s *GameScene, id string, right bool) {
	f, _ := s.figure(id)
	b := s.figureBox(f)
	c := b.Min.Add(b.Max).Div(2)
	if right {
		s.update(input{rightClick: true, mouse: c})
	} else {
		s.update(input{click: true, mouse: c})
	}
}

// firstCollapsed espera a que alguien se desplome y termine de acostarse.
func firstCollapsed(t *testing.T, s *GameScene) string {
	t.Helper()
	var id string
	playUntil(t, s, 60*60*4, "alguien se desplomara", func() bool {
		for _, p := range s.snap.Patients {
			if p.Stage == game.Collapsed && s.anims[p.ID].showsBubble() {
				id = p.ID
				return true
			}
		}
		return false
	})
	return id
}

// NUEVA PARTIDA abre el juego con el motor corriendo; SALIR AL MENÚ lo
// detiene (cancela el context y espera a que la goroutine termine).
func TestTitle_NewGameOpensTheGame(t *testing.T) {
	needFonts(t)
	next, err := choose(t, NewTitleScene(), optNew)
	s, ok := next.(*GameScene)
	if err != nil || !ok {
		t.Fatalf("NUEVA PARTIDA llevó a (%T, %v); se esperaba el juego", next, err)
	}
	start := s.g.Snapshot().Progress
	time.Sleep(400 * time.Millisecond)
	if s.g.Snapshot().Progress == start {
		t.Error("el reloj del turno no avanzó: el motor no está corriendo")
	}
	s.onQuit()
	select {
	case <-s.done:
	default:
		t.Error("después de salir el motor sigue corriendo")
	}
}

// Esc, PAUSA o perder el foco pausan la partida (Game.Pause); al reanudar
// sigue. En la demo, perder el foco no abre la pausa.
func TestGameScene_PausesAndLosingFocusPausesToo(t *testing.T) {
	s := newTestGameScene(t)
	s.onPause()
	if !s.g.Snapshot().Paused {
		t.Error("con la pausa abierta la partida no está en pausa")
	}
	s.onResume()
	if s.g.Snapshot().Paused {
		t.Error("al reanudar la partida sigue en pausa")
	}
	if next, _ := s.update(input{click: true, mouse: pauseButton.Min.Add(image.Pt(5, 5))}); next == Scene(s) {
		t.Error("el botón PAUSA no abrió la pausa")
	}
	s.onResume()

	defer func() { isFocused = func() bool { return true } }()
	isFocused = func() bool { return false }
	app := &App{scene: s}
	app.update(false)
	if _, ok := app.scene.(*PauseScene); !ok || !s.g.Snapshot().Paused {
		t.Errorf("sin el foco la escena es %T y la partida en pausa=%v; se esperaba la pausa", app.scene, s.g.Snapshot().Paused)
	}
	demo := &App{scene: NewDemoScene(game.NewDemo())}
	demo.update(false)
	if _, ok := demo.scene.(*PauseScene); ok {
		t.Error("en la demo perder el foco abrió la pausa; solo va en el juego")
	}
}

// Clic en un paciente desplomado y después en el camillero: Dispatch. El
// camillero llega cuando el juego dice que llegó, lo lleva en camilla a la
// cama, y después el médico, mandado igual, lo revisa. Con clic derecho en
// el revisado, WakeEarly lo despierta antes de tiempo.
func TestGameScene_ClicksDispatchAndWake(t *testing.T) {
	s := newTestGameScene(t)
	id := firstCollapsed(t, s)

	clickOn(s, id, false)
	if s.selected != id {
		t.Fatalf("clic en %s (desplomado): elegido %q", id, s.selected)
	}
	clickOn(s, "C-01", false)
	if st := staffNow(s, "C-01"); st.Activity != game.GoingToPickUp || st.PatientID != id {
		t.Fatalf("después del clic en C-01, está (%v, %q); se esperaba que fuera a recoger a %s", st.Activity, st.PatientID, id)
	}

	// Llega cuando el juego lo dice, ni antes ni después: en el tick en que lo
	// recoge ya está junto al paciente, y no llegó mucho antes a esperar.
	var target vec
	arrived := -1 // tick en que llegó a su destino
	tick := 0
	playUntil(t, s, 60*10, "el camillero lo recogiera", func() bool {
		if stageOf(s, id) == game.AwaitingReview {
			return true
		}
		f, _ := s.figure("C-01")
		target = f.target
		if st, _ := s.staffView("C-01"); st.Activity == game.GoingToPickUp && arrived < 0 && s.pos["C-01"] == target {
			arrived = tick
		}
		tick++
		return false
	})
	if d := math.Hypot(s.pos["C-01"].x-target.x, s.pos["C-01"].y-target.y); d > 12 {
		t.Errorf("cuando el juego lo recoge, C-01 está a %.0f px de donde lo esperaba (%v); se esperaba junto al paciente", d, target)
	}
	if arrived >= 0 && tick-arrived > 30 {
		t.Errorf("C-01 llegó %d ticks antes de que el juego lo recogiera; se esperaba que caminara al ritmo del juego", tick-arrived)
	}

	// La camilla llega a la cama cuando el juego termina el traslado (con un
	// margen de medio segundo).
	playUntil(t, s, 60*10, "el juego terminara el traslado", func() bool { return staffNow(s, "C-01").Activity != game.Carrying })
	late := 0
	for ; late < 60*3 && s.carryOf(id) != nil; late++ {
		play(s, 1)
	}
	if late > 30 {
		t.Errorf("la camilla llegó a la cama %d ticks después de que el juego terminó el traslado; se esperaba a tiempo", late)
	}
	playUntil(t, s, 60*12, "la camilla llegara a la cama", func() bool {
		f, _ := s.figure(id)
		return s.onBed(f) && s.carryOf(id) == nil
	})

	clickOn(s, id, false)
	clickOn(s, "D-01", false)
	if st := staffNow(s, "D-01"); st.Activity != game.GoingToReview {
		t.Fatalf("D-01 está %v; se esperaba que fuera a revisarlo", st.Activity)
	}
	playUntil(t, s, 60*12, "el médico lo revisara", func() bool { return stageOf(s, id) == game.InBed })

	rep := s.snap.Reputation
	clickOn(s, id, true)
	play(s, 1)
	if st := stageOf(s, id); st == game.InBed {
		t.Errorf("con clic derecho %s sigue en cama; se esperaba que se despertara", id)
	}
	if s.snap.Reputation >= rep {
		t.Errorf("despertarlo antes de tiempo no costó reputación (%d → %d)", rep, s.snap.Reputation)
	}
}

// BLOQUE 3 del pase de juice (§12.10): al desplomarse (de verdad, jugando)
// deja la nube de polvo, el aviso "!" y un shake leve.
func TestGameScene_CollapseLeavesDustAndExclaim(t *testing.T) {
	s := newTestGameScene(t)
	id := firstCollapsed(t, s) // espera a que termine de desplomarse (showsBubble)
	if _, ok := s.fx.collapses[id]; !ok {
		t.Errorf("%s se desplomó pero no quedó su aviso (collapseFlourish)", id)
	}
	if len(s.fx.dust) == 0 {
		t.Error("al desplomarse no quedó ninguna partícula de polvo")
	}
}

// Los errores del juego salen como aviso: mandar al médico a recoger a
// alguien cuando hay un camillero libre (§5.5: recoger le toca al camillero).
func TestGameScene_DispatchErrorsBecomeNotices(t *testing.T) {
	s := newTestGameScene(t)
	id := firstCollapsed(t, s)
	clickOn(s, id, false)
	clickOn(s, "D-01", false)
	if st := staffNow(s, "D-01"); st.Activity != game.Free {
		t.Errorf("D-01 salió a recoger (%v) habiendo un camillero libre", st.Activity)
	}
	if s.messageTicks == 0 || !strings.Contains(s.message, "camillero libre") {
		t.Errorf("aviso %q; se esperaba el error del juego", s.message)
	}
}

// En el juego las etiquetas van ocultas (§9.1, confirmado el 2026-10-11): se
// ven al pasar el mouse o con Shift. El punto del personal y las burbujas,
// siempre.
func TestGameScene_LabelsOnlyOnHoverOrShift(t *testing.T) {
	s := newTestGameScene(t)
	f, _ := s.figure("D-01")
	has := func(os []overlay) (label, dot bool) {
		for _, o := range os {
			label = label || o.text == "D-01"
			dot = dot || (o.text == "" && o.rect.Dx() == 8)
		}
		return
	}
	if label, dot := has(s.gameOverlays(f, false)); label || !dot {
		t.Errorf("sin el mouse: etiqueta=%v, punto=%v; se esperaba solo el punto", label, dot)
	}
	if label, _ := has(s.gameOverlays(f, true)); !label {
		t.Error("con el mouse encima no salió la etiqueta")
	}
	b := s.figureBox(f)
	s.update(input{mouse: b.Min.Add(b.Max).Div(2)})
	if s.hover != "D-01" || len(s.tooltipLines(f)) != 3 {
		t.Errorf("con el mouse sobre D-01: hover %q, ficha %q", s.hover, s.tooltipLines(f))
	}
}

// REPORTE muestra las 4 consultas con la partida en pausa y vuelve a ella.
func TestGameScene_ReportPausesAndComesBack(t *testing.T) {
	s := newTestGameScene(t)
	next, _ := s.update(input{click: true, mouse: reportButton.Min.Add(image.Pt(5, 5))})
	r, ok := next.(*ReportScene)
	if !ok || !s.g.Snapshot().Paused {
		t.Fatalf("REPORTE llevó a %T con la partida en pausa=%v", next, s.g.Snapshot().Paused)
	}
	// El reporte abre animado desde el botón REPORTE (§10.10): mientras
	// abre, Espacio no hace nada.
	for i := 0; r.anim.phase != panelOpen; i++ {
		if i > reportOpenTicks {
			t.Fatalf("el reporte no terminó de abrir en %d ticks", reportOpenTicks)
		}
		r.update(input{})
	}
	var back Scene = r
	var err error
	for i := 0; back == Scene(r); i++ {
		if i > reportOpenTicks+reportCloseTicks+1 {
			t.Fatalf("el reporte no terminó de cerrar en %d ticks", reportCloseTicks)
		}
		in := input{}
		if i == 0 {
			in.space = true // arranca el cierre; se termina de cerrar solo
		}
		back, err = r.update(in)
	}
	if err != nil || back != Scene(s) || s.g.Snapshot().Paused {
		t.Errorf("al cerrar el reporte volvió a (%T, %v) con la pausa=%v; se esperaba la partida andando", back, err, s.g.Snapshot().Paused)
	}
}

// En el juego tampoco hay teletransporte: tick a tick nadie se mueve más que
// un poco (el personal camina al ritmo que marca el juego, más rápido que
// en la demo), salvo al aparecer; y los pies de quien camina o está de pie
// no pisan paredes ni muebles. Juega solo: manda a recoger y a revisar a
// quien lo necesite.
//
// La EXCEPCIÓN del giro de golpe (GAME_DESIGN §10.8, 2026-10-15) también
// vale aquí: en el tick en que una camilla gira, el paciente encima salta
// de la huella de antes a la de después.
func TestGameScene_NobodyJumpsOrWalksThroughWalls(t *testing.T) {
	s := newTestGameScene(t)
	const maxStep = 10.0
	before := map[string]vec{}
	for tick := 0; tick < 60*120; tick++ {
		if tick%60 == 0 { // el "jugador": cada segundo despacha lo que pueda
			for _, p := range s.snap.Patients {
				if !needsSomeone(p.Stage) {
					continue
				}
				for _, st := range s.snap.Staff {
					if st.Activity == game.Free && s.g.Dispatch(p.ID, st.ID) == nil {
						break
					}
				}
			}
		}
		prevDirs := map[*carrying]dir{}
		for _, c := range s.allCarries() {
			if c.riding() {
				prevDirs[c] = c.dir
			}
		}
		play(s, 1)
		turning := map[string]bool{} // quien gira de golpe con la camilla en este tick
		for _, c := range s.allCarries() {
			if d, ok := prevDirs[c]; ok && c.dir != d {
				turning[c.order.patient] = true
			}
		}
		present := map[string]bool{}
		for _, f := range s.figures {
			present[f.id] = true
		}
		for id, was := range before { // nadie desaparece en medio del mapa: sale caminando de la pantalla
			if _, still := s.pos[id]; !still && was.x+figWidth/2 < ScreenWidth {
				t.Fatalf("tick %d: %s desapareció en %v, dentro de la pantalla", tick, id, was)
			}
		}
		for _, f := range s.figures {
			now := s.pos[f.id]
			if was, ok := before[f.id]; ok && !turning[f.id] && math.Hypot(now.x-was.x, now.y-was.y) > maxStep {
				t.Fatalf("tick %d: %s se movió %.1f px (de %v a %v)", tick, f.id, math.Hypot(now.x-was.x, now.y-was.y), was, now)
			}
			pa := s.anims[f.id]
			lying := pa != nil && (pa.showsBubble() || pa.anim == animCollapse)
			if s.inCarry(f.id) || lying || (f.patient && f.room != 0) || now.x+figWidth/2 > ScreenWidth || s.actorOf(f.id).exiting {
				continue // en camilla, acostado, en su cama o afuera de la pantalla
			}
			p := image.Pt(int(math.Round(now.x)), int(math.Round(now.y)))
			feet := feetBox.Add(p.Add(feetOffsetPt))
			if pa != nil && pa.room != 0 && feet.Overlaps(bedObstacle(roomZone(pa.room))) {
				// se levantó de su cama y se baja: sus pies pueden ir sobre ella, no sobre nada más
				if count(wallSum, feet) > 0 || onFurnitureBut(feet, bedObstacle(roomZone(pa.room))) {
					t.Fatalf("tick %d: al bajarse de la cama, los pies de %s (%v) tocan una pared u otro mueble", tick, f.id, feet)
				}
				continue
			}
			if touchesWallOrFurniture(feet) {
				t.Fatalf("tick %d: los pies de %s (%v) tocan una pared o un mueble", tick, f.id, feet)
			}
		}
		before = map[string]vec{}
		for id, p := range s.pos {
			before[id] = p
		}
		if s.snap.GameOver != game.NotOver {
			break
		}
	}
}

// El que se va (enojado, a los 45 s sin que nadie lo revise) camina a la
// puerta en el tiempo del juego y, cuando el juego lo quita del mapa, sigue
// caminando por la acera hasta salir de la pantalla: no desaparece en la
// puerta.
func TestGameScene_TheOneWhoLeavesWalksOffTheScreen(t *testing.T) {
	s := newTestGameScene(t)
	id := firstCollapsed(t, s)
	playUntil(t, s, 60*60, "se fuera enojado", func() bool { return stageOf(s, id) == game.Leaving })
	playUntil(t, s, 60*10, "el juego lo quitara del mapa", func() bool { _, ok := s.patientView(id); return !ok })
	if p, ok := s.pos[id]; !ok || p.x+figWidth/2 >= ScreenWidth {
		t.Fatalf("cuando el juego lo quita del mapa, %s está en %v (¿en la escena? %v); se esperaba todavía en la acera", id, p, ok)
	}
	last := s.pos[id]
	for i := 0; i < 60*20; i++ {
		p, ok := s.pos[id]
		if !ok {
			break
		}
		last = p
		play(s, 1)
	}
	if _, ok := s.pos[id]; ok || last.x+figWidth/2 < ScreenWidth-patientSpeed-1 {
		t.Errorf("%s desapareció en %v; se esperaba que saliera por la derecha de la pantalla", id, last)
	}
}
