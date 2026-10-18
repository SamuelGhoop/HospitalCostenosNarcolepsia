package ui

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Sin reproductor (como en el backdoor de pruebas, que nunca llama a
// initSound: GAME_DESIGN §10.9) playSound, playCoin y tickSound no hacen
// nada y, sobre todo, no revientan.
func TestPlaySound_WithoutAPlayerDoesNothingAndDoesNotPanic(t *testing.T) {
	old := soundPlayer
	soundPlayer = nil
	defer func() { soundPlayer = nil; soundPlayer = old }()

	playSound(sfxClick)
	playCoin(2)
	tickSound()
	// Si llegó hasta aquí sin pánico, ya está: no hay nada que "sonó" que
	// se pueda comprobar sin un reproductor de verdad.
}

// El backdoor de pruebas (-agent) nunca llama a initSound (solo lo hace
// NewApp, app.go): en el binario de tests nada la llama tampoco, así que
// el reproductor global tiene que seguir en nil. Si algún día alguien
// agrega initSound() a loadAssets (que sí usa el backdoor), este test
// avisa.
func TestAgentMode_NeverCreatesAPlayer(t *testing.T) {
	if soundPlayer != nil {
		t.Fatal("ya hay un reproductor de sonido armado antes de que el backdoor haga nada; algo llamó a initSound")
	}
	if err := loadAssets(); err != nil {
		t.Fatal(err)
	}
	if soundPlayer != nil {
		t.Error("loadAssets (lo que usa -agent) armó un reproductor de sonido; -agent no debe sonar nunca")
	}
}

// Con el sonido apagado (tecla M) playSound tampoco hace nada, aunque
// haya un cupo libre.
func TestPlaySound_MutedDoesNothing(t *testing.T) {
	old := soundOn
	soundOn = false
	defer func() { soundOn = old }()

	p := &audioPlayer{effects: map[string][]byte{sfxClick: {1, 2, 3, 4}}, cooldown: map[string]int{}}
	oldPlayer := soundPlayer
	soundPlayer = p
	defer func() { soundPlayer = oldPlayer }()

	playSound(sfxClick) // soundOn en falso: ni siquiera llega a tocar el cupo
	if p.cooldown[sfxClick] != 0 {
		t.Error("con el sonido apagado, playSound igual gastó el cupo del efecto")
	}
}

// El cupo evita que el mismo efecto se apile: no vuelve a sonar hasta que
// pasan soundCooldownTicks.
func TestAudioPlayer_CooldownPreventsStacking(t *testing.T) {
	p := &audioPlayer{cooldown: map[string]int{}}
	p.cooldown[sfxClick] = soundCooldownTicks
	if p.cooldown[sfxClick] <= 0 {
		t.Fatal("el cupo no quedó puesto")
	}
	for i := 0; i < soundCooldownTicks; i++ {
		p.tick()
	}
	if p.cooldown[sfxClick] != 0 {
		t.Errorf("tras %d ticks el cupo sigue en %d; se esperaba 0", soundCooldownTicks, p.cooldown[sfxClick])
	}
}

// La tecla M silencia y vuelve a activar el sonido, con su aviso corto
// "SONIDO: NO"/"SONIDO: SÍ" (app.go).
func TestApp_MTogglesSound(t *testing.T) {
	old := soundOn
	defer func() { soundOn = old }()
	oldKeyJustPressed := keyJustPressed
	defer func() { keyJustPressed = oldKeyJustPressed }()

	soundOn = true
	pressM := func(k ebiten.Key) bool { return k == ebiten.KeyM }
	noKey := func(ebiten.Key) bool { return false }

	a := &App{scene: &plainScene{}}
	keyJustPressed = pressM
	if err := a.update(false); err != nil {
		t.Fatal(err)
	}
	if soundOn {
		t.Error("con M apretado, el sonido sigue prendido; se esperaba apagado")
	}
	if a.muteNotice != muteNoticeTicks {
		t.Errorf("muteNotice = %d; se esperaba %d", a.muteNotice, muteNoticeTicks)
	}

	keyJustPressed = noKey
	for i := 0; i < muteNoticeTicks; i++ {
		if err := a.update(false); err != nil {
			t.Fatal(err)
		}
	}
	if a.muteNotice != 0 {
		t.Errorf("tras %d ticks el aviso sigue en %d; se esperaba 0", muteNoticeTicks, a.muteNotice)
	}
	if soundOn {
		t.Error("el sonido se volvió a prender solo, sin apretar M otra vez")
	}

	keyJustPressed = pressM
	if err := a.update(false); err != nil {
		t.Fatal(err)
	}
	if !soundOn {
		t.Error("con M otra vez, el sonido sigue apagado; se esperaba prendido")
	}
}
