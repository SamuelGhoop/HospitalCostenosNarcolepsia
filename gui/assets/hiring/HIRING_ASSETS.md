# Bolsa de empleo

Mismo portapapeles de la portada y la pausa, ahora grande. Todo en píxeles de pantalla de 1280×720, dibujado a **1×**. Los rectángulos son `image.Rect(x0, y0, x1, y1)` con `x1`, `y1` exclusivos; "x, y, ancho, alto" donde se diga. Vistas aprobadas el 2026-10-15 (Figma, sección "Bolsa de empleo", frames A, B y C).

## Imágenes

| Archivo | Tamaño | Dónde |
|---|---|---|
| `hiring_board.png` | 730×556 | **(280, 81)**. Tablero vacío: cuerpo de madera (280, 96)–(1000, 624), gancho (560, 81)–(720, 127), sombra de alfa 20 corrida +10, +13 (ya viene en el PNG). Papel (con su sombra) (310, 139)–(969, 598). |
| `hiring_arrow_left_0.png` / `_1.png` | 56×80 | **(208, 320)**. `_0` normal, `_1` presionada (mientras el mouse está apretado encima o 0,10 s tras ←/A). |
| `hiring_arrow_right_0.png` / `_1.png` | 56×80 | **(1016, 320)**. Igual que la izquierda. |
| `hud_hire_clipboard.png` | 120×74 | **(976, 634)**. Portapapeles chiquito del HUD, vacío. El cuerpo es de 120×68 en (976, 640)–(1096, 708); el gancho sobresale 6 px por arriba (por eso la imagen mide 74 de alto). Hitbox de clic: el cuerpo. |

Fuente: Press Start 2P (`drawPixelText`). Tamaño 16 = renglones normales, 24 = nombre y botón, 8 = HUD. Las tildes y la Ñ con `accentMarks`, como en el resto.

## Portapapeles chiquito del HUD (lo dibuja el código encima de la imagen)
- **Cabeza** del candidato de la hoja actual: cuadro *quieto* 0 del cuerpo + pelo teñido, recorte (2, 3)–(14, 11) del cuadro de 16×24 (12×8), a **×3** = 36×24, en **(1018, 659)**.
- **"CONTRATAR"** a 8 px, #141414, centrado en x = 1036, y = 687.
- Papel interior: (988, 655)–(1084, 696).

## Abrir y cerrar
- **Fondo**: el mapa sigue dibujado y encima un rectángulo negro de 1280×720 cuyo alfa va de 0 a 140.
- **Abrir**: 0,22 s, *ease-out* (`e = 1 - (1-t)^3`, con `t` de 0 a 1). Lo más simple: dibujar el tablero completo (imagen + textos + flechas) en una imagen de 1280×720 y ponerla en pantalla con escala `s = 0,167 + (1 - 0,167)·e` y centro `c = (1036, 674) + ((640, 360) - (1036, 674))·e`; es decir, un punto `p` de la vista final queda en `c + (p - (640, 360))·s`. Al empezar (s = 0,167, 120/720) el tablero tapa justo el portapapeles chiquito del HUD, cuyo centro es (1036, 674); al terminar está en su sitio. Filtro *nearest*.
- **Cerrar**: 0,18 s, *ease-in* (`t^3`), lo mismo al revés (s de 1 a 0,167, centro de (640, 360) a (1036, 674), alfa del oscurecido 140 → 0).
- Mientras anima no se aceptan clics en el tablero.

## Cambio de hoja (← → o flechas)
0,10 s en total: en los primeros 0,05 s todo lo que va sobre el papel (textos, foto, puntos, botón) sube de 0 a **6 px**; a los 0,05 s se cambia al candidato nuevo; en los 0,05 s siguientes baja de 6 a 0 px. Lineal. El tablero de madera y las flechas no se mueven (el papel del PNG tampoco: solo el contenido). Mientras dura, las otras pulsaciones de flecha se ignoran.

