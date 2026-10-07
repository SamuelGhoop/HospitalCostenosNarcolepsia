package hospital

// Person agrupa los datos de identidad que comparten pacientes y personal.
//
// En Go no hay herencia: en vez de "Patient extends Person", Patient, Doctor
// y Orderly EMBEBEN un Person (composición). Así los tres campos se escriben
// una sola vez y los métodos de Person se "promueven": se puede llamar
// paciente.Name() como si Name fuera un método propio de Patient.
//
// Los campos van en minúscula (no exportados): desde fuera del paquete
// hospital nadie puede leerlos ni cambiarlos directamente; solo a través de
// los métodos ID, Name y Age. Eso es el encapsulamiento en Go.
type Person struct {
	id   string
	name string
	age  int
}

// NewPerson crea los datos de identidad de una persona. Es el "constructor":
// Go no tiene constructores, por convención se usa una función NewX.
func NewPerson(id, name string, age int) Person {
	return Person{id: id, name: name, age: age}
}

// Los métodos de Person usan receiver de VALOR (p Person, sin *) porque solo
// leen. Un receiver de valor trabaja sobre una copia, y estos métodos se
// promueven tanto a Patient como a *Patient.

// ID devuelve el identificador de la persona, por ejemplo "P-001".
func (p Person) ID() string { return p.id }

// Name devuelve el nombre completo.
func (p Person) Name() string { return p.name }

// Age devuelve la edad en años.
func (p Person) Age() int { return p.age }

// String implementa la interfaz fmt.Stringer: cuando se imprime una persona
// (o un Patient/Doctor que la embebe) con fmt.Println, sale "P-001 Yeimy Padilla".
func (p Person) String() string { return p.id + " " + p.name }
