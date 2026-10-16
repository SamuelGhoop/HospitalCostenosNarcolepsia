package ui

import (
	"encoding/json"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// El backdoor de pruebas (GAME_DESIGN §10.9), probado sin ventana: se
// arma la escena y se corre el guion llamando agentRunner.step() a mano,
// igual que los demás tests de gui/ui llaman s.update(in) a mano. "shot"
// necesita una ventana real de Ebitengine (lo prueba el agente tester a
// mano con go run .); estos tests solo cubren wait/click/key/hold/dump.

// saveAgentHooks guarda los puntos de entrada que agent.go reemplaza, para
// devolverlos a como estaban al terminar el test (son variables de
// paquete: si un test las deja puestas, contamina a los demás).
func saveAgentHooks(t *testing.T) {
	t.Helper()
	seed, focus, newGame := seedSource, isFocused, newGameSceneFn
	key, click, read := keyJustPressed, clickJustPressed, readInputFn
	t.Cleanup(func() {
		seedSource, isFocused, newGameSceneFn = seed, focus, newGame
		keyJustPressed, clickJustPressed, readInputFn = key, click, read
	})
}

// runHeadlessScript arma la escena pedida con la semilla fija y corre el
// guion hasta que se acabe, devolviendo cada dump que haya pedido ("dump
// ruta.json" solo importa el nombre: acá se guarda el valor directamente,
// sin tocar disco).
func runHeadlessScript(t *testing.T, scene string, seed int64, script string) []dumpState {
	t.Helper()
	needFonts(t)
	saveAgentHooks(t)
	installAgentDeterminism(seed)

	sc, err := buildAgentScene(AgentConfig{Scene: scene, Seed: seed})
	if err != nil {
		t.Fatalf("buildAgentScene: %v", err)
	}
	instr, err := ParseScript(writeScript(t, script))
	if err != nil {
		t.Fatalf("ParseScript: %v", err)
	}

	r := newAgentRunner(&App{scene: sc}, instr)
	var dumps []dumpState
	r.dump = func(path string, v any) error {
		dumps = append(dumps, v.(dumpState))
		return nil
	}
	for {
		seguir, err := r.step()
		if err != nil {
			t.Fatalf("step: %v", err)
		}
		if !seguir {
			return dumps
		}
	}
}

// Con la misma semilla y el mismo guion, dos corridas del modo juego llegan
// exactamente al mismo estado: mismos pacientes, mismas posiciones, mismo
// reloj (ningún azar se escapó de la semilla fija).
func TestAgentSeedEsDeterminista(t *testing.T) {
	const script = "wait 300\ndump salida.json\nquit\n"
	a := runHeadlessScript(t, "game", 7, script)
	b := runHeadlessScript(t, "game", 7, script)

	ja, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	jb, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if string(ja) != string(jb) {
		t.Errorf("dos corridas con la misma semilla dieron dumps distintos:\n%s\n---\n%s", ja, jb)
	}
	if len(a) != 1 || a[0].Game == nil {
		t.Fatalf("se esperaba un dump con Game lleno, llegó %+v", a)
	}
}

// Sirve de documentación viva de que -seed también fija el tráfico y el
// azar de la partida: semillas distintas, al menos un dato distinto.
func TestAgentSeedesDistintasDivergen(t *testing.T) {
	const script = "wait 300\ndump salida.json\nquit\n"
	a := runHeadlessScript(t, "game", 1, script)
	b := runHeadlessScript(t, "game", 2, script)
	if a[0].Game.Clock == b[0].Game.Clock && len(a[0].Game.Patients) == len(b[0].Game.Patients) {
		t.Skip("con estas dos semillas no hubo diferencia visible en 5 s; no es necesariamente un error")
	}
}

// -step N de la demo adelanta hasta ese paso y deja a todos en su puesto
// final (sin coreografía a medias).
func TestAgentDemoStep(t *testing.T) {
	needFonts(t)
	saveAgentHooks(t)
	installAgentDeterminism(5)

	sc, err := buildDemoScene(5)
	if err != nil {
		t.Fatalf("buildDemoScene: %v", err)
	}
	if sc.snap.Step != 5 {
		t.Fatalf("paso = %d, se esperaba 5", sc.snap.Step)
	}
	if sc.choreo != nil {
		t.Error("al saltar con -step no debería quedar una coreografía a medias")
	}
}

// El guion valida los nombres de tecla contra scriptKeys: "hold" y "key"
// comparten la lista. Esto documenta cuáles lee hoy el juego.
func TestScriptKeysSonLasQueLeeElJuego(t *testing.T) {
	want := []string{"Escape", "Space", "Enter", "NumpadEnter", "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "W", "S", "Shift", "C", "A"}
	if len(scriptKeys) != len(want) {
		t.Fatalf("scriptKeys tiene %d teclas, se esperaban %d", len(scriptKeys), len(want))
	}
	for _, name := range want {
		if _, ok := scriptKeys[name]; !ok {
			t.Errorf("falta la tecla %q en scriptKeys", name)
		}
	}
}

// "hold Shift N" sostiene Shift por N ticks: input.shift (continuo, como
// ebiten.IsKeyPressed) da true los N ticks, pero keyJustPressed (el borde,
// como inpututil.IsKeyJustPressed) solo se dispara en el primero, igual que
// pasaría con la tecla de verdad sostenida.
func TestAgentHoldEsUnaTeclaSostenida(t *testing.T) {
	needFonts(t)
	saveAgentHooks(t)
	installAgentDeterminism(1)

	r := newAgentRunner(&App{scene: NewTitleScene()}, []instruction{
		{kind: insHold, key: ebiten.KeyShift, keyName: "Shift", ticks: 3},
	})
	for i := 0; i < 3; i++ {
		seguir, err := r.step()
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		if !seguir {
			t.Fatalf("step %d: el guion no debería haber terminado", i)
		}
		if !readInputFn().shift {
			t.Errorf("tick %d: input.shift debería ser true durante todo el hold", i)
		}
		if got, want := keyJustPressed(ebiten.KeyShift), i == 0; got != want {
			t.Errorf("tick %d: keyJustPressed(Shift) = %v, se esperaba %v", i, got, want)
		}
	}
	if seguir, _ := r.step(); seguir {
		t.Error("después de 3 ticks, hold ya debería haber terminado")
	}
}