## Textos y rectángulos (hoja de un candidato)
Columna izquierda x = 334; columna de datos x = 526; borde derecho x = 946.

| Qué | Posición (x, y) | Tamaño | Color | Alineación |
|---|---|---|---|---|
| "BOLSA DE EMPLEO" | (334, 158) | 16 | #2f6fd6 | izquierda |
| "ESC: CERRAR" | derecha en 946, y = 158 | 16 | #e5484d | derecha |
| Divisor | rect 334, 184, 612×3 | — | #141414 | — |
| Marco de la foto | rect 334, 204, 160×224 | — | #141414 | — |
| Fondo de la foto | rect 339, 209, 150×214 (contorno de 5) | — | #eceae4 | — |
| Retrato ×8 | (350, 220), 128×192 | — | cuadro *quieto* 0 completo (cuerpo + pelo teñido) a ×8 | — |
| "1/3" (hoja) | centro x = 414, y = 444 | 16 | #141414 | centrado |
| Nombre ("ROSA MENDOZA") | (526, 210) | 24 | #141414 | izquierda |
| Rol | (526, 250) | 16 | MÉDICO/MÉDICA #2f6fd6, CAMILLERO #1f8a3a | izquierda |
| Especialidad | (526, 276) | 16 | #5a616b | izquierda |
| "EDAD: 34" | (526, 302) | 16 | #5a616b | izquierda |
| Divisor fino | rect 526, 328, 420×2 | — | #e3ddcc | — |
| "RAPIDEZ" | (526, 344) | 16 | #141414 | izquierda |
| Puntos de rapidez | 5 cuadros 24×24 en x = 706 + 30·i, y = 340; relleno 18×18 en (709 + 30·i, 343) | — | contorno #141414; lleno #ffd23f, vacío #e3ddcc | — |
| "PERICIA" + puntos | igual, y = 380 (texto) y 376 (cuadros), 379 (relleno) | 16 | igual | — |
| Camillero: "SIN PERICIA: NO REVISA" | (526, 380), en lugar de PERICIA y sus puntos | 16 | #5a616b | izquierda |
| Divisor fino 2 | rect 526, 406, 420×2 | — | #e3ddcc | — |
| "SUELDO POR DÍA" / valor | (526, 418) / derecha en 946 | 16 | #5a616b / #141414 | izq. / der. |
| "CONTRATARLO" / valor | (526, 442) / derecha en 946 | 16 | #5a616b / #141414 | izq. / der. |
| "TU PLATA" / valor | (526, 466) / derecha en 946 | 16 | #5a616b / #1f8a3a si alcanza, #e5484d si no | izq. / der. |

Plata con punto de miles: `$ 2.000`, `$ 600`.

## Botón CONTRATAR
| Parte | Rect (x, y, ancho × alto) | Alcanza | No alcanza |
|---|---|---|---|
| Contorno | 334, 500, 612×64 | #141414 | #141414 |
| Relleno (contorno de 5) | 339, 505, 602×54 | #2fb84f | #c9ced6 |
| Sombra de abajo | 339, 553, 602×6 | #1f8a3a | #8a909a |
| "CONTRATAR" (24) | centro x = 640, y = 520 | #ffffff | #5a616b |
| "TE FALTAN $ X" (16) | centro x = 640, y = 574 | — | #e5484d |

Hitbox del botón: (334, 500)–(946, 564). Sin plata no hace nada al clic.

## Nota de sprites del camillero
`gui/assets/sprites/staff_orderly_skin{1,3,4}.png`: 64×120, mismo formato que `staff_doctor_*` (filas: quieto 2, caminar abajo 4, caminar arriba 4, caminar izquierda 4, **empujar 4**; la derecha se voltea en código). En la fila *empujar* (fila 4) el pelo usa la fila de **caminar a la izquierda** (fila 3) de `patient_hair_*`, columna por columna: la cabeza está igual en las dos.
