// Package simulation es el bonus de concurrencia: cada paciente vive en su
// propia goroutine y se queda dormido a intervalos aleatorios, mientras el
// hospital lo atiende, le busca cama y lo despierta.
//
// Regla de oro: las goroutines SOLO llaman métodos de hospital.Hospital,
// que son los que tienen el candado (sync.Mutex). Nunca leen ni cambian un
// Patient o un Room directamente. Por eso, para saber si el paciente quedó
// en cama, se le pregunta al hospital (AssignRoom) y no al paciente (State).
// Los getters de las entidades solo se leen cuando Run ya terminó.
package simulation

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// Tiempos de la simulación: cortos, para que en un par de segundos pasen
// varios ataques y alguien se quede sin cama.
const (
	minAwake  = 50 * time.Millisecond // despierto entre minAwake y maxAwake
	maxAwake  = 250 * time.Millisecond
	minAsleep = 100 * time.Millisecond // dormido entre minAsleep y maxAsleep
	maxAsleep = 400 * time.Millisecond
	retryBed  = 30 * time.Millisecond // cada cuánto se reintenta cama desde el pasillo
)

// Stats resume lo que pasó durante la simulación.
type Stats struct {
	Episodes      int // ataques registrados en el hospital
	LeftInHallway int // ataques en los que no había cama al dormirse
	LateBeds      int // camas conseguidas después, reintentando desde el pasillo
	WakeUps       int // veces que alguien se despertó
	Errors        int // errores inesperados (deben ser 0)
}

// counters son los contadores compartidos por todas las goroutines.
// atomic.Int64 permite sumar desde varias goroutines a la vez sin candado:
// Add(1) es una operación indivisible.
type counters struct {
	episodes, leftInHallway, lateBeds, wakeUps, errors atomic.Int64
}

// Run lanza una goroutine por paciente y espera a que todas terminen, lo
// que pasa cuando ctx se cancela (por ejemplo, con context.WithTimeout).
// Cuando Run devuelve, ya no queda ninguna goroutine escribiendo: es seguro
// leer los getters del hospital y de sus entidades.
func Run(ctx context.Context, h *hospital.Hospital, patients []*hospital.Patient, locations []string) Stats {
	var c counters
	var wg sync.WaitGroup // cuenta cuántas goroutines siguen vivas

	for _, p := range patients {
		p := p // copia explícita: en Go 1.21 la variable del for es la misma en cada vuelta
		wg.Add(1)
		go func() {
			defer wg.Done() // al terminar, avisa que esta goroutine acabó
			livePatient(ctx, h, p, locations, &c)
		}()
	}
	wg.Wait() // bloquea hasta que todas llamen Done

	return Stats{
		Episodes:      int(c.episodes.Load()),
		LeftInHallway: int(c.leftInHallway.Load()),
		LateBeds:      int(c.lateBeds.Load()),
		WakeUps:       int(c.wakeUps.Load()),
		Errors:        int(c.errors.Load()),
	}
}

// livePatient es la "vida" de un paciente, en ciclos de
// despierto → ataque → dormido (con o sin cama) → despierta.
// Termina cuando se cancela ctx o si algo falla de forma inesperada.
func livePatient(ctx context.Context, h *hospital.Hospital, p *hospital.Patient, locations []string, c *counters) {
	for {
		// 1. Despierto un rato.
		if !wait(ctx, randomBetween(minAwake, maxAwake)) {
			return
		}

		// 2. Ataque de sueño en un lugar al azar. El hospital despacha
		//    personal y, si hay, le da cama.
		location := locations[rand.Intn(len(locations))]
		if err := h.RegisterEpisode(p, location); err != nil {
			c.errors.Add(1)
			return
		}
		c.episodes.Add(1)

		// 3. Dormido un rato; si quedó en el pasillo, se le sigue buscando cama.
		if !sleepAndRetryBed(ctx, h, p, c) {
			return
		}

		// 4. Se despierta y libera la cama, si tenía.
		if err := h.WakePatient(p); err != nil {
			c.errors.Add(1)
			return
		}
		c.wakeUps.Add(1)
	}
}

// sleepAndRetryBed mantiene dormido al paciente un tiempo al azar. Si quedó
// en el pasillo, cada retryBed le pide cama al hospital: compite con las
// demás goroutines por las camas que se van liberando. Devuelve false si ctx
// se canceló.
func sleepAndRetryBed(ctx context.Context, h *hospital.Hospital, p *hospital.Patient, c *counters) bool {
	napEnd := time.Now().Add(randomBetween(minAsleep, maxAsleep))

	// ¿Quedó en cama? Se le pregunta al hospital: si RegisterEpisode ya le
	// dio una, AssignRoom responde ErrAlreadyInBed.
	_, err := h.AssignRoom(p)
	inBed := true
	switch {
	case errors.Is(err, hospital.ErrAlreadyInBed):
		// RegisterEpisode ya lo acostó.
	case err == nil:
		// Quedó en el pasillo, pero justo se liberó una cama.
		c.leftInHallway.Add(1)
		c.lateBeds.Add(1)
	case errors.Is(err, hospital.ErrNoRoomAvailable):
		c.leftInHallway.Add(1)
		inBed = false
	default:
		c.errors.Add(1)
		return false
	}

	for !inBed && time.Now().Before(napEnd) {
		if !wait(ctx, retryBed) {
			return false
		}
		_, err := h.AssignRoom(p)
		switch {
		case err == nil:
			c.lateBeds.Add(1)
			inBed = true
		case !errors.Is(err, hospital.ErrNoRoomAvailable):
			c.errors.Add(1)
			return false
		}
	}
	return wait(ctx, time.Until(napEnd)) // lo que le quede de siesta
}

// wait espera d, o menos si ctx se cancela antes. Devuelve false si se
// canceló. select se queda esperando el primero de los dos canales que
// reciba algo: el temporizador o la cancelación.
func wait(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// randomBetween devuelve una duración al azar en [min, max).
func randomBetween(min, max time.Duration) time.Duration {
	return min + time.Duration(rand.Int63n(int64(max-min)))
}
