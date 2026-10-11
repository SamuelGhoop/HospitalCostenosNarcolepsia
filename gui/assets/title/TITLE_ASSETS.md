# Portada: capas y animaciones

Sacado de `gui/design/claude-design/Hospital de los Costeños/Title Screen.dc.html` (Claude Design). Todas las imágenes están a **resolución nativa de la escena (320×180)** y se dibujan **×4** (escalado entero, filtro nearest) para llenar 1280×720. Todas las coordenadas de este archivo están en píxeles nativos (multiplicar por 4 para la pantalla).

Todo se anima por ticks de `Update` (60/s), nunca con `time.Sleep`.

## Orden de dibujo y posición

| # | Archivo | Posición (x, y) | Animación |
|---|---|---|---|
| 1 | `fondo.png` (320×180) | (0, 0) | Ninguna: cielo, sol y mar |
| 2 | `nube_a.png` y `nube_b.png` (42×15) | nube A: y = 57 · nube B: y = 29 | x va de −42 a 320 y vuelve a empezar, 1 px por paso. Nube A: un paso cada 0,249 s (ciclo de 90 s) y arranca en x = 40. Nube B: un paso cada 0,359 s (ciclo de 130 s) y arranca en x = 12 |
| 3 | `olas.png` (336×28) | (−8, 100) | x = −8 + d, con d = 0, 1, 2, 3, 2, 1… (ida y vuelta), un paso cada 0,6 s |
| 4 | `espuma.png` (336×5) | (−8, 123) | x = −8 − d, igual que las olas pero un paso cada 0,45 s |
| 5 | `suelo.png` (320×180) | (0, 0) | Ninguna: arena, hospital y troncos de las palmeras |
| 6 | `copa1_frames.png` (5 cuadros de 74×42 en fila) | (−13, 79) | Copa girada −2°, −1°, 0°, +1° y +2°. Cuadros 0,1,2,3,4,3,2,1… (ida y vuelta), uno cada 0,75 s |
| 7 | `copa2_frames.png` (igual) | (145, 85) | Igual que la copa 1, pero desfasada medio ciclo |
| 8 | `patient_legs.png` (32×48) | (15, 112) | Ninguna |
| 9 | `patient_torso.png` (32×48) | (15, 112 + r) | Respirar: r = 0, −1, −2, −1, un valor cada 1 s (ciclo de 4 s) |
| 10 | `patient_head.png` (32×48) | (15 + cx, 112 + r + cy) | Respira con el torso (mismo r) y además cabecea en un ciclo de 6 s: (cx, cy) = (0,0) hasta 4,2 s · (1,1) hasta 4,68 s · (1,2) hasta 5,16 s · (0,1) hasta 5,64 s · (0,0) hasta 6 s |
| 11 | `zzz1.png`, `zzz2.png`, `zzz3.png` (32×48) | (15, 112 + z) | Ciclo de 3 s en 6 pasos de 0,5 s: z va de +2 a −4 (1 px por paso). Opacidad: 0 → 1 al 30 % del ciclo → 0 al final. Desfases: zzz1 0 s, zzz2 1 s, zzz3 2 s |

Las capas 8 a 11 comparten el mismo cuadro de 32×48, así que se dibujan en el mismo punto y solo cambian los desplazamientos.

## Lo que dibuja el juego con texto (fuente Press Start 2P)

Se toma de la maqueta (`gui/design/claude-design/Hospital de los Costeños/Title Screen.dc.html`) solo como referencia visual, con los textos en español:

- **Logo**: "HOSPITAL DE LOS" en blanco y "COSTEÑOS" en amarillo #ffd23f con sombra roja #c4302b. Los dos llevan contorno negro #141414 en las 8 direcciones (texto dibujado 8 veces desplazado y encima el relleno). Usa tamaños múltiplos de 8 px para que la fuente quede nítida.
- **Banda azul** #2f6fd6 con borde negro, la almohada y "con Narcolepsia".
- **Zzz del título**: tres letras (z, Z, Z) a la derecha de la banda. Cada una sube y se desvanece en un ciclo de 2,4 s de 6 pasos, desfasadas 0,8 s entre sí.
- **Portapapeles**: NO se dibuja con texto, va como imagen (ver abajo).
- **"PRESIONA ENTER"**: blanco con contorno, parpadea cada 1,2 s.
- **Versión**: "v1.0 · Universidad EIA · POO en Go", abajo a la derecha.

## Portapapeles del menú (`clipboard/`)

El portapapeles ya viene dibujado en español, inclinado 2,5° (con nearest, sin borroso) y con su sombra. Hay **una imagen por estado** y todas se dibujan **a 1× en (797, 241)**, en píxeles de pantalla de 1280×720 (no ×4).

- **Nombres**: `clipboard_<opción>_<cuadro>.png`, con opción `continuar`, `nueva`, `demo`, `ajustes`, `creditos` o `salir`, y cuadro `0` o `1`. En cada imagen esa opción está seleccionada: fondo amarillo, cursor rojo, chulo verde y texto azul.
- **Cursor**: los cuadros 0 y 1 solo cambian la posición del cursor. Se alternan cada 0,35 s (ciclo de 0,7 s).
- **Sin partida guardada**: se usan las `clipboard_sincontinuar_<opción>_<cuadro>.png`, donde CONTINUAR sale en gris. Ahí la selección se salta CONTINUAR, así que no existe `sincontinuar_continuar`.
- **Mouse**: `hitboxes.json` trae, para cada opción, los 4 vértices de su renglón ya inclinado, en píxeles de pantalla. El hover se detecta con punto dentro de polígono.
- **Al confirmar con Enter o clic**: el cursor se queda quieto (cuadro 0) durante 0,45 s y luego cambia de escena.
- Las tildes de SECCIÓN y CRÉDITOS están dibujadas a mano encima de la O y la E normales. La Ó y la É de Press Start 2P vienen encogidas para que la tilde quepa en el cuadro de 8×8, y se ven como minúsculas.
