# Hospital de los Costeños con Narcolepsia

Sistema de gestión del hospital escrito en Go. Registra dónde está cada paciente, qué médico lo atiende y cada episodio de sueño repentino, para asignarle cama rápido antes de que bloquee un pasillo.

Ejercicio aplicado de **Programación Orientada a Objetos**, Universidad EIA (profesor Sebastián Zapata Ramírez).

| | |
|---|---|
| **Integrante** | Samuel Giraldo Jiménez |
| **Lenguaje** | Go 1.21 o superior. El núcleo usa solo la librería estándar |
| **Estado** | 🚧 En construcción |

## Cómo correrlo

```bash
go run .          # escenario de demostración (Sección 6)
go test ./...     # pruebas
```

## Estructura

```
main.go         escenario de demostración + impresión (sin lógica de negocio)
hospital/       el modelo: structs, métodos, interfaces y consultas
simulation/     (bonus) simulación concurrente con goroutines
gui/            (bonus) juego en Ebitengine, módulo Go aparte
```

## Avance

- [x] Fase 0: repositorio, módulo y esqueleto
- [ ] Fase 1: `Person`, estados tipados y errores
- [ ] Fase 2: `Patient` y `Room`
- [ ] Fase 3: `Attender`, `Doctor`, `Orderly` y `EpisodeRecord`
- [ ] Fase 4: `Hospital` y las cuatro consultas
- [ ] Fase 5: escenario de demostración en `main.go`
- [ ] Fase 6: bonus de simulación concurrente
- [ ] Fase 7: bonus de interfaz gráfica
- [ ] Fase 8: revisión final y `AI_USAGE.md`
