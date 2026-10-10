package game_test

import (
	"context"
	"sync"
	"testing"
	"time"
)

// El motor llama Tick solo (cada 100 ms) y se detiene cuando se cancela el
// contexto de la partida.
func TestStart_EngineAdvancesTheClockAndStopsWithTheContext(t *testing.T) {
	g := newGame(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := g.Start(ctx)
	time.Sleep(350 * time.Millisecond) // en los tests sí se puede esperar con Sleep
	if g.Snapshot().Progress == 0 {
		t.Fatal("con el motor corriendo, el reloj debe avanzar")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("el motor no se detuvo en 1 s después de cancelar el contexto")
	}

	before := g.Snapshot().Clock
	progress := g.Snapshot().Progress
	time.Sleep(250 * time.Millisecond)
	if after := g.Snapshot(); after.Progress != progress || after.Clock != before {
		t.Errorf("después de detener el motor el reloj siguió avanzando: %v → %v", progress, after.Progress)
	}
}

// Varias goroutines tocan la partida a la vez, como en el juego real: el
// motor (Tick), la interfaz (lee Snapshot sin parar) y el jugador (pausa y
// reanuda de vez en cuando). Con -race (en Docker) no debe haber carreras.
//
// La que lee NO pausa: si una misma goroutine leyera y tomara el candado
// todo el tiempo, quedaría sincronizada con el motor y una carrera real
// (por ejemplo, Snapshot sin candado) pasaría desapercibida.
func TestStart_SnapshotsWhileTheEngineRunsAreRaceFree(t *testing.T) {
	g := newGame(t)
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()

	done := g.Start(ctx)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { // la interfaz
		defer wg.Done()
		for ctx.Err() == nil {
			_ = g.Snapshot()
		}
	}()
	go func() { // el jugador
		defer wg.Done()
		for ctx.Err() == nil {
			g.Pause()
			time.Sleep(time.Millisecond)
			g.Resume()
			time.Sleep(20 * time.Millisecond)
		}
	}()
	wg.Wait()
	<-done

	if snap := g.Snapshot(); snap.Day != 1 || len(snap.Rooms) != 3 {
		t.Errorf("la partida quedó en un estado raro: %+v", snap)
	}
}
