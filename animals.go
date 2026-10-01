package main

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Passive animals wander the surface by day and drop meat when killed.

type AnimalKind int

const (
	AnimalPig AnimalKind = iota
	AnimalCow
	AnimalSheep
	numAnimalKinds
)

type animalSpec struct {
	Name   string
	HP     int
	Speed  float32
	Radius float32
	Height float32
	Body   rl.Color
	Head   rl.Color
	Legs   rl.Color
	Meat   int
}

var animalKinds = [...]animalSpec{
	AnimalPig:   {"Pig", 4, 1.6, 0.35, 0.9, rl.NewColor(235, 160, 170, 255), rl.NewColor(240, 170, 180, 255), rl.NewColor(220, 140, 150, 255), 2},
	AnimalCow:   {"Cow", 6, 1.4, 0.4, 1.3, rl.NewColor(80, 55, 40, 255), rl.NewColor(90, 65, 50, 255), rl.NewColor(60, 40, 30, 255), 3},
	AnimalSheep: {"Sheep", 4, 1.5, 0.4, 1.1, rl.NewColor(230, 230, 225, 255), rl.NewColor(70, 60, 55, 255), rl.NewColor(60, 55, 50, 255), 1},
}

type Animal struct {
	Kind    AnimalKind
	Spec    *animalSpec
	Pos     rl.Vector3
	Heading rl.Vector3
	VelY    float32
	HP      int
	WanderT float32
	Walking bool
	Flee    float32 // seconds left running from the player
	Phase   float32
	Alive   bool
	DeathT  float32
	Lum     float32
}

func NewAnimal(pos rl.Vector3, kind AnimalKind) *Animal {
	s := &animalKinds[kind]
	a := rand.Float64() * 2 * math.Pi
	return &Animal{Kind: kind, Spec: s, Pos: pos, HP: s.HP, Alive: true, Lum: 1,
		Heading: rl.NewVector3(float32(math.Sin(a)), 0, float32(math.Cos(a)))}
}

func (a *Animal) BB() rl.BoundingBox {
	r := a.Spec.Radius
	return rl.NewBoundingBox(rl.NewVector3(a.Pos.X-r, a.Pos.Y, a.Pos.Z-r), rl.NewVector3(a.Pos.X+r, a.Pos.Y+a.Spec.Height, a.Pos.Z+r))
}

func (a *Animal) Update(dt float32, w *World, p *Player) {
	if !a.Alive {
		a.DeathT += dt
		return
	}
	a.Flee = max(0, a.Flee-dt)
	a.WanderT -= dt
	if a.WanderT <= 0 {
		a.WanderT = 1 + rand.Float32()*4
		a.Walking = rand.Float32() < 0.55
		ang := rand.Float64() * 2 * math.Pi
		a.Heading = rl.NewVector3(float32(math.Sin(ang)), 0, float32(math.Cos(ang)))
	}
	speed := float32(0)
	if a.Flee > 0 {
		// Run away from the player.
		away := rl.Vector3Subtract(a.Pos, p.Pos)
		away.Y = 0
		if l := rl.Vector3Length(away); l > 0.01 {
			a.Heading = rl.Vector3Scale(away, 1/l)
		}
		speed = a.Spec.Speed * 2.4
	} else if a.Walking {
		speed = a.Spec.Speed
	}
	inWater := w.WaterAt(rl.NewVector3(a.Pos.X, a.Pos.Y+0.3, a.Pos.Z))
	if inWater {
		speed *= 0.5
		a.VelY = max(a.VelY-gravity*0.3*dt, -1.5)
		if w.WaterAt(rl.NewVector3(a.Pos.X, a.Pos.Y+a.Spec.Height*0.8, a.Pos.Z)) {
			a.VelY = min(a.VelY+14*dt, 2.5)
		}
	} else {
		a.VelY = max(a.VelY-gravity*dt, -30)
	}
	if speed > 0 {
		a.Phase += dt * speed * 2.5
	}
	delta := rl.NewVector3(a.Heading.X*speed*dt, a.VelY*dt, a.Heading.Z*speed*dt)
	var res MoveResult
	a.Pos, res = w.MoveBox(a.Pos, a.Spec.Radius, a.Spec.Height, delta, true)
	if res.Ground || res.Ceiling {
		a.VelY = 0
	}
	if res.Wall {
		a.WanderT = 0 // pick a new direction
	}
	// Do not wander into deep water.
	if inWater && a.Flee == 0 && !w.Solid(floorI(a.Pos.X), floorI(a.Pos.Y)-1, floorI(a.Pos.Z)) {
		a.Heading = rl.Vector3Scale(a.Heading, -1)
		a.Walking = true
	}
}

// Hit damages the animal and sends it running; returns true if it died.
func (a *Animal) Hit(dmg int) bool {
	if !a.Alive {
		return false
	}
	a.HP -= dmg
	a.Flee = 5
	if a.HP <= 0 {
		a.Alive = false
		return true
	}
	return false
}

