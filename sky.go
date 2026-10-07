package main

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	dayLength = 360.0 // seconds for a full day/night cycle
	dayFrac   = 0.6   // fraction of the cycle with the sun up
	cloudY    = 84
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

// SunTint is the colour of sunlight: warm at dawn and dusk, white by day, blue at night.
func (s *Sky) SunTint() [3]float32 {
	e := s.Elevation()
	day := [3]float32{1, 1, 1}
	dusk := [3]float32{1, 0.72, 0.5}
	night := [3]float32{0.62, 0.7, 1}
	lerp3 := func(a, b [3]float32, t float32) [3]float32 {
		t = clamp(t, 0, 1)
		return [3]float32{a[0] + (b[0]-a[0])*t, a[1] + (b[1]-a[1])*t, a[2] + (b[2]-a[2])*t}
	}
	switch {
	case e < -0.2:
		return night
	case e < 0.05:
		return lerp3(night, dusk, (e+0.2)/0.25)
	case e < 0.35:
		return lerp3(dusk, day, (e-0.05)/0.3)
	}
	return day
}

// Env builds the shader environment for the frame.
func (s *Sky) Env(underwater bool, t float32) Env {
	flicker := 1 + 0.05*float32(math.Sin(float64(t*9))) + 0.03*float32(math.Sin(float64(t*23.7)))
	env := Env{Light: s.Light(), Fog: s.Color(), FogStart: 90 - 40*s.Rain, FogEnd: 190 - 80*s.Rain, SunTint: s.SunTint(), SunDir: s.SunDir(), Flicker: flicker, Time: t}
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

// DrawDome paints a gradient sky with a glow around the sun, far behind everything.
func (s *Sky) DrawDome(cam rl.Camera3D) {
	eye := cam.Position
	horizon := s.Color()
	zenith := mix(horizon, rl.NewColor(40, 90, 200, 255), 0.55*(1-s.Rain))
	if s.Elevation() < 0 {
		zenith = mix(horizon, rl.NewColor(2, 3, 12, 255), 0.7)
	}
	sun := s.SunDir()
	glow := rl.NewColor(255, 215, 140, 255)
	if s.Elevation() < 0.25 {
		glow = rl.NewColor(255, 150, 80, 255)
	}
	const segs, rings = 24, 6
	const r = 420
	colorAt := func(elev, az float32) rl.Color {
		t := clamp(elev/1.2, 0, 1)
		c := mix(horizon, zenith, t*t)
		dir := rl.NewVector3(float32(math.Cos(float64(az)))*float32(math.Cos(float64(elev))), float32(math.Sin(float64(elev))), float32(math.Sin(float64(az)))*float32(math.Cos(float64(elev))))
		d := rl.Vector3DotProduct(dir, sun)
		if d > 0.6 && s.Elevation() > -0.25 {
			c = mix(c, glow, (d-0.6)/0.4*0.6*(1-s.Rain))
		}
		return c
	}
	rl.DisableBackfaceCulling()
	rl.Begin(rl.Triangles)
	for i := 0; i < rings; i++ {
		e0 := -0.15 + float32(i)*(math.Pi/2+0.15)/rings
		e1 := -0.15 + float32(i+1)*(math.Pi/2+0.15)/rings
		for j := 0; j < segs; j++ {
			a0 := float32(j) * 2 * math.Pi / segs
			a1 := float32(j+1) * 2 * math.Pi / segs
			pt := func(e, a float32) rl.Vector3 {
				return rl.NewVector3(eye.X+r*float32(math.Cos(float64(a))*math.Cos(float64(e))), eye.Y+r*float32(math.Sin(float64(e))), eye.Z+r*float32(math.Sin(float64(a))*math.Cos(float64(e))))
			}
			v := func(e, a float32) {
				c := colorAt(e, a)
				rl.Color4ub(c.R, c.G, c.B, 255)
				p := pt(e, a)
				rl.Vertex3f(p.X, p.Y, p.Z)
			}
			v(e0, a0)
			v(e1, a0)
			v(e1, a1)
			v(e0, a0)
			v(e1, a1)
			v(e0, a1)
		}
	}
	rl.End()
	rl.EnableBackfaceCulling()
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
func (s *Sky) DrawRain(cam rl.Camera3D, t float32, snow bool) {
	if s.Rain < 0.05 {
		return
	}
	n := int(260 * s.Rain)
	c := rl.NewColor(180, 200, 235, uint8(140*s.Rain))
	speed := float32(14)
	if snow {
		c = rl.NewColor(255, 255, 255, uint8(220*s.Rain))
		speed = 2.5
		n = n * 2 / 3
	}
	for i := 0; i < n; i++ {
		ox := hash2(i, 1, 55)*16 - 8
		oz := hash2(i, 2, 55)*16 - 8
		fall := float32(math.Mod(float64(t*speed+hash2(i, 3, 55)*20), 20))
		y := cam.Position.Y + 10 - fall
		p := rl.NewVector3(cam.Position.X+ox, y, cam.Position.Z+oz)
		if snow {
			drift := float32(math.Sin(float64(t*1.5+float32(i)))) * 0.3
			rl.DrawCube(rl.NewVector3(p.X+drift, p.Y, p.Z), 0.08, 0.08, 0.08, c)
		} else {
			rl.DrawLine3D(p, rl.NewVector3(p.X, p.Y-0.9, p.Z), c)
		}
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
