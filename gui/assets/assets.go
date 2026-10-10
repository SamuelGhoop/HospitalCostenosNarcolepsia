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
