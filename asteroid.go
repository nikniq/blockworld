package main

import (
	"fmt"
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Asteroids: now and then a rock falls out of the sky. A warning names the
// bearing and distance and counts down; the rock streaks in with a fiery
// trail, blasts a crater, and leaves meteorite blocks worth mining.

const (
	asteroidWarning = 20.0 // seconds of warning
	asteroidEvery   = 420.0
)

type Asteroid struct {
	Target rl.Vector3 // impact point
	T      float32    // seconds until impact
	Active bool
}

// tickAsteroid schedules and runs the event (host and solo only).
func (g *Game) tickAsteroid(dt float32) {
	if settings.Difficulty == 0 && !g.Asteroid.Active {
		return
	}
	a := &g.Asteroid
	if !a.Active {
		g.AsteroidCD -= dt
		if g.AsteroidCD > 0 {
			return
		}
		g.AsteroidCD = asteroidEvery*0.6 + rand.Float32()*asteroidEvery*0.8
		// Somewhere within a hundred blocks of a player, on the surface.
		ts := g.targets()
		ref := ts[rand.Intn(len(ts))].Pos
		ang := rand.Float64() * 2 * math.Pi
		dist := 25 + rand.Float64()*75
		x, z := ref.X+float32(math.Cos(ang)*dist), ref.Z+float32(math.Sin(ang)*dist)
		y := float32(g.World.SurfaceY(floorI(x), floorI(z)))
		a.Target = WrapPos(rl.NewVector3(x, y, z))
		a.T = asteroidWarning
		a.Active = true
		g.announce("WARNING: asteroid inbound!  "+g.asteroidBearing(), 4)
		g.Audio.Play(g.Audio.Alarm, 0.9)
		g.sendFx(Fx{Kind: FxAsteroid, Pos: a.Target, Shake: asteroidWarning})
		return
	}
	a.T -= dt
	if a.T > 0 {
		if int(a.T+dt) != int(a.T) && int(a.T) <= 5 && int(a.T) >= 1 {
			g.Audio.Play(g.Audio.Click, 0.8)
		}
		return
	}
	a.Active = false
	g.asteroidImpact(a.Target)
}

func (g *Game) asteroidBearing() string {
	d := WrapDelta(g.Asteroid.Target, g.Player.Pos)
	dist := int(math.Sqrt(float64(d.X*d.X + d.Z*d.Z)))
	deg := math.Atan2(float64(d.X), float64(-d.Z)) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	dirs := [...]string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	return fmt.Sprintf("impact %dm %s in %ds", dist, dirs[int((deg+22.5)/45)%8], int(math.Ceil(float64(g.Asteroid.T))))
}

// asteroidImpact blasts the crater and seeds it with meteorite.
func (g *Game) asteroidImpact(at rl.Vector3) {
	g.blast(rl.Vector3Add(at, rl.NewVector3(0, 1, 0)), 5.5, 90)
	cx, cy, cz := floorI(at.X), floorI(at.Y), floorI(at.Z)
	n := 0
	for try := 0; try < 60 && n < 7+rand.Intn(6); try++ {
		x := cx + rand.Intn(9) - 4
		z := cz + rand.Intn(9) - 4
		for y := cy + 2; y > cy-8 && y > 1; y-- {
			if g.World.Get(x, y, z) == Air && blocks[g.World.Get(x, y-1, z)].Solid {
				g.World.Set(x, y, z, Meteorite)
				n++
				break
			}
		}
	}
	for i := 0; i < 40; i++ {
		v := rl.NewVector3(rand.Float32()*2-1, 3+rand.Float32()*5, rand.Float32()*2-1)
		g.Sparks = append(g.Sparks, Spark{Pos: at, Vel: rl.Vector3Scale(v, 2), Life: 1.2 + rand.Float32(), Col: rl.NewColor(255, 120+uint8(rand.Intn(100)), 40, 255)})
	}
	g.announce("Impact! Meteorite lies in the crater.", 4)
	g.Shake = max(g.Shake, 1.5)
	g.unlock(AchMeteor)
}

// drawAsteroid draws the falling rock and its trail during the countdown.
func (g *Game) drawAsteroid(cam rl.Camera3D) {
	a := &g.Asteroid
	if !a.Active {
		return
	}
	tgt := Near(a.Target, cam.Position)
	f := a.T / asteroidWarning // 1 at warning, 0 at impact
	// Comes in from high up and to the side, straight at the target.
	start := rl.Vector3Add(tgt, rl.NewVector3(90, 160, 60))
	pos := rl.Vector3Lerp(tgt, start, f*f)
	rl.DrawSphere(pos, 1.6, rl.NewColor(90, 70, 60, 255))
	rl.DrawSphere(pos, 2.2, rl.NewColor(255, 140, 40, 120))
	for i := 1; i <= 10; i++ {
		t := float32(i) / 10
		p := rl.Vector3Lerp(pos, rl.Vector3Lerp(tgt, start, (f+0.08*t)*(f+0.08*t)), 1)
		rl.DrawSphere(p, 2.2*(1-t*0.8), rl.NewColor(255, 200-uint8(t*120), 60, uint8(140*(1-t))))
	}
}
