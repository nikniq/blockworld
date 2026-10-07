package main

import (
	"encoding/binary"
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Audio holds procedurally synthesised sound effects, so the game needs no
// asset files. All calls are no-ops if the audio device failed to open.
type Audio struct {
	ok       bool
	Shoot    rl.Sound
	Click    rl.Sound
	Hit      rl.Sound
	Head     rl.Sound
	Reload   rl.Sound
	Hurt     rl.Sound
	Die      rl.Sound
	Pickup   rl.Sound
	Wave     rl.Sound
	Clear    rl.Sound
	GameOver rl.Sound
	Dig      rl.Sound
	Place    rl.Sound
	Explode  rl.Sound
	Fuse     rl.Sound
	Craft    rl.Sound
	Swing    rl.Sound
	Burn     rl.Sound
	Splash   rl.Sound
	Eat      rl.Sound
	Groan    rl.Sound
	Music    rl.Sound // generated ambient loop
	Roar     rl.Sound
	Rain     rl.Sound
	Rattle   rl.Sound
	Steps    [4]rl.Sound // grass, stone, sand, wood
	sounds   []rl.Sound
}

const sampleRate = 22050

func NewAudio() *Audio {
	rl.InitAudioDevice()
	a := &Audio{ok: rl.IsAudioDeviceReady()}
	if !a.ok {
		return a
	}
	noise := func() float32 { return rand.Float32()*2 - 1 }
	sin := func(f, t float32) float32 { return float32(math.Sin(2 * math.Pi * float64(f*t))) }
	exp := func(k, t float32) float32 { return float32(math.Exp(-float64(k * t))) }
	saw := func(f, t float32) float32 { x := f * t; return 2*(x-float32(math.Floor(float64(x)))) - 1 }

	a.Shoot = a.synth(0.28, func(t float32) float32 {
		return (noise()*0.7 + sin(80, t)*0.8 + sin(160, t)*0.3) * exp(22, t)
	})
	a.Click = a.synth(0.05, func(t float32) float32 { return noise() * exp(80, t) * 0.4 })
	a.Hit = a.synth(0.10, func(t float32) float32 { return (sin(520, t) + noise()*0.3) * exp(45, t) * 0.7 })
	a.Head = a.synth(0.35, func(t float32) float32 { return (sin(1320, t)*0.6 + sin(1980, t)*0.3) * exp(11, t) })
	a.Reload = a.synth(0.55, func(t float32) float32 {
		v := float32(0)
		if t < 0.06 {
			v = noise() * exp(60, t)
		}
		if t > 0.34 {
			v += noise() * exp(60, t-0.34)
		}
		return v * 0.5
	})
	a.Hurt = a.synth(0.4, func(t float32) float32 { return (saw(110, t)*0.6 + noise()*0.2) * exp(8, t) })
	a.Die = a.synth(0.5, func(t float32) float32 {
		f := 380 - 300*t
		return (saw(f, t)*0.5 + noise()*0.25) * exp(5, t)
	})
	a.Pickup = a.synth(0.36, func(t float32) float32 {
		notes := []float32{523, 659, 784}
		i := int(t / 0.12)
		if i > 2 {
			i = 2
		}
		return sin(notes[i], t) * exp(9, t-float32(i)*0.12) * 0.5
	})
	a.Wave = a.synth(0.7, func(t float32) float32 {
		f := float32(220)
		if t > 0.3 {
			f = 330
		}
		return (saw(f, t)*0.4 + sin(f/2, t)*0.4) * (1 - t/0.7) * 0.8
	})
	a.Clear = a.synth(0.8, func(t float32) float32 {
		notes := []float32{392, 523, 659, 784}
		i := min(int(t/0.18), 3)
		return sin(notes[i], t) * exp(6, t-float32(i)*0.18) * 0.5
	})
	a.GameOver = a.synth(1.2, func(t float32) float32 {
		f := 260 - 180*t
		return (saw(f, t)*0.5 + sin(f/2, t)*0.4) * exp(2.5, t)
	})
	a.Dig = a.synth(0.12, func(t float32) float32 { return (noise()*0.6 + sin(140, t)*0.4) * exp(30, t) * 0.7 })
	a.Place = a.synth(0.09, func(t float32) float32 { return (sin(420, t)*0.5 + noise()*0.3) * exp(50, t) * 0.6 })
	a.Explode = a.synth(1.1, func(t float32) float32 {
		return (noise()*0.9 + sin(45, t)*0.8 + sin(90, t)*0.3) * exp(4, t)
	})
	a.Fuse = a.synth(1.4, func(t float32) float32 { return noise() * (0.25 + 0.35*t) * float32(math.Min(1, float64(t*8))) })
	a.Craft = a.synth(0.3, func(t float32) float32 {
		v := noise() * exp(40, t) * 0.5
		if t > 0.12 {
			v += sin(880, t) * exp(20, t-0.12) * 0.4
		}
		return v
	})
	a.Swing = a.synth(0.18, func(t float32) float32 { return noise() * exp(18, t) * (0.2 + 0.5*float32(math.Sin(float64(t*17)))) })
	a.Burn = a.synth(0.4, func(t float32) float32 { return noise() * exp(8, t) * 0.35 })
	a.Eat = a.synth(0.5, func(t float32) float32 {
		i := int(t / 0.17)
		return noise() * exp(25, t-float32(i)*0.17) * 0.4
	})
	a.Rain = a.synth(1.0, func(t float32) float32 {
		env := float32(math.Sin(float64(t * math.Pi)))
		return noise() * 0.25 * (0.4 + 0.6*env)
	})
	a.Roar = a.synth(1.4, func(t float32) float32 {
		f := 70 + 40*float32(math.Sin(float64(t*4))) - 20*t
		env := float32(math.Sin(float64(t / 1.4 * math.Pi)))
		return (saw(f, t)*0.5 + saw(f*1.5, t)*0.3 + noise()*0.25) * env * 0.9
	})
	a.Music = a.synthMusic()
	a.Groan = a.synth(0.9, func(t float32) float32 {
		f := 95 - 25*t
		return (saw(f, t)*0.5 + sin(f*2.01, t)*0.3 + noise()*0.1) * float32(math.Sin(float64(t/0.9*math.Pi))) * 0.6
	})
	a.Rattle = a.synth(0.5, func(t float32) float32 {
		i := int(t / 0.07)
		return noise() * exp(60, t-float32(i)*0.07) * 0.35
	})
	a.Steps[0] = a.synth(0.08, func(t float32) float32 { return (noise()*0.5 + sin(140, t)*0.3) * exp(45, t) * 0.5 })
	a.Steps[1] = a.synth(0.06, func(t float32) float32 { return (noise()*0.7 + sin(900, t)*0.2) * exp(70, t) * 0.45 })
	a.Steps[2] = a.synth(0.1, func(t float32) float32 { return noise() * exp(35, t) * 0.35 })
	a.Steps[3] = a.synth(0.08, func(t float32) float32 { return (sin(210, t)*0.6 + noise()*0.3) * exp(50, t) * 0.5 })
	a.Splash = a.synth(0.35, func(t float32) float32 { return (noise()*0.6 + sin(300-200*t, t)*0.4) * exp(9, t) * 0.7 })
	return a
}

// synthMusic composes a gentle 32-second pentatonic loop: a soft lead with a
// slow bass, enough to give the world an atmosphere without any asset files.
func (a *Audio) synthMusic() rl.Sound {
	scale := []float32{261.63, 293.66, 329.63, 392.00, 440.00, 523.25, 587.33, 659.25}
	const beat = 0.5
	const bars = 64
	notes := make([]float32, bars)
	idx := 3
	seed := uint32(20240)
	next := func() uint32 { seed = seed*1664525 + 1013904223; return seed >> 8 }
	for i := range notes {
		step := int(next()%5) - 2
		idx = max(0, min(len(scale)-1, idx+step))
		if next()%4 == 0 {
			notes[i] = 0 // rest
		} else {
			notes[i] = scale[idx]
		}
	}
	bass := []float32{130.81, 130.81, 98.00, 110.00}
	dur := float32(bars) * beat
	return a.synth(dur, func(t float32) float32 {
		i := int(t / beat)
		if i >= bars {
			i = bars - 1
		}
		nt := t - float32(i)*beat
		v := float32(0)
		if f := notes[i]; f > 0 {
			env := float32(math.Exp(-float64(nt*3))) * float32(math.Min(1, float64(nt*40)))
			v += (float32(math.Sin(2*math.Pi*float64(f*t))) + 0.3*float32(math.Sin(4*math.Pi*float64(f*t)))) * env * 0.18
		}
		b := bass[(i/8)%len(bass)]
		v += float32(math.Sin(2*math.Pi*float64(b*t))) * 0.07 * (0.6 + 0.4*float32(math.Sin(float64(t*0.5))))
		return v
	})
}

// synth renders fn over dur seconds into a 16-bit mono sound.
func (a *Audio) synth(dur float32, fn func(t float32) float32) rl.Sound {
	n := int(dur * sampleRate)
	data := make([]byte, n*2)
	for i := 0; i < n; i++ {
		v := clamp(fn(float32(i)/sampleRate), -1, 1)
		binary.LittleEndian.PutUint16(data[i*2:], uint16(int16(v*32000)))
	}
	w := rl.NewWave(uint32(n), sampleRate, 16, 1, data)
	s := rl.LoadSoundFromWave(w)
	a.sounds = append(a.sounds, s)
	return s
}

// Play triggers s at the given volume with slight random pitch variation.
func (a *Audio) Play(s rl.Sound, vol float32) {
	if !a.ok {
		return
	}
	rl.SetSoundVolume(s, vol*settings.Volume)
	rl.SetSoundPitch(s, 0.92+rand.Float32()*0.16)
	rl.PlaySound(s)
}

func (a *Audio) Close() {
	if !a.ok {
		return
	}
	for _, s := range a.sounds {
		rl.UnloadSound(s)
	}
	rl.CloseAudioDevice()
}
