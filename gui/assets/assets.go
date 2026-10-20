// Package assets mete los recursos gráficos (sprites, el mapa, más adelante
// fuentes y sonidos) DENTRO del ejecutable, con //go:embed. Así el juego no depende
// de encontrar archivos sueltos al correr.
//
// Va en su propio paquete porque //go:embed solo puede ver archivos de su
// misma carpeta o de las de abajo: desde gui/ui no se puede llegar a
// gui/assets.
package assets

import "embed"

// Sprites tiene todas las hojas de sprites PNG de la carpeta sprites/.
//
//go:embed sprites/*.png
var Sprites embed.FS

// Map tiene el mapa de fondo y lo que se dibuja encima: la cobija, el taxi y
// el humo de la olla (GAME_DESIGN §10.7).
//
//go:embed map/*.png
var Map embed.FS

// Title tiene la portada: las capas del paisaje, el portapapeles del menú
// (una imagen por opción y cuadro del cursor) y sus zonas para el mouse
// (gui/assets/title/TITLE_ASSETS.md).
//
//go:embed title/*.png title/clipboard
var Title embed.FS

// Fonts tiene Press Start 2P (licencia OFL, en fonts/OFL.txt).
//
//go:embed fonts/PressStart2P-Regular.ttf
var Fonts embed.FS

// Pause tiene el menú de pausa: el portapapeles (una imagen por opción y
// cuadro del cursor) y sus renglones para el mouse
// (gui/assets/pause/PAUSE_ASSETS.md).
//
//go:embed pause/*.png pause/hitboxes.json
var Pause embed.FS

// Hiring tiene las imágenes de la bolsa de empleo y el portapapeles chiquito
// del HUD (gui/assets/hiring/HIRING_ASSETS.md).
//
//go:embed hiring/*.png
var Hiring embed.FS

// Report tiene el tablero de corcho y las chinchetas del Reporte del día
// (gui/assets/report/REPORT_ASSETS.md).
//
//go:embed report/*.png
var Report embed.FS

// GameOver tiene los tres íconos del motivo de derrota (gui/assets/gameover/GAMEOVER_ASSETS.md).
// El portapapeles de este panel reusa hiring_board.png (assets.Hiring): no
// hay un PNG propio, como ya avisa GAMEOVER_ASSETS.md.
//
//go:embed gameover/*.png
var GameOver embed.FS
