# Menú de pausa

Mismo portapapeles de la portada, pero derecho y centrado. Todo en píxeles de pantalla de 1280×720, dibujado a **1×**.

- **Imágenes**: `pause_<opción>_<cuadro>.png`, con opción `reanudar`, `ajustes` o `salir` (SALIR AL MENÚ) y cuadro `0` o `1`. Cada una con esa opción seleccionada. Todas se dibujan en **(442, 213)**.
- **Cursor**: los cuadros 0 y 1 solo cambian la posición del cursor. Se alternan cada 0,35 s.
- **Fondo**: detrás, el mapa sigue dibujado y encima se pone un rectángulo negro de 1280×720 con alfa 140 (≈55 %), para oscurecerlo.
- **Mouse**: `hitboxes.json` trae el rectángulo [x, y, ancho, alto] de cada renglón (no hay giro).
- **Teclado**:
  - **Esc** abre la pausa y, con la pausa abierta, la cierra (igual que REANUDAR).
  - ↑/↓ (o W/S) mueven la selección y Enter confirma.
- **Al confirmar**: el cursor se queda quieto (cuadro 0) 0,45 s y luego se hace la acción.
