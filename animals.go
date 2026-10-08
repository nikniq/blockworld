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
	AnimalDino
	AnimalTrader
	AnimalVillager
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
	Damage int // bite damage when provoked (0: harmless)
}

var animalKinds = [...]animalSpec{
	AnimalPig:      {"Pig", 4, 1.6, 0.35, 0.9, rl.NewColor(235, 160, 170, 255), rl.NewColor(240, 170, 180, 255), rl.NewColor(220, 140, 150, 255), 2, 0},
	AnimalCow:      {"Cow", 6, 1.4, 0.4, 1.3, rl.NewColor(80, 55, 40, 255), rl.NewColor(90, 65, 50, 255), rl.NewColor(60, 40, 30, 255), 3, 0},
	AnimalSheep:    {"Sheep", 4, 1.5, 0.4, 1.1, rl.NewColor(230, 230, 225, 255), rl.NewColor(70, 60, 55, 255), rl.NewColor(60, 55, 50, 255), 1, 0},
	AnimalDino:     {"Dinosaur", 45, 2.2, 0.7, 2.8, rl.NewColor(70, 120, 60, 255), rl.NewColor(80, 130, 65, 255), rl.NewColor(60, 100, 50, 255), 8, 18},
	AnimalTrader:   {"Wandering Trader", 20, 1.2, 0.3, 1.8, rl.NewColor(90, 60, 130, 255), rl.NewColor(205, 160, 120, 255), rl.NewColor(60, 40, 90, 255), 0, 0},
	AnimalVillager: {"Villager", 20, 1.3, 0.3, 1.8, rl.NewColor(150, 110, 70, 255), rl.NewColor(205, 160, 120, 255), rl.NewColor(80, 60, 40, 255), 0, 0},
}

type Animal struct {
	ID       uint32
	Kind     AnimalKind
	Spec     *animalSpec
	Pos      rl.Vector3
	Heading  rl.Vector3
	VelY     float32
	HP       int
	WanderT  float32
	Walking  bool
	Flee     float32 // seconds left running from the player (or charging, for a dinosaur)
	BiteCD   float32
	RoarCD   float32
	StepFlag int
	Visit    float32 // seconds a trader has been around
	// Villagers.
	Home       rl.Vector3
	Prof       Profession
	Name       string
	Quest      int
	QuestDone  bool
	QuestKills int // kills when a kill quest was accepted (-1: not yet)
	Phase      float32
	Alive      bool
	DeathT     float32
	Lum        float32
}

func NewAnimal(pos rl.Vector3, kind AnimalKind) *Animal {
	s := &animalKinds[kind]
	a := rand.Float64() * 2 * math.Pi
	return &Animal{Kind: kind, Spec: s, Pos: pos, HP: s.HP, Alive: true, Lum: 1, QuestKills: -1,
		Heading: rl.NewVector3(float32(math.Sin(a)), 0, float32(math.Cos(a)))}
}

func (a *Animal) BB() rl.BoundingBox {
	r := a.Spec.Radius
	return rl.NewBoundingBox(rl.NewVector3(a.Pos.X-r, a.Pos.Y, a.Pos.Z-r), rl.NewVector3(a.Pos.X+r, a.Pos.Y+a.Spec.Height, a.Pos.Z+r))
}

