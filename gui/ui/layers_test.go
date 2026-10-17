package ui

import (
	"image"
	"image/color"
	"math/rand"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// Todas las hojas vienen embebidas: las del paciente miden 4 cuadros × 10
// filas (una por animación) y las del médico 4 × 5 (quieto, caminar abajo,
// arriba e izquierda, y atender).
func TestSpriteSheets_AllEmbeddedWithTheirSizes(t *testing.T) {
	patient := image.Pt(4*cellWidth, 10*cellHeight)
	doctor := image.Pt(4*cellWidth, 5*cellHeight)
	var sheets []string
	sheets = append(sheets, bodySheets[:]...)
	for _, p := range shirtSheets {
		sheets = append(sheets, p)
	}
	for _, p := range hairSheets {
		sheets = append(sheets, p)
	}
	for _, p := range hatSheets {
		sheets = append(sheets, p)
	}
	for _, p := range sheets {
		img, err := decodePNG(assets.Sprites, p)
		if err != nil {
			t.Fatal(err)
		}
		if got := img.Bounds().Size(); got != patient {
			t.Errorf("%s mide %v; se esperaba %v", p, got, patient)
		}
	}
	for _, p := range doctorSheets {
		img, err := decodePNG(assets.Sprites, p)
		if err != nil {
			t.Fatal(err)
		}
		if got := img.Bounds().Size(); got != doctor {
			t.Errorf("%s mide %v; se esperaba %v", p, got, doctor)
		}
	}
	// El camillero tiene el mismo formato que el médico en sus primeras 5
	// filas (la quinta es empujar de lado, no atender), pero suma dos más:
	// empujar hacia abajo y hacia arriba (GAME_DESIGN §10.3, 2026-10-17).
	orderly := image.Pt(4*cellWidth, 7*cellHeight)
	for _, p := range orderlySheets {
		img, err := decodePNG(assets.Sprites, p)
		if err != nil {
			t.Fatal(err)
		}
		if got := img.Bounds().Size(); got != orderly {
			t.Errorf("%s mide %v; se esperaba %v (7 filas)", p, got, orderly)
		}
	}
	if len(shirtSheets) != 3 || len(hairSheets) != 3 || len(hatSheets) != 2 {
		t.Errorf("hay %d camisas, %d pelos y %d sombreros; se esperaban 3, 3 (calvo no tiene hoja) y 2 (sin sombrero tampoco)",
			len(shirtSheets), len(hairSheets), len(hatSheets))
	}
}

// Cada tono de piel usa su hoja; el que todavía no tiene hoja (la piel 2)
// usa la 1, así la piel 2 entra sola el día que llegue el PNG.
func TestSkinSheets_AMissingToneUsesSkin1(t *testing.T) {
	fsys := fstest.MapFS{
		"sprites/patient_body_skin1.png": {},
		"sprites/patient_body_skin3.png": {},
	}
	want := [4]string{
		"sprites/patient_body_skin1.png", "sprites/patient_body_skin1.png",
		"sprites/patient_body_skin3.png", "sprites/patient_body_skin1.png",
	}
	if got := skinSheets(fsys, "patient_body"); got != want {
		t.Errorf("con las pieles 1 y 3: %q; se esperaba %q", got, want)
	}

	// Con los PNG de verdad: las pieles 1, 3 y 4 son las suyas.
	for _, prefix := range []string{"patient_body", "staff_doctor"} {
		got := skinSheets(assets.Sprites, prefix)
		for tone, own := range map[int]string{0: "1", 2: "3", 3: "4"} {
			if p := "sprites/" + prefix + "_skin" + own + ".png"; got[tone] != p {
				t.Errorf("%s, tono %d: %q; se esperaba %q", prefix, tone, got[tone], p)
			}
		}
	}
}

// El paciente se dibuja en el orden cuerpo → camisa → pelo → sombrero,
// todas con el mismo cuadro de la hoja. Camisa y pelo van teñidos con su
// color de la paleta; el cuerpo y el sombrero, tal cual.
func TestPatientLayers_OrderFrameAndTints(t *testing.T) {
	look := game.Appearance{SkinTone: 2, ShirtColor: 3, ShirtPattern: game.FloralShirt,
		Hat: game.VueltiaoHat, Hair: game.AfroHair, HairColor: 1}
	r := frameRect(animWalkLeft, 2)
	want := []layer{
		{"sprites/patient_body_skin3.png", r, nil},
		{"sprites/patient_shirt_florida.png", r, shirtPalette[3]},
		{"sprites/patient_hair_afro.png", r, hairPalette[1]},
		{"sprites/patient_hat_vueltiao.png", r, nil},
	}
	if got := patientLayers(look, animWalkLeft, 2); !reflect.DeepEqual(got, want) {
		t.Errorf("capas = %+v\nse esperaba %+v", got, want)
	}

	// Calvo y sin sombrero: solo cuerpo y camisa.
	look.Hair, look.Hat = game.Bald, game.NoHat
	got := patientLayers(look, animIdle, 0)
	if len(got) != 2 || got[0].sheet != bodySheets[2] || got[1].sheet != shirtSheets[game.FloralShirt] {
		t.Errorf("calvo y sin sombrero: %+v; se esperaba solo cuerpo y camisa", got)
	}
}

// El médico: su hoja (bata) y encima el pelo del paciente. En quieto y
// caminando el pelo usa el mismo cuadro; al atender (fila 5 de su hoja) el
// pelo usa el cuadro 1 de quieto, porque la cabeza no se mueve.
func TestDoctorLayers_HairFrames(t *testing.T) {
	look := game.Appearance{SkinTone: 3, Hair: game.ShortHair, HairColor: 2, Hat: game.CapHat}
	for _, c := range []struct {
		a     animation
		frame int
	}{{animIdle, 1}, {animWalkDown, 3}, {animWalkUp, 0}, {animWalkLeft, 2}} {
		want := []layer{
			{"sprites/staff_doctor_skin4.png", frameRect(c.a, c.frame), nil},
			{"sprites/patient_hair_corto.png", frameRect(c.a, c.frame), hairPalette[2]},
		}
		if got := doctorLayers(look, c.a, c.frame); !reflect.DeepEqual(got, want) {
			t.Errorf("%v cuadro %d: %+v\nse esperaba %+v (el médico no lleva sombrero ni camisa)", c.a, c.frame, got, want)
		}
	}

	got := doctorLayers(look, animAttend, 3)
	if want := image.Rect(3*cellWidth, 4*cellHeight, 4*cellWidth, 5*cellHeight); got[0].rect != want {
		t.Errorf("atender, cuerpo: %v; se esperaba la fila 5, cuadro 4: %v", got[0].rect, want)
	}
	if want := frameRect(animIdle, 0); got[1].rect != want {
		t.Errorf("atender, pelo: %v; se esperaba la fila 1, cuadro 1: %v", got[1].rect, want)
	}

	look.Hair = game.Bald
	if got := doctorLayers(look, animIdle, 0); len(got) != 1 {
		t.Errorf("médico calvo: %d capas; se esperaba solo el cuerpo", len(got))
	}
}

// El camillero: su hoja (uniforme) y encima el pelo del paciente. En quieto
// y caminando el pelo usa el mismo cuadro; empujando (filas 5 a 7 de SU
// hoja) el pelo usa la fila de caminar en esa misma dirección, columna por
// columna (la cabeza no se mueve distinto en las dos; GAME_DESIGN §10.3,
// ampliado el 2026-10-17 con empujar hacia abajo y hacia arriba).
func TestOrderlyLayers_PushUsesTheMatchingWalkHairRow(t *testing.T) {
	look := game.Appearance{SkinTone: 1, Hair: game.BraidsHair, HairColor: 0}
	for _, c := range []struct {
		a     animation
		frame int
	}{{animIdle, 0}, {animWalkDown, 2}, {animWalkLeft, 1}} {
		want := []layer{
			{orderlySheets[1], frameRect(c.a, c.frame), nil},
			{"sprites/patient_hair_trenzas.png", frameRect(c.a, c.frame), hairPalette[0]},
		}
		if got := orderlyLayers(look, c.a, c.frame); !reflect.DeepEqual(got, want) {
			t.Errorf("%v cuadro %d: %+v\nse esperaba %+v", c.a, c.frame, got, want)
		}
	}

	for _, c := range []struct {
		a        animation
		row      int
		hairAnim animation
	}{
		{animPush, 4, animWalkLeft},
		{animPushDown, 5, animWalkDown},
		{animPushUp, 6, animWalkUp},
	} {
		got := orderlyLayers(look, c.a, 2)
		wantBody := image.Rect(2*cellWidth, c.row*cellHeight, 3*cellWidth, (c.row+1)*cellHeight)
		if got[0].rect != wantBody {
			t.Errorf("%v, cuerpo: %v; se esperaba la fila %d, cuadro 3: %v", c.a, got[0].rect, c.row+1, wantBody)
		}
		if want := frameRect(c.hairAnim, 2); got[1].rect != want {
			t.Errorf("%v, pelo: %v; se esperaba la fila de %v, cuadro 3: %v", c.a, got[1].rect, c.hairAnim, want)
		}
	}

	look.Hair = game.Bald
	if got := orderlyLayers(look, animIdle, 0); len(got) != 1 {
		t.Errorf("camillero calvo: %d capas; se esperaba solo el cuerpo", len(got))
	}
}

// Las camisas usan los 6 colores del catálogo de sprites de la maqueta y el
// pelo los 4 de GAME_DESIGN §7. Ningún sorteo de game se sale de las
// paletas, y cada índice sale alguna vez.
func TestPalettes_CoverEveryAppearance(t *testing.T) {
	shirts := []color.RGBA{
		{0x2f, 0x6f, 0xd6, 0xff}, {0xe5, 0x48, 0x4d, 0xff}, {0x2f, 0xb8, 0x4f, 0xff},
		{0xff, 0xd2, 0x3f, 0xff}, {0x22, 0xb8, 0xb0, 0xff}, {0xf5, 0xa6, 0x23, 0xff},
	}
	hair := []color.RGBA{
		{0x2e, 0x2a, 0x28, 0xff}, {0x5b, 0x3a, 0x24, 0xff}, {0x9c, 0x6b, 0x3f, 0xff}, {0xd9, 0xd6, 0xd0, 0xff},
	}
	if !reflect.DeepEqual(shirtPalette[:], shirts) {
		t.Errorf("paleta de camisas = %v; se esperaba %v", shirtPalette, shirts)
	}
	if !reflect.DeepEqual(hairPalette[:], hair) {
		t.Errorf("paleta de pelo = %v; se esperaba %v", hairPalette, hair)
	}

	rng := rand.New(rand.NewSource(1))
	seenShirt, seenHair, seenSkin := map[int]bool{}, map[int]bool{}, map[int]bool{}
	for i := 0; i < 2000; i++ {
		a := game.RandomAppearance(rng)
		if a.ShirtColor < 0 || a.ShirtColor >= len(shirtPalette) || a.HairColor < 0 || a.HairColor >= len(hairPalette) ||
			a.SkinTone < 0 || a.SkinTone >= len(bodySheets) {
			t.Fatalf("la apariencia %+v se sale de las paletas", a)
		}
		seenShirt[a.ShirtColor], seenHair[a.HairColor], seenSkin[a.SkinTone] = true, true, true
	}
	if len(seenShirt) != len(shirtPalette) || len(seenHair) != len(hairPalette) || len(seenSkin) != len(bodySheets) {
		t.Errorf("en 2000 sorteos salieron %d camisas, %d pelos y %d pieles distintas; se esperaban todas", len(seenShirt), len(seenHair), len(seenSkin))
	}
}

// La demo completa hasta el paso 8: cada personaje se dibuja con la
// apariencia de su ID; cuando un episodio lo atiende un médico, la
// coreografía espera a que atienda (y él hace la animación), y cuando lo
// atiende el camillero, no espera.
func TestDemo_LooksAndTheDoctorAttends(t *testing.T) {
	if err := loadFonts(); err != nil { // los puestos dependen de los rótulos
		t.Fatal(err)
	}
	s := NewDemoScene(game.NewDemo())
	holds := map[string]int{} // quien atendió cada episodio → la espera de la fase 2
	attended := map[string]bool{}
	for step := 1; step <= 8; step++ {
		if err := s.advance(); err != nil {
			t.Fatal(err)
		}
		if s.choreo != nil && len(s.choreo.phases) > 1 && s.choreo.phases[1].attend != "" {
			holds[s.choreo.phases[1].attend] = s.choreo.phases[1].hold
		}
		for i := 0; i < 3000 && s.choreo != nil; i++ {
			s.animate()
			for id, a := range s.staff {
				if a.anim == animAttend {
					attended[id] = true
				}
			}
		}
		for _, f := range s.figures {
			if f.look != demoLook(f.id) {
				t.Fatalf("paso %d: %s se dibuja con %+v; se esperaba la apariencia de su ID", step, f.id, f.look)
			}
		}
	}

	doctors, orderlies := 0, 0
	for id, hold := range holds {
		f, _ := s.figure(id)
		want := 0
		if f.doctor {
			want, doctors = attendTicks, doctors+1
			if !attended[id] {
				t.Errorf("%s atendió un episodio y nunca hizo la animación de atender", id)
			}
		} else {
			orderlies++
		}
		if hold != want {
			t.Errorf("%s atendió y la coreografía esperó %d ticks; se esperaban %d", id, hold, want)
		}
	}
	if doctors == 0 || orderlies == 0 {
		t.Errorf("en los pasos 5–8 atendieron %d médicos y %d camilleros; se esperaban de los dos", doctors, orderlies)
	}
}

// En la demo la apariencia sale del ID: el mismo personaje siempre se ve
// igual, y los de la demo no salen todos iguales.
func TestDemoLook_FixedByIDAndVaried(t *testing.T) {
	ids := []string{"P-001", "P-002", "P-003", "P-004", "P-005", "D-01", "D-02"}
	distinct := map[game.Appearance]bool{}
	for _, id := range ids {
		if demoLook(id) != demoLook(id) {
			t.Errorf("%s cambió de apariencia entre dos llamadas", id)
		}
		distinct[demoLook(id)] = true
	}
	if len(distinct) < len(ids)-1 {
		t.Errorf("los %d personajes de la demo tienen solo %d apariencias distintas", len(ids), len(distinct))
	}
}

// Variedad en la demo (pedido de Samuel, 2026-10-11): los 2 médicos se ven
// con piel distinta (hoja distinta: la piel 2 todavía se dibuja con la 1) y
// no hay dos pacientes con el mismo sombrero y el mismo color de camisa.
func TestDemoLook_DoctorsDifferAndNoTwinPatients(t *testing.T) {
	if a, b := demoLook("D-01"), demoLook("D-02"); doctorSheets[a.SkinTone] == doctorSheets[b.SkinTone] {
		t.Errorf("D-01 y D-02 se dibujan con la misma piel (%s)", doctorSheets[a.SkinTone])
	}
	type key struct {
		hat   game.Hat
		shirt int
	}
	seen := map[key]string{}
	for _, id := range []string{"P-001", "P-002", "P-003", "P-004", "P-005"} {
		l := demoLook(id)
		k := key{l.Hat, l.ShirtColor}
		if other, dup := seen[k]; dup {
			t.Errorf("%s y %s tienen el mismo sombrero (%v) y el mismo color de camisa (%d)", other, id, l.Hat, l.ShirtColor)
		}
		seen[k] = id
	}
}
