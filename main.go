// Programa de demostración del Hospital de los Costeños con Narcolepsia.
//
// main SOLO arma el escenario de la Sección 6 del enunciado e imprime los
// resultados. Toda la lógica (quién atiende, qué cama se asigna, qué cuenta
// el reporte) vive en el paquete hospital; aquí no se decide nada.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/simulation"
)

// simulationTime es lo que dura la simulación concurrente del bonus.
const simulationTime = 2 * time.Second

func main() {
	// ── Paso 1: hospital, 2 doctores, 1 camillero, 3 habitaciones, 5 pacientes ──
	h := hospital.NewHospital("Hospital de los Costeños con Narcolepsia")

	karen := hospital.NewDoctor("D-01", "Dra. Karen Ospina", 45, "Neurología del sueño")
	efrain := hospital.NewDoctor("D-02", "Dr. Efraín Barraza", 52, "Medicina interna")
	wilmer := hospital.NewOrderly("C-01", "Wilmer Camargo", 29)
	check(h.HireDoctor(karen))
	check(h.HireDoctor(efrain))
	check(h.HireStaff(wilmer))

	for _, number := range []int{101, 102, 103} {
		check(h.AddRoom(number, 1))
	}

	yeimy := hospital.NewPatient("P-001", "Yeimy Padilla", 34, hospital.Severe)
	kevin := hospital.NewPatient("P-002", "Kevin Mercado", 27, hospital.Moderate)
	ludys := hospital.NewPatient("P-003", "Ludys Arrieta", 58, hospital.Mild)
	wilfrido := hospital.NewPatient("P-004", "Wilfrido Berrío", 61, hospital.Severe)
	breiner := hospital.NewPatient("P-005", "Breiner Julio", 22, hospital.Severe)
	for _, p := range []*hospital.Patient{yeimy, kevin, ludys, wilfrido, breiner} {
		check(h.AdmitPatient(p))
	}

	// Cada paciente queda a cargo de un médico tratante.
	for _, p := range []*hospital.Patient{yeimy, kevin, breiner} {
		check(karen.DiagnosePatient(p))
	}
	for _, p := range []*hospital.Patient{ludys, wilfrido} {
		check(efrain.DiagnosePatient(p))
	}

	printSetup(h)

	// ── Paso 2: cuatro ataques de sueño en lugares distintos ──
	section("Ataques de sueño")
	attacks := []struct {
		patient  *hospital.Patient
		location string
	}{
		{yeimy, "cafetería, después del sancocho"},
		{kevin, "fila de radiología"},
		{ludys, "pista de champeta del patio"},
		{wilfrido, "pasillo 2, segundo piso"},
	}
	for _, a := range attacks {
		check(h.RegisterEpisode(a.patient, a.location))
		printPatientStatus(a.patient)
	}

	// El cuarto no consiguió cama: se vuelve a pedir y se imprime el error.
	section("Consulta 5.3 — Asignar cama a " + wilfrido.String())
	printAssignRoom(h, wilfrido)

	section("Consulta 5.1 — Pacientes dormidos en el pasillo (antes de liberar camas)")
	printHallway(h.PatientsInHallway())

	// ── Paso 3: alguien se despierta, libera su cama y se la dan al del pasillo ──
	section("Se despierta " + yeimy.String())
	check(h.WakePatient(yeimy))
	printPatientStatus(yeimy)

	section("Consulta 5.3 — Asignar cama a " + wilfrido.String() + " (otra vez)")
	printAssignRoom(h, wilfrido)

	// ── Paso 4: resultado final de las cuatro consultas ──
	fmt.Println()
	fmt.Println("════════════════ RESULTADO FINAL DE LAS CUATRO CONSULTAS ════════════════")

	section("Consulta 5.1 — Pacientes dormidos en el pasillo")
	printHallway(h.PatientsInHallway())

	for _, d := range h.Doctors() {
		section("Consulta 5.2 — Episodios atendidos por " + d.Name())
		printEpisodes(d.MyEpisodes())
	}

	section("Consulta 5.3 — Estado de las habitaciones")
	printRooms(h.Rooms())

	section("Consulta 5.4 — Reporte de severidad (pacientes Severe)")
	printSevereReport(h.SevereReport())

	// Extra: el historial completo muestra el despacho polimórfico en acción.
	// El episodio que atendió el camillero no sale en ninguna consulta 5.2.
	section("Extra — Historial completo (quién atendió cada episodio)")
	printEpisodes(h.History())

	runConcurrentSimulation()
}

