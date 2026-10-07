package hospital

// Attender es cualquier miembro del personal que puede atender a un
// paciente que se quedó dormido.
//
// Las interfaces en Go son IMPLÍCITAS: no existe "implements". Un tipo
// cumple Attender por el simple hecho de tener estos cuatro métodos con
// estas firmas exactas. Doctor y Orderly no mencionan a Attender en ningún
// lado, y aun así el hospital los guarda juntos en un []Attender y los
// despacha sin saber cuál es cuál. Eso es el polimorfismo en Go.
//
// ID y Name ni siquiera los escriben Doctor ni Orderly: los reciben
// promovidos de Person, que llevan embebida.
type Attender interface {
	ID() string
	Name() string
	// Attend atiende a un paciente dormido en location y devuelve el
	// registro del episodio.
	Attend(p *Patient, location string) (EpisodeRecord, error)
	// IsAvailable dice si en este momento puede salir a atender a alguien.
	IsAvailable() bool
}

// Verificación en tiempo de compilación: se intenta guardar un *Doctor y un
// *Orderly (nil, sin crear nada) en variables de tipo Attender. Si a alguno
// le faltara un método, el programa no compilaría y el error diría cuál falta.
var (
	_ Attender = (*Doctor)(nil)
	_ Attender = (*Orderly)(nil)
)
