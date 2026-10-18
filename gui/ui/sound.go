package ui

import (
	"math/rand"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/sfx"
)

// Sonido generado por código (pedido de Samuel el 2026-10-17: "añade
// sonidos acordes, que la vida está ahí"; GAME_DESIGN §9.5 y §10.11). Nada
// de archivos de audio ni licencias: los efectos los sintetiza
// gui/sfx (funciones puras, con sus tests) al arrancar, como PCM de 16
// bits a 44100 Hz; este archivo es el único que importa
// github.com/hajimehoshi/ebiten/v2/audio.
//
// soundSampleRate: 44100 Hz, el que pide GAME_DESIGN §10.11.
const soundSampleRate = 44100

// Nombres de los efectos (game_scene.go, hiring_scene.go, menu.go, anim.go
// y sound_events.go los usan para pedir que suenen).
const (
	sfxClick      = "click"
	sfxOpen       = "open"
	sfxClose      = "close"
	sfxHire       = "hire"
	sfxCoin       = "coin"
	sfxStreakLost = "streakLost"
	sfxStarDown   = "starDown"
	sfxStarUp     = "starUp"
	sfxCollapse   = "collapse"
	sfxRush       = "rush"
	sfxDayEnd     = "dayEnd"
	sfxGameOver   = "gameOver"
	sfxSnore      = "snore"
)

// soundCooldownTicks: el mismo efecto no vuelve a sonar antes de esto
// (GAME_DESIGN §10.11: "que no se apilen"), unos 3 ticks (0,05 s a 60/s).
const soundCooldownTicks = 3

// soundOn: si está en falso, playSound no suena (tecla M, app.go).
// soundPlayer es el reproductor ya armado; nil mientras no se llamó a
// initSound (el backdoor -agent nunca la llama, así nunca suena nada,
// GAME_DESIGN §10.9) o si el dispositivo de audio no se pudo abrir.
var (
	soundOn        = true
	soundPlayer    *audioPlayer
	soundPlayerOne sync.Once
)

// audioPlayer sintetiza los efectos una sola vez y los reproduce con el
// contexto de Ebitengine, con un cupo corto para que el mismo efecto no se
// apile.
type audioPlayer struct {
	ctx      *audio.Context
	effects  map[string][]byte
	cooldown map[string]int
}

// effectGenerators son las funciones puras de gui/sfx, una por nombre de
// efecto (todas con la misma firma salvo coin, que además necesita la
// racha: se arma aparte en play()).
var effectGenerators = map[string]func(*rand.Rand) []byte{
	sfxClick:      sfx.Click,
	sfxOpen:       sfx.Open,
	sfxClose:      sfx.Close,
	sfxHire:       sfx.Hire,
	sfxStreakLost: sfx.StreakLost,
	sfxStarDown:   sfx.StarDown,
	sfxStarUp:     sfx.StarUp,
	sfxCollapse:   sfx.Collapse,
	sfxRush:       sfx.Rush,
	sfxDayEnd:     sfx.DayEnd,
	sfxGameOver:   sfx.GameOver,
	sfxSnore:      sfx.Snore,
}

// coinEffects son las 5 variantes de Coin, una por racha (0 a 4), para no
// sintetizarla de nuevo en cada cobro.
var coinEffects [5][]byte

// initSound arma el reproductor (solo la llama NewApp: el modo normal, no
// el backdoor de pruebas). Si el dispositivo de audio no se puede abrir
// (o el contexto ya existe, lo que no debería pasar fuera de los tests),
// se recupera del pánico y deja soundPlayer en nil: el juego sigue sin
// sonido, sin caerse (GAME_DESIGN §10.11).
func initSound() {
	soundPlayerOne.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				soundPlayer = nil
			}
		}()
		rnd := rand.New(rand.NewSource(1)) // azar propio de la interfaz, semilla fija: siempre el mismo sonido
		effects := map[string][]byte{}
		for name, gen := range effectGenerators {
			effects[name] = gen(rnd)
		}
		for streak := range coinEffects {
			coinEffects[streak] = sfx.Coin(streak, rnd)
		}
		soundPlayer = &audioPlayer{ctx: audio.NewContext(soundSampleRate), effects: effects, cooldown: map[string]int{}}
	})
}

// playSound pide que suene el efecto name, si el sonido está prendido, hay
// reproductor (no en -agent) y no sonó hace menos de soundCooldownTicks.
func playSound(name string) {
	if !soundOn || soundPlayer == nil {
		return
	}
	soundPlayer.play(name)
}

// playCoin es playSound para "coin": streak (topada a 4, GAME_DESIGN
// §10.11) elige cuál de las 5 variantes ya sintetizadas suena.
func playCoin(streak int) {
	if !soundOn || soundPlayer == nil {
		return
	}
	if streak < 0 {
		streak = 0
	}
	if streak >= len(coinEffects) {
		streak = len(coinEffects) - 1
	}
	soundPlayer.playData(sfxCoin, coinEffects[streak])
}

// tickSound baja en 1 el cupo de cada efecto; la llama GameScene.update
// una vez por tick, como hud.update.
func tickSound() {
	if soundPlayer == nil {
		return
	}
	soundPlayer.tick()
}

func (p *audioPlayer) play(name string) {
	data, ok := p.effects[name]
	if !ok {
		return
	}
	p.playData(name, data)
}

func (p *audioPlayer) playData(name string, data []byte) {
	if p.cooldown[name] > 0 {
		return
	}
	pl := p.ctx.NewPlayerFromBytes(data)
	pl.Play()
	p.cooldown[name] = soundCooldownTicks
}

func (p *audioPlayer) tick() {
	for name, left := range p.cooldown {
		if left > 0 {
			p.cooldown[name] = left - 1
		}
	}
}
