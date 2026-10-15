package ui

import "time"

// seedSource da la semilla del azar de una partida o demostración nueva, y
// la del tráfico decorado de la calle. En producción es la hora (cada
// partida se ve distinta); el backdoor de pruebas (-agent -seed N, ver
// agent.go) la reemplaza por una fija para que una corrida se pueda repetir.
var seedSource = func() int64 { return time.Now().UnixNano() }