// runConcurrentSimulation es el bonus: un hospital nuevo, igual al del
// escenario, donde cada paciente es una goroutine que se duerme a ratos.
// Se corre limpio con: go run -race .
func runConcurrentSimulation() {
	fmt.Println()
	fmt.Printf("════════════ BONUS — SIMULACIÓN CONCURRENTE (%v, una goroutine por paciente) ════════════\n", simulationTime)

	h := hospital.NewHospital("Hospital de los Costeños con Narcolepsia (simulación)")
	check(h.HireDoctor(hospital.NewDoctor("D-01", "Dra. Karen Ospina", 45, "Neurología del sueño")))
	check(h.HireDoctor(hospital.NewDoctor("D-02", "Dr. Efraín Barraza", 52, "Medicina interna")))
	check(h.HireStaff(hospital.NewOrderly("C-01", "Wilmer Camargo", 29)))
	for _, number := range []int{101, 102, 103} {
		check(h.AddRoom(number, 1))
	}
	patients := []*hospital.Patient{
		hospital.NewPatient("P-001", "Yeimy Padilla", 34, hospital.Severe),
		hospital.NewPatient("P-002", "Kevin Mercado", 27, hospital.Moderate),
		hospital.NewPatient("P-003", "Ludys Arrieta", 58, hospital.Mild),
		hospital.NewPatient("P-004", "Wilfrido Berrío", 61, hospital.Severe),
		hospital.NewPatient("P-005", "Breiner Julio", 22, hospital.Severe),
	}
	for _, p := range patients {
		check(h.AdmitPatient(p))
	}
	locations := []string{"cafetería", "fila de radiología", "pista de champeta", "pasillo 2", "parqueadero"}

	// context.WithTimeout cancela la simulación sola cuando pasa el tiempo.
	ctx, cancel := context.WithTimeout(context.Background(), simulationTime)
	defer cancel()
	stats := simulation.Run(ctx, h, patients, locations)

	// Run ya esperó a todas las goroutines (wg.Wait): desde aquí es seguro
	// leer el hospital y sus pacientes.
	section("Lo que pasó")
	fmt.Printf("  %d ataques de sueño, %d despertares\n", stats.Episodes, stats.WakeUps)
	fmt.Printf("  %d veces alguien se durmió sin cama libre; %d consiguieron cama después\n",
		stats.LeftInHallway, stats.LateBeds)
	fmt.Printf("  %d errores inesperados\n", stats.Errors)

	section("Consulta 5.1 al final de la simulación")
	printHallway(h.PatientsInHallway())

	section("Consulta 5.3 al final de la simulación — habitaciones")
	printRooms(h.Rooms())

	section("Consulta 5.4 al final de la simulación")
	printSevereReport(h.SevereReport())
}

// ─── Ayudantes de impresión ──────────────────────────────────────────────
// Solo formatean lo que devuelve el paquete hospital; no toman decisiones.

// check detiene el programa si falla un paso que en este escenario no
// debería fallar (contratar, admitir, registrar…).
func check(err error) {
	if err != nil {
		fmt.Println("error inesperado en el escenario:", err)
		os.Exit(1)
	}
}

// section imprime un encabezado para que cada consulta se identifique.
func section(title string) {
	fmt.Printf("\n== %s ==\n", title)
}

func printSetup(h *hospital.Hospital) {
	fmt.Println("=== " + h.Name() + " ===")

	// Staff() devuelve []Attender: doctores y camillero en la misma lista.
	fmt.Println("Personal:")
	for _, a := range h.Staff() {
		fmt.Printf("  %-5s %s\n", a.ID(), a.Name())
	}

	fmt.Println("Habitaciones:")
	for _, r := range h.Rooms() {
		fmt.Printf("  %d (capacidad %d, %v)\n", r.Number(), r.Capacity(), r.State())
	}

	fmt.Println("Pacientes:")
	for _, p := range h.Patients() {
		treating := "sin tratante"
		if d := p.AssignedDoctor(); d != nil { // nil si nadie lo ha diagnosticado
			treating = d.Name()
		}
		fmt.Printf("  %-26s narcolepsia %-8v tratante: %s\n", p, p.Level(), treating)
	}
}

// printPatientStatus muestra dónde quedó un paciente y en qué estado.
func printPatientStatus(p *hospital.Patient) {
	fmt.Printf("  %-26s %-22v %s\n", p, p.State(), p.Location())
}

// printAssignRoom llama la consulta 5.3 y muestra el resultado o el error.
func printAssignRoom(h *hospital.Hospital, p *hospital.Patient) {
	room, err := h.AssignRoom(p)
	switch {
	case errors.Is(err, hospital.ErrNoRoomAvailable):
		fmt.Println("  ✗", err) // error previsto: se imprime y el programa sigue
	case err != nil:
		check(err)
	default:
		fmt.Printf("  ✓ %s → habitación %d\n", p, room.Number())
	}
}

func printHallway(patients []*hospital.Patient) {
	if len(patients) == 0 {
		fmt.Println("  (nadie: todos los dormidos tienen cama)")
		return
	}
	for _, p := range patients {
		fmt.Printf("  %-26s (%s)\n", p, p.Location())
	}
}

func printEpisodes(episodes []hospital.EpisodeRecord) {
	if len(episodes) == 0 {
		fmt.Println("  (ninguno)")
		return
	}
	for _, e := range episodes {
		fmt.Println("  " + e.Summary())
	}
}

func printRooms(rooms []*hospital.Room) {
	for _, r := range rooms {
		occupants := "vacía"
		if list := r.Occupants(); len(list) > 0 {
			occupants = list[0].String()
			for _, p := range list[1:] {
				occupants += ", " + p.String()
			}
		}
		fmt.Printf("  %d  %-11v %s\n", r.Number(), r.State(), occupants)
	}
}

// printSevereReport ordena el reporte por ID antes de imprimir: un map de Go
// no tiene orden, y sin ordenar la salida cambiaría en cada ejecución.
func printSevereReport(report map[*hospital.Patient]int) {
	patients := make([]*hospital.Patient, 0, len(report))
	for p := range report {
		patients = append(patients, p)
	}
	sort.Slice(patients, func(i, j int) bool {
		return patients[i].ID() < patients[j].ID()
	})

	for _, p := range patients {
		fmt.Printf("  %-26s %d episodio(s) hoy\n", p, report[p])
	}
}