// Update moves the animal; returns bite damage dealt to the target this frame.
func (a *Animal) Update(dt float32, w *World, p Target) int {
	if !a.Alive {
		a.DeathT += dt
		return 0
	}
	a.Flee = max(0, a.Flee-dt)
	a.BiteCD = max(0, a.BiteCD-dt)
	a.RoarCD = max(0, a.RoarCD-dt)
	bite := 0
	a.WanderT -= dt
	if a.WanderT <= 0 {
		a.WanderT = 1 + rand.Float32()*4
		a.Walking = rand.Float32() < 0.55
		ang := rand.Float64() * 2 * math.Pi
		a.Heading = rl.NewVector3(float32(math.Sin(ang)), 0, float32(math.Cos(ang)))
	}
	speed := float32(0)
	if a.Flee > 0 && a.Spec.Damage > 0 {
		// A provoked dinosaur charges and bites instead of fleeing.
		to := WrapDelta(p.Pos, a.Pos)
		to.Y = 0
		dist := rl.Vector3Length(to)
		if dist > 0.01 {
			a.Heading = rl.Vector3Scale(to, 1/dist)
		}
		speed = a.Spec.Speed * 1.9
		if dist < a.Spec.Radius+1.4 {
			speed = 0
			if a.BiteCD == 0 && math.Abs(float64(p.Pos.Y-a.Pos.Y)) < 3 {
				a.BiteCD = 1.3
				bite = a.Spec.Damage
			}
		}
	} else if a.Flee > 0 {
		// Run away from the player.
		away := WrapDelta(a.Pos, p.Pos)
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
	a.Pos = WrapPos(a.Pos)
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
	return bite
}

// Hit damages the animal and sends it running; returns true if it died.
func (a *Animal) Hit(dmg int) bool {
	if !a.Alive {
		return false
	}
	a.HP -= dmg
	a.Flee = 5
	if a.Spec.Damage > 0 {
		a.Flee = 14 // dinosaurs hold a grudge
	}
	if a.HP <= 0 {
		a.Alive = false
		return true
	}
	return false
}

func (a *Animal) Draw(w *World) {
	kind := SkinPig
	scale := float32(1)
	switch a.Kind {
	case AnimalCow:
		kind, scale = SkinCow, 0.93
	case AnimalSheep:
		kind, scale = SkinSheep, 0.95
	case AnimalVillager:
		pose := Pose{Pos: a.Pos, Yaw: yawOf(a.Heading), Phase: a.Phase, Amp: 0, Scale: 0.9, Lum: a.Lum, Alpha: 1}
		if a.Walking || a.Flee > 0 {
			pose.Amp = 1
		}
		if !a.Alive {
			pose.Death = clamp(a.DeathT*2.5, 0, 1)
			pose.Alpha = 1 - pose.Death
			if pose.Death >= 1 {
				return
			}
		}
		if a.Flee > 3.7 {
			pose.Flash = (a.Flee - 3.7) / 0.3
		}
		skin := SkinFarmer
		switch a.Prof {
		case ProfGuard:
			skin = SkinGuard
			pose.Held = Item{Kind: ItemSword}
			pose.SwordTier = TierIron
			pose.Swing = clamp(a.BiteCD-0.8, 0, 0.4) / 0.4
		case ProfLibrarian:
			skin = SkinLibrarian
		}
		skins.DrawHumanoid(w, skin, &pose)
		if a.Alive && a.HP < a.Spec.HP {
			top := a.Pos.Y + a.Spec.Height + 0.2
			frac := float32(a.HP) / float32(a.Spec.HP)
			rl.DrawCubeV(rl.NewVector3(a.Pos.X, top, a.Pos.Z), rl.NewVector3(1.0, 0.08, 0.08), rl.NewColor(0, 0, 0, 180))
			rl.DrawCubeV(rl.NewVector3(a.Pos.X-(1-frac)*0.5, top, a.Pos.Z), rl.NewVector3(frac, 0.1, 0.1), rl.Lime)
		}
		return
	case AnimalTrader:
		pose := Pose{Pos: a.Pos, Yaw: yawOf(a.Heading), Phase: a.Phase, Amp: 0, Scale: 0.9, Lum: a.Lum, Alpha: 1}
		if a.Walking || a.Flee > 0 {
			pose.Amp = 1
		}
		if !a.Alive {
			pose.Death = clamp(a.DeathT*2.5, 0, 1)
			pose.Alpha = 1 - pose.Death
			if pose.Death >= 1 {
				return
			}
		}
		skins.DrawHumanoid(w, SkinTrader, &pose)
		return
	case AnimalDino:
		pose := Pose{Pos: a.Pos, Yaw: yawOf(a.Heading), Phase: a.Phase, Amp: 0, Scale: 1, Lum: a.Lum, Alpha: 1}
		if a.Walking || a.Flee > 0 {
			pose.Amp = 1
		}
		if !a.Alive {
			pose.Death = clamp(a.DeathT*2.5, 0, 1)
			pose.Alpha = 1 - pose.Death
			pose.Amp = 0
			if pose.Death >= 1 {
				return
			}
		}
		if a.Flee > 13.7 {
			pose.Flash = (a.Flee - 13.7) / 0.3
		}
		pose.Swing = clamp(a.BiteCD-0.9, 0, 0.4) / 0.4 // jaw snap just after a bite
		skins.DrawDino(w, &pose)
		if a.Alive && a.HP < a.Spec.HP {
			top := a.Pos.Y + a.Spec.Height + 0.2
			frac := float32(a.HP) / float32(a.Spec.HP)
			rl.DrawCubeV(rl.NewVector3(a.Pos.X, top, a.Pos.Z), rl.NewVector3(2.0, 0.1, 0.1), rl.NewColor(0, 0, 0, 180))
			rl.DrawCubeV(rl.NewVector3(a.Pos.X-(1-frac)*1.0, top, a.Pos.Z), rl.NewVector3(frac*2, 0.12, 0.12), rl.Lime)
		}
		return
	}
	amp := float32(0)
	if a.Walking || a.Flee > 0 {
		amp = 1
	}
	pose := Pose{Pos: a.Pos, Yaw: yawOf(a.Heading), Phase: a.Phase, Amp: amp, Scale: scale, Lum: a.Lum, Alpha: 1}
	if !a.Alive {
		pose.Death = clamp(a.DeathT*2.5, 0, 1)
		pose.Alpha = 1 - pose.Death
		pose.Amp = 0
		if pose.Death >= 1 {
			return
		}
	}
	if a.Flee > 4.7 {
		pose.Flash = (a.Flee - 4.7) / 0.3
	}
	skins.DrawQuadruped(w, kind, &pose)
	if a.Alive && a.HP < a.Spec.HP {
		top := a.Pos.Y + a.Spec.Height + 0.2
		frac := float32(a.HP) / float32(a.Spec.HP)
		rl.DrawCubeV(rl.NewVector3(a.Pos.X, top, a.Pos.Z), rl.NewVector3(1.0, 0.08, 0.08), rl.NewColor(0, 0, 0, 180))
		rl.DrawCubeV(rl.NewVector3(a.Pos.X-(1-frac)*0.5, top, a.Pos.Z), rl.NewVector3(frac, 0.1, 0.1), rl.Lime)
	}
}

func (g *Game) spawnAnimals(n int) {
	for i := 0; i < n; i++ {
		p := g.World.RandomFreePoint(g.Player.Pos, 6)
		if g.World.Get(floorI(p.X), floorI(p.Y)-1, floorI(p.Z)) != Grass {
			continue
		}
		g.Animals = append(g.Animals, NewAnimal(p, AnimalKind(rand.Intn(int(AnimalDino)))))
	}
}

// spawnTrader brings a wandering trader to the surface near the player.
func (g *Game) spawnTrader() {
	p := g.World.RandomFreePoint(g.Player.Pos, 12)
	t := NewAnimal(p, AnimalTrader)
	t.Walking = true
	g.Animals = append(g.Animals, t)
	g.announce("A wandering trader has arrived (right click to trade)", 4)
}

func (g *Game) traderAlive() bool {
	for _, a := range g.Animals {
		if a.Alive && a.Kind == AnimalTrader {
			return true
		}
	}
	return false
}

// spawnDinosaur puts one roaming dinosaur far from the player on open ground.
func (g *Game) spawnDinosaur() {
	for try := 0; try < 300; try++ {
		p := g.World.RandomFreePoint(g.Player.Pos, 25)
		ground := g.World.Get(floorI(p.X), floorI(p.Y)-1, floorI(p.Z))
		if (ground == Grass || ground == Sand || ground == Snow || try > 200) && g.World.PointFree(p, 0.8) {
			d := NewAnimal(p, AnimalDino)
			d.Walking = true
			g.Animals = append(g.Animals, d)
			return
		}
	}
}

func (g *Game) dinosaurs() int {
	n := 0
	for _, a := range g.Animals {
		if a.Alive && a.Kind == AnimalDino {
			n++
		}
	}
	return n
}

func (g *Game) updateAnimals(dt float32) {
	alive := 0
	keep := g.Animals[:0]
	for _, a := range g.Animals {
		if a.Alive && a.Kind == AnimalVillager {
			g.villagerTick(a, dt)
		}
		t := g.nearestTarget(a.Pos)
		if bite := a.Update(dt, g.World, t); bite > 0 {
			g.hurtTargetFrom(t.ID, int(float32(bite)*damageScale()+0.5), "was eaten by a Dinosaur", true, a.Pos, 7)
			g.Audio.Play(g.Audio.Hit, 0.8)
		}
		if a.Alive && a.Kind == AnimalDino {
			// Thudding steps and the occasional roar, by distance.
			d := WrapDist(a.Pos, g.Player.Pos)
			if a.RoarCD == 0 {
				a.RoarCD = 9 + rand.Float32()*12
				if vol := 0.9 * clamp(1-d/45, 0, 1); vol > 0.05 {
					g.Audio.Play(g.Audio.Roar, vol)
				}
			}
			if (a.Walking || a.Flee > 0) && int(a.Phase*2)%4 == 0 && d < 30 && a.StepFlag != int(a.Phase*2) {
				a.StepFlag = int(a.Phase * 2)
				g.Audio.Play(g.Audio.Steps[1], 0.5*clamp(1-d/30, 0, 1))
				if d < 12 {
					g.Shake = max(g.Shake, 0.12)
				}
			}
		}
		if a.Alive {
			alive++
		}
		if a.Alive || a.DeathT < 1 {
			keep = append(keep, a)
		}
	}
	g.Animals = keep
	// Herds slowly recover during the day (villagers do not count as herd animals).
	g.AnimalCD -= dt
	if g.AnimalCD <= 0 {
		g.AnimalCD = 25
		herd := 0
		for _, a := range g.Animals {
			if a.Alive && a.Kind < AnimalDino {
				herd++
			}
		}
		if herd < 10 && !g.Sky.IsNight() {
			g.spawnAnimals(2)
		}
		if g.dinosaurs() == 0 && rand.Float32() < 0.35 {
			g.spawnDinosaur()
		}
		// A trader visits roughly every other day and wanders off after a while.
		if !g.traderAlive() && !g.Sky.IsNight() && rand.Float32() < 0.12 {
			g.spawnTrader()
		}
		for _, a := range g.Animals {
			if a.Alive && a.Kind == AnimalTrader {
				a.Visit += 25
				if a.Visit > 400 {
					a.Alive = false
					a.DeathT = 0
					g.say("The trader wandered off", 2)
				}
			}
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
	case AnimalVillager:
		g.Score -= 200
		g.say("You killed a villager  -200", 2)
	case AnimalDino:
		g.Score += 800
		g.spawnDrop(c, Leather, 0)
		g.spawnDrop(c, Leather, 0)
		g.spawnDrop(c, Leather, 0)
		g.announce("DINOSAUR SLAIN  +800", 3)
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
