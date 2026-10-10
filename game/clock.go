package game

import (
	"fmt"
	"time"
)

// clock es el reloj del turno: 12 horas de juego (08:00–20:00) que pasan en
// shiftDuration de tiempo real. Solo avanza cuando Game.Tick lo llama.
type clock struct {
	day     int           // día de juego, desde 1
	elapsed time.Duration // tiempo transcurrido en el turno (0 a shiftDuration)
}

// newClock arranca el día 1 a las 08:00.
func newClock() clock { return clock{day: 1} }

// advance suma dt al turno, sin pasar de las 20:00.
func (c *clock) advance(dt time.Duration) {
	c.elapsed += dt
	if c.elapsed > shiftDuration {
		c.elapsed = shiftDuration
	}
}

// over dice si el turno ya llegó a las 20:00.
func (c clock) over() bool { return c.elapsed >= shiftDuration }

// minute es la hora de juego en minutos desde la medianoche.
//
// Se calcula con aritmética ENTERA de time.Duration (que por dentro es un
// int64 de nanosegundos), no con float64: la división entera trunca, así que
// el reloj nunca redondea hacia arriba (a los 299,9 s marca 19:59, no 20:00).
func (c clock) minute() int {
	shiftMinutes := time.Duration(shiftEnd - shiftStart) // 720 minutos de juego
	return shiftStart + int(c.elapsed*shiftMinutes/shiftDuration)
}

// String muestra la hora de juego como "14:15".
func (c clock) String() string {
	m := c.minute()
	return fmt.Sprintf("%02d:%02d", m/60, m%60)
}

// progress es cuánto va del turno, de 0 a 1 (para la barra del HUD).
func (c clock) progress() float64 {
	return float64(c.elapsed) / float64(shiftDuration)
}

// nextDay pasa al día siguiente, otra vez a las 08:00.
func (c *clock) nextDay() {
	c.day++
	c.elapsed = 0
}