func (a *Animal) Draw() {
	s := a.Spec
	x, z := a.Pos.X, a.Pos.Z
	if !a.Alive {
		t := clamp(1-a.DeathT*2.5, 0, 1)
		if t <= 0 {
			return
		}
		rl.DrawCubeV(rl.NewVector3(x, a.Pos.Y+0.3*t, z), rl.NewVector3(s.Radius*2.4, 0.6*t+0.05, s.Radius*2.4), mul(rl.NewColor(s.Body.R, s.Body.G, s.Body.B, uint8(200*t)), a.Lum))
		return
	}
	body, head, legs := mul(s.Body, a.Lum), mul(s.Head, a.Lum), mul(s.Legs, a.Lum)
	outline := rl.NewColor(20, 20, 20, 255)
	fwd := a.Heading
	side := rl.NewVector3(-fwd.Z, 0, fwd.X)
	h := s.Height
	legH := h * 0.4
	bodyLen, bodyW := s.Radius*2.6, s.Radius*1.7
	// Legs.
	for _, d := range [][2]float32{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
		swing := float32(math.Sin(float64(a.Phase)))*0.12*d[0]*d[1] + 0.2
		lp := rl.Vector3Add(rl.NewVector3(x, a.Pos.Y+legH/2, z), rl.Vector3Add(rl.Vector3Scale(side, d[0]*bodyW*0.35), rl.Vector3Scale(fwd, d[1]*bodyLen*0.35+swing*0.3-0.06)))
		rl.DrawCubeV(lp, rl.NewVector3(0.2, legH, 0.2), legs)
	}
	// Body: axis-aligned box scaled by the heading so it looks roughly oriented.
	bx := float32(math.Abs(float64(fwd.X)))*bodyLen + float32(math.Abs(float64(fwd.Z)))*bodyW
	bz := float32(math.Abs(float64(fwd.Z)))*bodyLen + float32(math.Abs(float64(fwd.X)))*bodyW
	bc := rl.NewVector3(x, a.Pos.Y+legH+(h-legH)*0.5, z)
	rl.DrawCubeV(bc, rl.NewVector3(bx, h-legH, bz), body)
	rl.DrawCubeWiresV(bc, rl.NewVector3(bx, h-legH, bz), outline)
	// Head.
	hs := s.Radius * 1.3
	hp := rl.Vector3Add(rl.NewVector3(x, a.Pos.Y+h-hs*0.4, z), rl.Vector3Scale(fwd, bodyLen*0.5+hs*0.3))
	rl.DrawCubeV(hp, rl.NewVector3(hs, hs, hs), head)
	rl.DrawCubeWiresV(hp, rl.NewVector3(hs, hs, hs), outline)
	if a.Kind == AnimalPig {
		snout := rl.Vector3Add(hp, rl.Vector3Scale(fwd, hs/2))
		rl.DrawCubeV(snout, rl.NewVector3(hs*0.4, hs*0.3, 0.08), mul(rl.NewColor(210, 120, 140, 255), a.Lum))
	}
	if a.HP < s.HP {
		top := a.Pos.Y + h + 0.2
		frac := float32(a.HP) / float32(s.HP)
		rl.DrawCubeV(rl.NewVector3(x, top, z), rl.NewVector3(1.0, 0.08, 0.08), rl.NewColor(0, 0, 0, 180))
		rl.DrawCubeV(rl.NewVector3(x-(1-frac)*0.5, top, z), rl.NewVector3(frac, 0.1, 0.1), rl.Lime)
	}
}

func (g *Game) spawnAnimals(n int) {
	for i := 0; i < n; i++ {
		p := g.World.RandomFreePoint(g.Player.Pos, 6)
		if g.World.Get(floorI(p.X), floorI(p.Y)-1, floorI(p.Z)) != Grass {
			continue
		}
		g.Animals = append(g.Animals, NewAnimal(p, AnimalKind(rand.Intn(int(numAnimalKinds)))))
	}
}

func (g *Game) updateAnimals(dt float32) {
	alive := 0
	keep := g.Animals[:0]
	for _, a := range g.Animals {
		a.Update(dt, g.World, g.Player)
		if a.Alive {
			alive++
		}
		if a.Alive || a.DeathT < 1 {
			keep = append(keep, a)
		}
	}
	g.Animals = keep
	// Herds slowly recover during the day.
	g.AnimalCD -= dt
	if g.AnimalCD <= 0 {
		g.AnimalCD = 25
		if alive < 10 && !g.Sky.IsNight() {
			g.spawnAnimals(2)
		}
	}
}

func (g *Game) killAnimal(a *Animal) {
	g.Score += 10
	g.Audio.Play(g.Audio.Die, 0.4)
	c := rl.NewVector3(a.Pos.X, a.Pos.Y+0.4, a.Pos.Z)
	for i := 0; i < a.Spec.Meat; i++ {
		g.spawnDrop(c, Meat, 0)
	}
	switch a.Kind {
	case AnimalSheep:
		g.spawnDrop(c, Wool, 0)
		g.spawnDrop(c, Wool, 0)
	case AnimalCow:
		g.spawnDrop(c, Leather, 0)
		if rand.Float32() < 0.5 {
			g.spawnDrop(c, Leather, 0)
		}
	}
}
