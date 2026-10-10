package game

// Este archivo termina en _test.go, así que SOLO se compila al correr los
// tests: lo que define aquí no es parte de la API del paquete. Es la forma
// idiomática de Go para que los tests de caja negra (package game_test)
// preparen situaciones que el juego todavía no produce solo.

import "github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"

// CollapseForTest admite un paciente nuevo y lo deja desplomado en location,
// como si le hubiera dado el ataque. (Desde F1.3 los pacientes llegan y se
// desploman solos con Tick.)
func (g *Game) CollapseForTest(id, name string, level hospital.NarcolepsyLevel, location string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	p := hospital.NewPatient(id, name, 30, level)
	if err := g.h.AdmitPatient(p); err != nil {
		return err
	}
	g.patients = append(g.patients, &patient{p: p, stage: Collapsed, location: location})
	return nil
}
