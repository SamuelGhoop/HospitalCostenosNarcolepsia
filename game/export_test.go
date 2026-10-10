package game

// Este archivo termina en _test.go, así que SOLO se compila al correr los
// tests: lo que define aquí no es parte de la API del paquete. Es la forma
// idiomática de Go para que los tests de caja negra (package game_test)
// preparen situaciones exactas sin esperar al azar.

import "github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"

// ArriveForTest pone en la calle a un paciente nuevo, como una llegada real
// pero sin esperar el intervalo. Devuelve el ID que le puso el juego.
func (g *Game) ArriveForTest(name string, level hospital.NarcolepsyLevel) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.addArrivalLocked(name, 30, level).p.ID()
}

// CollapseForTest admite un paciente nuevo y lo deja desplomado en zone,
// como si le hubiera dado el ataque, sin esperar a que llegue y se duerma
// solo. Devuelve el ID que le puso el juego: es el siguiente de la partida,
// así no choca con los pacientes que llegan por la calle.
func (g *Game) CollapseForTest(name string, level hospital.NarcolepsyLevel, zone Zone) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	p := hospital.NewPatient(g.nextPatientIDLocked(), name, 30, level)
	if err := g.h.AdmitPatient(p); err != nil {
		return "", err
	}
	g.patients = append(g.patients, &patient{p: p, stage: Collapsed, zone: zone})
	return p.ID(), nil
}
