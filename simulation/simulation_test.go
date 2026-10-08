package simulation_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/simulation"
)

// newSimHospital arma un hospital pequeño: 1 doctor, 1 camillero,
// `rooms` habitaciones de capacidad 1 y `patients` pacientes.
func newSimHospital(t *testing.T, rooms, patients int) (*hospital.Hospital, []*hospital.Patient) {
	t.Helper()
	h := hospital.NewHospital("Hospital de simulación")
	if err := h.HireDoctor(hospital.NewDoctor("D-01", "Dra. Karen Ospina", 45, "Neurología del sueño")); err != nil {
		t.Fatal(err)
	}
	if err := h.HireStaff(hospital.NewOrderly("C-01", "Wilmer Camargo", 29)); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= rooms; i++ {
		if err := h.AddRoom(100+i, 1); err != nil {
			t.Fatal(err)
		}
	}
	var list []*hospital.Patient
	for i := 1; i <= patients; i++ {
		p := hospital.NewPatient(fmt.Sprintf("P-%03d", i), fmt.Sprintf("Paciente %d", i), 30, hospital.Severe)
		if err := h.AdmitPatient(p); err != nil {
			t.Fatal(err)
		}
		list = append(list, p)
	}
	return h, list
}

// Corre la simulación con más pacientes que camas y revisa, cuando ya
// terminó (Run espera a todas las goroutines), que el estado sea coherente.
// Con -race, además, cualquier acceso concurrente sin candado haría fallar el test.
func TestRun_KeepsTheHospitalConsistent(t *testing.T) {
	h, patients := newSimHospital(t, 2, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	stats := simulation.Run(ctx, h, patients, []string{"cafetería", "parqueadero", "pasillo 2"})

	if stats.Errors != 0 {
		t.Errorf("hubo %d errores inesperados en la simulación", stats.Errors)
	}
	// Cada paciente espera como máximo 250 ms antes de su primer ataque,
	// así que en 500 ms los 5 tuvieron al menos uno.
	if stats.Episodes < len(patients) {
		t.Errorf("Episodes = %d; se esperaban al menos %d", stats.Episodes, len(patients))
	}
	if got := len(h.History()); got != stats.Episodes {
		t.Errorf("el historial tiene %d episodios pero Stats dice %d", got, stats.Episodes)
	}
	if stats.LeftInHallway > stats.Episodes || stats.LateBeds > stats.LeftInHallway {
		t.Errorf("estadísticas incoherentes: %+v", stats)
	}

	// Ninguna habitación pasó de su capacidad, y cada paciente cumple la
	// invariante cama/estado. Se lee DESPUÉS de Run: ya no hay goroutines.
	for _, r := range h.Rooms() {
		if n := len(r.Occupants()); n > r.Capacity() {
			t.Errorf("la habitación %d tiene %d ocupantes con capacidad %d", r.Number(), n, r.Capacity())
		}
	}
	for _, p := range patients {
		inBed := p.State() == hospital.AsleepInBed
		if inBed != (p.Room() != nil) {
			t.Errorf("%s: estado %v pero Room() = %v", p, p.State(), p.Room())
		}
	}
}

// Run debe terminar poco después de que se cancela el contexto: las
// goroutines no se quedan colgadas esperando.
func TestRun_StopsWhenTheContextIsCancelled(t *testing.T) {
	h, patients := newSimHospital(t, 1, 3)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	simulation.Run(ctx, h, patients, []string{"cafetería"})

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("Run tardó %v en terminar; debía parar poco después de 100 ms", elapsed)
	}
}

func TestRun_WithoutPatientsReturnsImmediately(t *testing.T) {
	h, _ := newSimHospital(t, 1, 0)

	stats := simulation.Run(context.Background(), h, nil, []string{"cafetería"})

	if stats != (simulation.Stats{}) {
		t.Errorf("sin pacientes Stats = %+v; se esperaba todo en cero", stats)
	}
}
