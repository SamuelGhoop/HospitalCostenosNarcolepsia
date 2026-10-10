package game

import (
	"context"
	"time"
)

// Start lanza la goroutine motor de la partida: cada 100 ms llama Tick con
// 100 ms de tiempo de juego, hasta que se cancela ctx (al salir al menú o
// empezar otra partida).
//
// Avanza un tick fijo en vez de medir el reloj real: si la máquina se
// atrasa, el juego va un poco más lento, pero nunca salta.
//
// Devuelve un canal que se cierra cuando la goroutine termina; los tests lo
// esperan para saber que el motor ya se detuvo, y la interfaz puede ignorarlo.
func (g *Game) Start(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done) // al salir, avisa que el motor terminó

		ticker := time.NewTicker(tickInterval)
		defer ticker.Stop()
		for {
			// Espera SIN el candado: lo toma solo Tick, por un instante.
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				g.Tick(tickInterval)
			}
		}
	}()
	return done
}
