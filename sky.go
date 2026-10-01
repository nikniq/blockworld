package main

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	dayLength = 360.0 // seconds for a full day/night cycle
	dayFrac   = 0.6   // fraction of the cycle with the sun up
	cloudY    = 64
	cloudCell = 8
)

// Sky tracks the time of day. T runs 0..1 from sunrise to sunrise.
type Sky struct {
	T        float32
	Day      int
	Rain     float32 // 0 clear .. 1 pouring (smoothed)
	Raining  bool
	WeatherT float32 // seconds until the weather changes
	stars    []rl.Vector3
}

func NewSky() *Sky {
	s := &Sky{T: 0.08, Day: 1, WeatherT: 150 + rand.Float32()*200}
	for i := 0; i < 220; i++ {
		d := rl.NewVector3(hash2(i, 1, 77)*2-1, hash2(i, 2, 77)*0.9+0.08, hash2(i, 3, 77)*2-1)
		s.stars = append(s.stars, rl.Vector3Normalize(d))
	}
	return s
}

func (s *Sky) Update(dt float32) {
	s.T += dt / dayLength
	if s.T >= 1 {
		s.T -= 1
		s.Day++
	}
	s.WeatherT -= dt
	if s.WeatherT <= 0 {
		s.Raining = !s.Raining
		if s.Raining {
			s.WeatherT = 60 + rand.Float32()*90
		} else {
			s.WeatherT = 180 + rand.Float32()*300
		}
	}
	target := float32(0)
	if s.Raining {
		target = 1
	}
	s.Rain = lerp(s.Rain, target, dt*0.5)
}

// angle maps T onto the sun's arc: 0..pi during the day, pi..2pi at night.
func (s *Sky) angle() float64 {
	if s.T < dayFrac {
		return math.Pi * float64(s.T) / dayFrac
	}
	return math.Pi + math.Pi*float64(s.T-dayFrac)/(1-dayFrac)
}

// Elevation is the sun height, -1..1; negative at night.
func (s *Sky) Elevation() float32 { return float32(math.Sin(s.angle())) }

func (s *Sky) IsNight() bool { return s.Elevation() < 0 }

// SunDir points from the viewer toward the sun.
func (s *Sky) SunDir() rl.Vector3 {
	a := s.angle()
	return rl.Vector3Normalize(rl.NewVector3(float32(math.Cos(a)), float32(math.Sin(a)), 0.35))
}

// Light is the sunlight strength, 1 at noon down to moonlight at night.
func (s *Sky) Light() float32 {
	e := s.Elevation()
	return (0.32 + 0.68*smooth(clamp((e+0.12)/0.45, 0, 1))) * (1 - 0.3*s.Rain)
}

// Color is the sky and fog colour for the current time.
func (s *Sky) Color() rl.Color {
	e := s.Elevation()
	night := rl.NewColor(8, 10, 26, 255)
	dusk := rl.NewColor(232, 132, 88, 255)
	day := rl.NewColor(118, 178, 235, 255)
	switch {
	case e < -0.3:
		return night
	case e < 0:
		return mix(night, dusk, (e+0.3)/0.3)
	case e < 0.3:
		return s.overcast(mix(dusk, day, e/0.3))
	}
	return s.overcast(day)
}

// overcast greys the sky while it rains.
func (s *Sky) overcast(c rl.Color) rl.Color {
	return mix(c, rl.NewColor(120, 125, 135, 255), s.Rain*0.8)
}

// Env builds the shader environment for the frame.
func (s *Sky) Env(underwater bool) Env {
	env := Env{Light: s.Light(), Fog: s.Color(), FogStart: 48 - 20*s.Rain, FogEnd: 115 - 45*s.Rain}
	if underwater {
		env.Fog = rl.NewColor(16, 50, 110, 255)
		env.FogStart, env.FogEnd = 1, 22
		env.Light *= 0.8
	}
	return env
}

// TimeLabel names the part of the day for the HUD.
func (s *Sky) TimeLabel() string {
	switch e := s.Elevation(); {
	case e < 0:
		return "Night"
	case s.T < 0.1:
		return "Sunrise"
	case s.T > dayFrac-0.08:
		return "Sunset"
	}
	return "Day"
}

// DrawSky draws the sun, moon and stars far away around the camera (before the world).
func (s *Sky) DrawSky(cam rl.Camera3D) {
	eye := cam.Position
	sun := s.SunDir()
	sunPos := rl.Vector3Add(eye, rl.Vector3Scale(sun, 180))
	rl.DrawCube(sunPos, 12, 12, 12, rl.NewColor(255, 240, 180, 255))
	rl.DrawCube(sunPos, 16, 16, 16, rl.NewColor(255, 220, 120, 60))
	moonPos := rl.Vector3Add(eye, rl.Vector3Scale(sun, -180))
	rl.DrawCube(moonPos, 9, 9, 9, rl.NewColor(215, 220, 235, 255))
	rl.DrawCube(rl.Vector3Add(moonPos, rl.NewVector3(-2, 2, 2)), 5, 5, 5, rl.NewColor(160, 165, 185, 255))
	if a := clamp(-s.Elevation()*4, 0, 1); a > 0 {
		c := rl.NewColor(255, 255, 255, uint8(220*a))
		for _, d := range s.stars {
			rl.DrawCube(rl.Vector3Add(eye, rl.Vector3Scale(d, 190)), 0.9, 0.9, 0.9, c)
		}
	}
}

// DrawRain draws falling streaks around the camera; call when the player is under open sky.
func (s *Sky) DrawRain(cam rl.Camera3D, t float32) {
	if s.Rain < 0.05 {
		return
	}
	n := int(260 * s.Rain)
	c := rl.NewColor(180, 200, 235, uint8(140*s.Rain))
	for i := 0; i < n; i++ {
		ox := hash2(i, 1, 55)*16 - 8
		oz := hash2(i, 2, 55)*16 - 8
		fall := float32(math.Mod(float64(t*14+hash2(i, 3, 55)*20), 20))
		y := cam.Position.Y + 10 - fall
		p := rl.NewVector3(cam.Position.X+ox, y, cam.Position.Z+oz)
		rl.DrawLine3D(p, rl.NewVector3(p.X, p.Y-0.9, p.Z), c)
	}
}

// DrawClouds draws a drifting layer of blocky clouds above the world.
func (s *Sky) DrawClouds(cam rl.Camera3D, t float32) {
	drift := t * 0.7
	px, pz := cam.Position.X, cam.Position.Z
	ci0 := floorI((px + drift) / cloudCell)
	cj0 := floorI(pz / cloudCell)
	c := mul(rl.NewColor(255, 255, 255, 205), brightness(s.Light()))
	for cj := cj0 - 14; cj <= cj0+14; cj++ {
		for ci := ci0 - 14; ci <= ci0+14; ci++ {
			if vnoise(float32(ci)/3.5, float32(cj)/3.5, 31) < 0.58 || hash2(ci, cj, 32) < 0.2 {
				continue
			}
			x := (float32(ci)+0.5)*cloudCell - drift
			z := (float32(cj) + 0.5) * cloudCell
			rl.DrawCube(rl.NewVector3(x, cloudY, z), cloudCell, 3, cloudCell, c)
		}
	}
}
