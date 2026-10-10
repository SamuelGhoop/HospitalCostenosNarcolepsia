package ui

import (
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// stepN avanza la animación n ticks con el mismo movimiento.
func stepN(a *patientAnim, n int, dx, dy float64, inBed bool) {
	for i := 0; i < n; i++ {
		a.step(dx, dy, inBed)
	}
}

// ticksOf: cuánto dura completa una animación que no se repite.
func ticksOf(an animation) int { return animSpecs[an].frames * animSpecs[an].ticksPerFrame }

// Se duerme y le toca cama (pasos 5–7 de la demo): camina hasta donde le
// dio el ataque, se desploma ahí, lo llevan cargado (dormido en el piso, sin
// rotar) y al llegar a la cama queda dormido en cama.
func TestPatientAnim_FallsAsleepAndIsCarriedToBed(t *testing.T) {
	a := &patientAnim{state: hospital.Awake, anim: animIdle}

	a.observe(hospital.AsleepInBed, 101)
	stepN(a, 5, -3, 1, false) // camina hacia la cafetería
	if a.anim != animWalkLeft {
		t.Fatalf("antes del desplome: animación %d; se esperaba caminar", a.anim)
	}

	stepN(a, 1, 0, 0, false) // llegó al sitio del ataque
	if a.anim != animCollapse {
		t.Fatalf("al llegar: animación %d; se esperaba desplomarse", a.anim)
	}

	stepN(a, ticksOf(animCollapse), 0, 0, false) // se desplomó y espera
	stepN(a, 3, 2, -2, false)                    // ya lo van cargando
	if a.anim != animSleepFloor {
		t.Errorf("mientras lo cargan: animación %d; se esperaba dormido en el piso", a.anim)
	}

	stepN(a, 1, 0, 0, true) // llegó a la cama
	if a.anim != animSleepBed {
		t.Errorf("en la cama: animación %d; se esperaba dormido en cama", a.anim)
	}
}

// Se duerme sin cama (paso 8): primero camina hasta el sitio del ataque y
// solo al llegar se desploma.
func TestPatientAnim_WithoutBedWalksFirstThenCollapses(t *testing.T) {
	a := &patientAnim{state: hospital.Awake, anim: animIdle}

	a.observe(hospital.AsleepInHallway, 0)
	stepN(a, 5, 3, 0, false)
	if a.anim != animWalkLeft || !a.flip {
		t.Errorf("caminando a la derecha: (animación %d, volteado %v); se esperaba caminar volteado", a.anim, a.flip)
	}

	stepN(a, 1, 0, 0, false) // llegó al pasillo
	if a.anim != animCollapse {
		t.Fatalf("al llegar: animación %d; se esperaba desplomarse", a.anim)
	}

	stepN(a, ticksOf(animCollapse), 0, 0, false)
	if a.anim != animSleepFloor {
		t.Errorf("después de desplomarse: animación %d; se esperaba dormido en el piso", a.anim)
	}
}

// Si el usuario adelanta la coreografía (Espacio), cada paciente salta a la
// animación de reposo de su estado.
func TestPatientAnim_SettleJumpsToTheRestingAnimation(t *testing.T) {
	inBed := &patientAnim{state: hospital.Awake, anim: animIdle}
	inBed.observe(hospital.AsleepInBed, 101) // iba caminando al sitio del ataque
	inBed.settle(true)
	if inBed.anim != animSleepBed || inBed.pendingCollapse {
		t.Errorf("en cama: (animación %d, pendiente %v); se esperaba dormido en cama", inBed.anim, inBed.pendingCollapse)
	}

	floor := &patientAnim{state: hospital.Awake, anim: animIdle}
	floor.observe(hospital.AsleepInHallway, 0)
	floor.settle(false)
	if floor.anim != animSleepFloor {
		t.Errorf("en el piso: animación %d; se esperaba dormido en el piso", floor.anim)
	}

	waking := &patientAnim{state: hospital.AsleepInBed, room: 101, anim: animSleepBed}
	waking.observe(hospital.Awake, 0)
	waking.settle(false)
	if waking.anim != animWakeUp {
		t.Errorf("despertándose: animación %d; debe dejarlo terminar de despertarse y saludar", waking.anim)
	}
}

// La burbuja Zzz solo aparece cuando ya está acostado: mientras camina al
// sitio del ataque o se está desplomando, todavía no.
func TestPatientAnim_BubbleOnlyWhileLyingAsleep(t *testing.T) {
	a := &patientAnim{state: hospital.Awake, anim: animIdle}
	a.observe(hospital.AsleepInHallway, 0)

	stepN(a, 5, 3, 0, false)
	if a.showsBubble() {
		t.Error("caminando hacia el pasillo no debe llevar burbuja")
	}
	stepN(a, 1, 0, 0, false) // empieza a desplomarse
	if a.showsBubble() {
		t.Error("mientras se desploma no debe llevar burbuja")
	}
	stepN(a, ticksOf(animCollapse), 0, 0, false)
	if !a.showsBubble() {
		t.Error("ya en el piso debe llevar la burbuja")
	}
}

// Pedido por Samuel: en la demo, al despertarse sigue "alta" (saluda) una
// sola vez y luego queda quieto. Nadie se va del hospital en la demo.
func TestPatientAnim_WakesUpThenWavesOnceThenStandsStill(t *testing.T) {
	a := &patientAnim{state: hospital.AsleepInBed, room: 101, anim: animSleepBed}

	leftBed := a.observe(hospital.Awake, 0)
	if !leftBed || a.anim != animWakeUp {
		t.Fatalf("al despertarse: (salió de la cama %v, animación %d); se esperaba (true, despertarse)", leftBed, a.anim)
	}
	if a.room != 101 {
		t.Errorf("debe recordar la cama 101 para dibujar ahí el cuadro 1; recuerda %d", a.room)
	}

	stepN(a, ticksOf(animWakeUp), 0, 0, false)
	if a.anim != animWave {
		t.Errorf("después de despertarse: animación %d; se esperaba alta (saluda)", a.anim)
	}

	stepN(a, ticksOf(animWave), 0, 0, false)
	if a.anim != animIdle {
		t.Errorf("después de saludar: animación %d; se esperaba quieto", a.anim)
	}

	stepN(a, 500, 0, 0, false)
	if a.anim != animIdle {
		t.Errorf("no debe volver a saludar: animación %d", a.anim)
	}
}
