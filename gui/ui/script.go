package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// El guion del backdoor de pruebas (-input, GAME_DESIGN §10.9): una
// instrucción por línea, para que el agente tester pueda jugar sin una
// persona delante de la pantalla.

// instrKind es qué tipo de instrucción es una línea ya interpretada.
type instrKind int

const (
	insWait   instrKind = iota // wait N        : N ticks de Update sin tocar nada
	insClick                   // click X Y     : mueve el mouse y hace clic izquierdo
	insRClick                  // rclick X Y    : mueve el mouse y hace clic derecho
	insMouse                   // mouse X Y     : solo mueve el mouse
	insKey                     // key Nombre    : esa tecla, apretada un tick
	insHold                    // hold Nombre N : esa tecla, sostenida N ticks
	insShot                    // shot ruta.png : captura la pantalla (1280×720, a 1×)
	insDump                    // dump ruta.json: el estado que dibuja la escena
	insQuit                    // quit          : termina la corrida
)

// instruction es una línea ya interpretada y validada del guion.
type instruction struct {
	kind    instrKind
	ticks   int        // wait N, hold Nombre N
	x, y    int        // click/rclick/mouse X Y
	key     ebiten.Key // key/hold: la tecla de Ebitengine
	keyName string     // key/hold: el nombre tal como vino (para el dump y los errores)
	path    string     // shot/dump: la ruta de salida
	line    int        // número de línea en el guion, para los mensajes de error
}

// scriptKeys son los únicos nombres de tecla válidos: justo las que lee
// alguna escena del juego (input.go, demo_scene.go, app.go). Un nombre que
// no esté acá es un error claro al cargar el guion, antes de correr nada.
var scriptKeys = map[string]ebiten.Key{
	"Escape":      ebiten.KeyEscape,
	"Space":       ebiten.KeySpace,
	"Enter":       ebiten.KeyEnter,
	"NumpadEnter": ebiten.KeyNumpadEnter,
	"ArrowUp":     ebiten.KeyArrowUp,
	"ArrowDown":   ebiten.KeyArrowDown,
	"W":           ebiten.KeyW,
	"S":           ebiten.KeyS,
	"Shift":       ebiten.KeyShift,
	"C":           ebiten.KeyC,
	"A":           ebiten.KeyA,
}

// ParseScript lee el guion de path, una instrucción por línea (las líneas
// vacías y las que empiezan con # se ignoran). Valida todo antes de
// devolver nada: una tecla desconocida o un número inválido es un error con
// el número de línea, para corregir el guion antes de correr la ventana.
func ParseScript(path string) ([]instruction, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("abriendo el guion: %w", err)
	}
	defer f.Close()

	var out []instruction
	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		in, err := parseInstructionLine(line)
		if err != nil {
			return nil, fmt.Errorf("guion, línea %d: %w", lineNo, err)
		}
		in.line = lineNo
		out = append(out, in)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("leyendo el guion: %w", err)
	}
	return out, nil
}

// parseInstructionLine interpreta una línea ya sin espacios de más.
func parseInstructionLine(line string) (instruction, error) {
	fields := strings.Fields(line)
	cmd, args := fields[0], fields[1:]

	needArgs := func(n int) error {
		if len(args) != n {
			return fmt.Errorf("%q necesita %d argumento(s), llegaron %d", cmd, n, len(args))
		}
		return nil
	}
	num := func(s string) (int, error) {
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("%q no es un número válido", s)
		}
		return n, nil
	}
	key := func(name string) (ebiten.Key, error) {
		k, ok := scriptKeys[name]
		if !ok {
			return 0, fmt.Errorf("tecla desconocida: %q", name)
		}
		return k, nil
	}

	switch cmd {
	case "wait":
		if err := needArgs(1); err != nil {
			return instruction{}, err
		}
		n, err := num(args[0])
		if err != nil {
			return instruction{}, err
		}
		return instruction{kind: insWait, ticks: n}, nil

	case "click", "rclick", "mouse":
		if err := needArgs(2); err != nil {
			return instruction{}, err
		}
		x, err := num(args[0])
		if err != nil {
			return instruction{}, err
		}
		y, err := num(args[1])
		if err != nil {
			return instruction{}, err
		}
		kind := map[string]instrKind{"click": insClick, "rclick": insRClick, "mouse": insMouse}[cmd]
		return instruction{kind: kind, x: x, y: y}, nil

	case "key":
		if err := needArgs(1); err != nil {
			return instruction{}, err
		}
		k, err := key(args[0])
		if err != nil {
			return instruction{}, err
		}
		return instruction{kind: insKey, key: k, keyName: args[0]}, nil

	case "hold":
		if err := needArgs(2); err != nil {
			return instruction{}, err
		}
		k, err := key(args[0])
		if err != nil {
			return instruction{}, err
		}
		n, err := num(args[1])
		if err != nil {
			return instruction{}, err
		}
		return instruction{kind: insHold, key: k, keyName: args[0], ticks: n}, nil

	case "shot", "dump":
		if err := needArgs(1); err != nil {
			return instruction{}, err
		}
		kind := insShot
		if cmd == "dump" {
			kind = insDump
		}
		return instruction{kind: kind, path: args[0]}, nil

	case "quit":
		if err := needArgs(0); err != nil {
			return instruction{}, err
		}
		return instruction{kind: insQuit}, nil

	default:
		return instruction{}, fmt.Errorf("instrucción desconocida: %q", cmd)
	}
}
