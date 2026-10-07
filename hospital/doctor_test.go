package hospital_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

func newTestDoctor(t *testing.T, id string) *hospital.Doctor {
	t.Helper()
	return hospital.NewDoctor(id, "Dra. "+id, 45, "Neurología")
}

func TestNewDoctor_StartsWithoutPatientsNorEpisodes(t *testing.T) {
	d := hospital.NewDoctor("D-01", "Dra. Karen Ospina", 45, "Neurología del sueño")

	// ID y Name son métodos promovidos desde Person.
	if d.ID() != "D-01" || d.Name() != "Dra. Karen Ospina" || d.Specialty() != "Neurología del sueño" {
		t.Errorf("doctor = (%q, %q, %q)", d.ID(), d.Name(), d.Specialty())
	}
	if len(d.Patients()) != 0 || len(d.MyEpisodes()) != 0 {
		t.Errorf("doctor nuevo con %d pacientes y %d episodios; se esperaba 0 y 0",
			len(d.Patients()), len(d.MyEpisodes()))
	}
	if !d.IsAvailable() {
		t.Error("un doctor sin pacientes debe estar disponible")
	}
}

func TestDoctor_DiagnosePatientBecomesHisTreatingDoctor(t *testing.T) {
	d := newTestDoctor(t, "D-01")
	p := newTestPatient(t, "P-001", hospital.Severe)

	if err := d.DiagnosePatient(p); err != nil {
		t.Fatalf("DiagnosePatient devolvió error: %v", err)
	}

	if p.AssignedDoctor() != d {
		t.Errorf("AssignedDoctor() = %v; se esperaba %v", p.AssignedDoctor(), d)
	}
	if got := d.Patients(); len(got) != 1 || got[0] != p {
		t.Errorf("Patients() = %v; se esperaba [%v]", got, p)
	}
}

func TestDoctor_DiagnoseSamePatientTwiceDoesNotDuplicate(t *testing.T) {
	d := newTestDoctor(t, "D-01")
	p := newTestPatient(t, "P-001", hospital.Severe)

	for i := 0; i < 2; i++ {
		if err := d.DiagnosePatient(p); err != nil {
			t.Fatalf("diagnóstico %d devolvió error: %v", i+1, err)
		}
	}

	if n := len(d.Patients()); n != 1 {
		t.Errorf("el doctor tiene %d pacientes; se esperaba 1", n)
	}
}

func TestDoctor_DiagnosePatientOfAnotherDoctorFails(t *testing.T) {
	first := newTestDoctor(t, "D-01")
	second := newTestDoctor(t, "D-02")
	p := newTestPatient(t, "P-001", hospital.Severe)
	if err := first.DiagnosePatient(p); err != nil {
		t.Fatal(err)
	}

	err := second.DiagnosePatient(p)

	if !errors.Is(err, hospital.ErrAlreadyHasDoctor) {
		t.Fatalf("err = %v; se esperaba ErrAlreadyHasDoctor", err)
	}
	if p.AssignedDoctor() != first || len(second.Patients()) != 0 {
		t.Error("un diagnóstico fallido no debe cambiar el médico tratante")
	}
}

func TestDoctor_DiagnoseNilPatientFails(t *testing.T) {
	if err := newTestDoctor(t, "D-01").DiagnosePatient(nil); !errors.Is(err, hospital.ErrNilPatient) {
		t.Fatalf("err = %v; se esperaba ErrNilPatient", err)
	}
}

func TestDoctor_FullDoctorIsNotAvailableAndRejectsNewPatients(t *testing.T) {
	d := newTestDoctor(t, "D-01")
	for i := 1; i <= 4; i++ {
		if err := d.DiagnosePatient(newTestPatient(t, fmt.Sprintf("P-%03d", i), hospital.Mild)); err != nil {
			t.Fatalf("paciente %d: %v", i, err)
		}
	}

	if d.IsAvailable() {
		t.Error("con 4 pacientes a cargo el doctor no debe estar disponible")
	}
	extra := newTestPatient(t, "P-005", hospital.Mild)
	if err := d.DiagnosePatient(extra); !errors.Is(err, hospital.ErrDoctorFull) {
		t.Fatalf("err = %v; se esperaba ErrDoctorFull", err)
	}
	if extra.AssignedDoctor() != nil {
		t.Error("el paciente rechazado no debe quedar con médico tratante")
	}
}

func TestDoctor_AttendSleepEmergencyRequiresSleepingPatient(t *testing.T) {
	d := newTestDoctor(t, "D-01")

	if err := d.AttendSleepEmergency(newTestPatient(t, "P-001", hospital.Mild)); !errors.Is(err, hospital.ErrNotAsleep) {
		t.Errorf("paciente despierto: err = %v; se esperaba ErrNotAsleep", err)
	}
	if err := d.AttendSleepEmergency(nil); !errors.Is(err, hospital.ErrNilPatient) {
		t.Errorf("paciente nil: err = %v; se esperaba ErrNilPatient", err)
	}
	if err := d.AttendSleepEmergency(asleepPatient(t, "P-002")); err != nil {
		t.Errorf("paciente dormido: err = %v; se esperaba nil", err)
	}
}

func TestDoctor_AttendStoresTheEpisodeInMyEpisodes(t *testing.T) {
	d := newTestDoctor(t, "D-01")
	p := asleepPatient(t, "P-001")

	rec, err := d.Attend(p, "pasillo 2")
	if err != nil {
		t.Fatalf("Attend devolvió error: %v", err)
	}

	mine := d.MyEpisodes()
	if len(mine) != 1 || mine[0].ID() != rec.ID() {
		t.Fatalf("MyEpisodes() = %v; se esperaba solo %s", mine, rec.ID())
	}
	// Atender una emergencia NO es lo mismo que tomarlo a cargo (eso es DiagnosePatient).
	if len(d.Patients()) != 0 {
		t.Error("Attend no debe agregar al paciente a cargo del doctor")
	}
}

func TestDoctor_AttendAwakePatientFailsAndStoresNothing(t *testing.T) {
	d := newTestDoctor(t, "D-01")

	_, err := d.Attend(newTestPatient(t, "P-001", hospital.Mild), "cafetería")

	if !errors.Is(err, hospital.ErrNotAsleep) {
		t.Fatalf("err = %v; se esperaba ErrNotAsleep", err)
	}
	if len(d.MyEpisodes()) != 0 {
		t.Error("una atención fallida no debe quedar en MyEpisodes")
	}
}

func TestDoctor_MyEpisodesReturnsACopy(t *testing.T) {
	d := newTestDoctor(t, "D-01")
	if _, err := d.Attend(asleepPatient(t, "P-001"), "pasillo 2"); err != nil {
		t.Fatal(err)
	}

	list := d.MyEpisodes()
	list[0] = hospital.EpisodeRecord{} // alguien de afuera intenta borrar el registro

	if d.MyEpisodes()[0].ID() == "" {
		t.Error("modificar el slice devuelto cambió el historial del doctor; MyEpisodes() debe devolver una copia")
	}
}
