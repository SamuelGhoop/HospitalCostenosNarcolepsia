// Package game tiene las reglas del modo demostración y del modo juego del
// Hospital de los Costeños con Narcolepsia (ver GAME_DESIGN.md).
//
// Se construye ENCIMA del paquete hospital, sin modificarlo: todo cambio al
// modelo pasa por los métodos de hospital.Hospital. Usa solo la librería
// estándar y no sabe nada de Ebitengine: la interfaz (gui/) solo lee las
// vistas que este paquete le entrega (copias con valores, nunca punteros del
// modelo) y llama sus métodos.
package game
