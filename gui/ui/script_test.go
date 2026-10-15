package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// writeScript guarda body como un guion temporal y devuelve su ruta.
func writeScript(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "guion.txt")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// El parser del guion (-input, GAME_DESIGN §10.9): una instrucción por
// línea; las líneas vacías y las que empiezan con # se ignoran.
func TestParseScriptLineas(t *testing.T) {
	tests := []struct {
		name string
		line string
		want instruction
	}{
		{"wait", "wait 30", instruction{kind: insWait, ticks: 30}},
		{"click", "click 100 200", instruction{kind: insClick, x: 100, y: 200}},
		{"rclick", "rclick 5 6", instruction{kind: insRClick, x: 5, y: 6}},
		{"mouse", "mouse 7 8", instruction{kind: insMouse, x: 7, y: 8}},
		{"key", "key Escape", instruction{kind: insKey, key: ebiten.KeyEscape, keyName: "Escape"}},
		{"hold", "hold Shift 12", instruction{kind: insHold, key: ebiten.KeyShift, keyName: "Shift", ticks: 12}},
		{"shot", "shot foto.png", instruction{kind: insShot, path: "foto.png"}},
		{"dump", "dump estado.json", instruction{kind: insDump, path: "estado.json"}},
		{"quit", "quit", instruction{kind: insQuit}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := writeScript(t, tc.line)
			got, err := ParseScript(p)
			if err != nil {
				t.Fatalf("ParseScript: %v", err)
			}
			if len(got) != 1 {
				t.Fatalf("se esperaba 1 instrucción, llegaron %d", len(got))
			}
			g := got[0]
			g.line = 0 // no nos importa el número de línea aquí
			if g != tc.want {
				t.Errorf("ParseScript(%q) = %+v, quiere %+v", tc.line, g, tc.want)
			}
		})
	}
}

// Los comentarios (#) y las líneas vacías se ignoran, y el número de línea
// de un guion con varias instrucciones es el real (para los errores).
func TestParseScriptComentariosYLineas(t *testing.T) {
	p := writeScript(t, "# un comentario\n\nwait 10\nclick 1 2\n")
	got, err := ParseScript(p)
	if err != nil {
		t.Fatalf("ParseScript: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("se esperaban 2 instrucciones, llegaron %d", len(got))
	}
	if got[0].line != 3 || got[1].line != 4 {
		t.Errorf("líneas = %d, %d; quiere 3, 4", got[0].line, got[1].line)
	}
}

// Una tecla desconocida, un número inválido, una instrucción desconocida o
// el número de argumentos que no es el esperado: todos son errores claros
// al cargar el guion, antes de correr nada.
func TestParseScriptErrores(t *testing.T) {
	tests := []struct {
		name, line, wantSubstr string
	}{
		{"tecla desconocida", "key Control", "tecla"},
		{"tecla desconocida en hold", "hold Ctrl 5", "tecla"},
		{"número inválido en wait", "wait abc", "número"},
		{"número inválido en click", "click x 2", "número"},
		{"instrucción desconocida", "salta 3", "instrucción desconocida"},
		{"faltan argumentos en click", "click 1", "argumento"},
		{"sobran argumentos en wait", "wait 1 2", "argumento"},
		{"faltan argumentos en key", "key", "argumento"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := writeScript(t, tc.line)
			_, err := ParseScript(p)
			if err == nil {
				t.Fatalf("ParseScript(%q): se esperaba error", tc.line)
			}
			if !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Errorf("ParseScript(%q) error = %q, se esperaba que contuviera %q", tc.line, err.Error(), tc.wantSubstr)
			}
		})
	}
}

func TestParseScriptArchivoInexistente(t *testing.T) {
	if _, err := ParseScript(filepath.Join(t.TempDir(), "no-existe.txt")); err == nil {
		t.Fatal("se esperaba error con un archivo que no existe")
	}
}
