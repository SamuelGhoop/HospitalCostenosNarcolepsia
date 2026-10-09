// Package assets mete los recursos gráficos (sprites, más adelante fuentes
// y sonidos) DENTRO del ejecutable, con //go:embed. Así el juego no depende
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
