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
	AnimalWolf
	AnimalCat
	AnimalHorse
	AnimalBoat
	AnimalKangaroo
	AnimalEmu
	AnimalKoala
	AnimalWombat
	AnimalPlatypus
	AnimalCrocodile
	AnimalBronto
	AnimalRaptor
	AnimalCompy
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
	AnimalPig:       {"Pig", 4, 1.6, 0.35, 0.9, rl.NewColor(235, 160, 170, 255), rl.NewColor(240, 170, 180, 255), rl.NewColor(220, 140, 150, 255), 2, 0},
	AnimalCow:       {"Cow", 6, 1.4, 0.4, 1.3, rl.NewColor(80, 55, 40, 255), rl.NewColor(90, 65, 50, 255), rl.NewColor(60, 40, 30, 255), 3, 0},
	AnimalSheep:     {"Sheep", 4, 1.5, 0.4, 1.1, rl.NewColor(230, 230, 225, 255), rl.NewColor(70, 60, 55, 255), rl.NewColor(60, 55, 50, 255), 1, 0},
	AnimalDino:      {"Dinosaur", 45, 2.2, 0.7, 2.8, rl.NewColor(70, 120, 60, 255), rl.NewColor(80, 130, 65, 255), rl.NewColor(60, 100, 50, 255), 8, 18},
	AnimalTrader:    {"Wandering Trader", 20, 1.2, 0.3, 1.8, rl.NewColor(90, 60, 130, 255), rl.NewColor(205, 160, 120, 255), rl.NewColor(60, 40, 90, 255), 0, 0},
	AnimalVillager:  {"Villager", 20, 1.3, 0.3, 1.8, rl.NewColor(150, 110, 70, 255), rl.NewColor(205, 160, 120, 255), rl.NewColor(80, 60, 40, 255), 0, 0},
	AnimalWolf:      {"Wolf", 16, 2.6, 0.35, 0.85, rl.NewColor(200, 200, 200, 255), rl.NewColor(210, 210, 210, 255), rl.NewColor(170, 170, 170, 255), 0, 6},
	AnimalKangaroo:  {"Kangaroo", 14, 3.4, 0.4, 1.9, rl.NewColor(170, 120, 80, 255), rl.NewColor(175, 125, 85, 255), rl.NewColor(150, 105, 70, 255), 3, 8},
	AnimalEmu:       {"Emu", 10, 3.8, 0.35, 2.3, rl.NewColor(90, 80, 70, 255), rl.NewColor(100, 110, 140, 255), rl.NewColor(110, 100, 90, 255), 2, 0},
	AnimalKoala:     {"Koala", 6, 0.8, 0.3, 0.75, rl.NewColor(140, 140, 145, 255), rl.NewColor(150, 150, 155, 255), rl.NewColor(120, 120, 125, 255), 0, 0},
	AnimalWombat:    {"Wombat", 12, 1.3, 0.4, 0.7, rl.NewColor(110, 85, 65, 255), rl.NewColor(115, 90, 70, 255), rl.NewColor(90, 70, 55, 255), 0, 0},
	AnimalPlatypus:  {"Platypus", 5, 1.6, 0.25, 0.3, rl.NewColor(110, 75, 50, 255), rl.NewColor(230, 180, 100, 255), rl.NewColor(100, 70, 45, 255), 0, 0},
	AnimalCrocodile: {"Crocodile", 30, 1.5, 0.5, 0.5, rl.NewColor(80, 100, 60, 255), rl.NewColor(90, 110, 65, 255), rl.NewColor(70, 90, 55, 255), 2, 12},
	AnimalBronto:    {"Brontosaurus", 120, 1.1, 1.3, 4.5, rl.NewColor(95, 125, 90, 255), rl.NewColor(105, 135, 95, 255), rl.NewColor(80, 105, 75, 255), 14, 0},
	AnimalRaptor:    {"Raptor", 12, 4.6, 0.3, 1.1, rl.NewColor(150, 110, 60, 255), rl.NewColor(160, 120, 65, 255), rl.NewColor(120, 90, 50, 255), 2, 7},
	AnimalCompy:     {"Compy", 3, 3.5, 0.18, 0.45, rl.NewColor(120, 150, 70, 255), rl.NewColor(130, 160, 75, 255), rl.NewColor(100, 130, 60, 255), 1, 0},
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
	Home          rl.Vector3
	Prof          Profession
	Name          string
	Quest         int
	QuestDone     bool
	QuestKills    int // kills when a kill quest was accepted (-1: not yet)
	QuestAccepted bool
	TalkCount     int
	Bubble        string  // ambient line shown over the head
	BubbleT       float32 // seconds the bubble stays
	BubbleCD      float32
	// Wolves.
	Tamed   bool
	AtkCD   float32
	Browse  float32 // brontosaur: seconds left chewing with the neck raised
	Sitting bool    // cats: told to stay
	Coat    int     // cats: colour variant
	Phase   float32
	Alive   bool
	DeathT  float32
	Lum     float32
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
	case AnimalWolf:
		kind, scale = SkinWolf, 0.8
	case AnimalCat:
		kind, scale = SkinCat+SkinKind(a.Coat), 0.55
	case AnimalHorse:
		kind, scale = SkinHorse+SkinKind(a.Coat), 1.25
	case AnimalBoat:
		a.drawBoat()
		return
	case AnimalKoala:
		kind, scale = SkinKoala, 0.55
	case AnimalWombat:
		kind, scale = SkinWombat, 0.8
	case AnimalKangaroo, AnimalEmu, AnimalCrocodile, AnimalPlatypus:
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
		if a.Flee > 4.7 || (a.Spec.Damage > 0 && a.Flee > 13.7) {
			pose.Flash = 0.6
		}
		switch a.Kind {
		case AnimalKangaroo:
			skins.DrawKangaroo(w, &pose)
		case AnimalEmu:
			skins.DrawEmu(w, &pose)
		case AnimalCrocodile:
			pose.Swing = clamp(a.BiteCD-0.9, 0, 0.4) / 0.4
			skins.DrawCroc(w, &pose)
		default:
			skins.DrawPlatypus(w, &pose)
		}
		if a.Alive && a.HP < a.Spec.HP {
			top := a.Pos.Y + a.Spec.Height + 0.2
			frac := float32(a.HP) / float32(a.Spec.HP)
			rl.DrawCubeV(rl.NewVector3(a.Pos.X, top, a.Pos.Z), rl.NewVector3(1.0, 0.08, 0.08), rl.NewColor(0, 0, 0, 180))
			rl.DrawCubeV(rl.NewVector3(a.Pos.X-(1-frac)*0.5, top, a.Pos.Z), rl.NewVector3(frac, 0.1, 0.1), rl.Lime)
		}
		return
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
	case AnimalDino, AnimalBronto, AnimalRaptor, AnimalCompy:
		pose := Pose{Pos: a.Pos, Yaw: yawOf(a.Heading), Phase: a.Phase, Amp: 0, Scale: 1, Lum: a.Lum, Alpha: 1}
		if a.Walking || a.Flee > 0 {
			pose.Amp = 1
		}
		if a.Kind == AnimalBronto {
			pose.Pitch = -a.Browse * 0.4 // neck up while eating
			if a.Browse > 0 {
				pose.Swing = clamp(float32(math.Sin(float64(rl.GetTime()*6))), 0, 1)
			}
		}
		if a.Kind == AnimalCompy {
			pose.Scale = 0.4
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
		switch a.Kind {
		case AnimalBronto:
			skins.DrawBronto(w, &pose)
		case AnimalRaptor, AnimalCompy:
			pose.Swing = clamp(a.BiteCD-0.9, 0, 0.4) / 0.4
			skins.DrawRaptor(w, &pose)
		default:
			pose.Swing = clamp(a.BiteCD-0.9, 0, 0.4) / 0.4 // jaw snap just after a bite
			skins.DrawDino(w, &pose)
		}
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
	if a.Kind == AnimalCat && a.Sitting {
		pose.Amp = 0
		pose.Pos.Y -= 0.12 // hunkered down
	}
	skins.DrawQuadruped(w, kind, &pose)
	if a.Kind == AnimalCat && a.Alive {
		// Tail up, and a collar when tamed.
		fwd := a.Heading
		if rl.Vector3Length(fwd) < 0.01 {
			fwd = rl.NewVector3(0, 0, 1)
		}
		tailBase := rl.Vector3Add(rl.NewVector3(a.Pos.X, pose.Pos.Y+0.45*scale, a.Pos.Z), rl.Vector3Scale(fwd, -0.5*scale))
		sway := float32(math.Sin(float64(rl.GetTime()*3+float64(a.Phase)))) * 0.08
		tailTip := rl.Vector3Add(tailBase, rl.NewVector3(sway, 0.35*scale, -0.15*scale*fwd.Z))
		rl.DrawCylinderEx(tailBase, tailTip, 0.05, 0.03, 4, mul(a.Spec.Legs, a.Lum))
		if a.Tamed {
			c := rl.Vector3Add(rl.NewVector3(a.Pos.X, pose.Pos.Y+0.42*scale, a.Pos.Z), rl.Vector3Scale(fwd, 0.4*scale))
			rl.DrawCubeV(c, rl.NewVector3(0.34*scale, 0.08, 0.34*scale), mul(rl.NewColor(60, 120, 220, 255), a.Lum))
		}
	}
	if a.Kind == AnimalWolf && a.Tamed && a.Alive {
		// Red collar.
		fwd := a.Heading
		if rl.Vector3Length(fwd) < 0.01 {
			fwd = rl.NewVector3(0, 0, 1)
		}
		c := rl.Vector3Add(rl.NewVector3(a.Pos.X, a.Pos.Y+0.62*scale, a.Pos.Z), rl.Vector3Scale(fwd, 0.42*scale))
		rl.DrawCubeV(c, rl.NewVector3(0.42*scale, 0.1, 0.42*scale), mul(rl.NewColor(220, 40, 40, 255), a.Lum))
	}
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

// spawnWolves scatters a few wild wolves in forests and taiga.
func (g *Game) spawnWolves(n int) {
	for i := 0; i < n*4 && n > 0; i++ {
		p := g.World.RandomFreePoint(g.Player.Pos, 20)
		lx, lz := wrapX(floorI(p.X)-originX), wrapZ(floorI(p.Z)-originZ)
		if b := g.World.Biome[lz*worldW+lx]; b != BiomeForest && b != BiomeTaiga {
			continue
		}
		wf := NewAnimal(p, AnimalWolf)
		wf.Walking = true
		g.Animals = append(g.Animals, wf)
		n--
	}
}

// drawBoat: a wooden hull that bobs on the water.
func (a *Animal) drawBoat() {
	fwd := a.Heading
	if rl.Vector3Length(fwd) < 0.01 {
		fwd = rl.NewVector3(0, 0, 1)
	}
	side := rl.NewVector3(-fwd.Z, 0, fwd.X)
	bob := float32(math.Sin(float64(rl.GetTime()*1.5+float64(a.ID)))) * 0.04
	y := a.Pos.Y + 0.2 + bob
	wood := mul(rl.NewColor(150, 110, 65, 255), a.Lum)
	dark := mul(rl.NewColor(100, 70, 40, 255), a.Lum)
	c := rl.NewVector3(a.Pos.X, y, a.Pos.Z)
	rl.DrawCubeV(c, rl.NewVector3(1.2, 0.18, 2.0), dark) // floor
	for _, sg := range []float32{-1, 1} {
		rl.DrawCubeV(rl.Vector3Add(c, rl.Vector3Add(rl.Vector3Scale(side, sg*0.55), rl.NewVector3(0, 0.22, 0))), rl.NewVector3(0.12, 0.45, 2.0), wood)
	}
	for _, sg := range []float32{-1, 1} {
		rl.DrawCubeV(rl.Vector3Add(c, rl.Vector3Add(rl.Vector3Scale(fwd, sg*1.0), rl.NewVector3(0, 0.25, 0))), rl.NewVector3(1.2, 0.5, 0.14), wood)
	}
	if a.HP < a.Spec.HP {
		rl.DrawCubeV(rl.NewVector3(a.Pos.X, y+1, a.Pos.Z), rl.NewVector3(float32(a.HP)/float32(a.Spec.HP)*1.2, 0.08, 0.08), rl.Lime)
	}
}

// spawnHorses scatters horses on the plains.
func (g *Game) spawnHorses(n int) {
	w := g.World
	for i := 0; i < n*6 && n > 0; i++ {
		p := w.RandomFreePoint(g.Player.Pos, 10)
		lx, lz := wrapX(floorI(p.X)-originX), wrapZ(floorI(p.Z)-originZ)
		if w.Biome[lz*worldW+lx] != BiomePlains || !w.PointFree(p, 0.6) {
			continue
		}
		h := NewAnimal(p, AnimalHorse)
		h.Coat = rand.Intn(3)
		h.Walking = true
		g.Animals = append(g.Animals, h)
		n--
	}
}

// tameHorse: two helpings of wheat or apples make a horse rideable.
func (g *Game) tameHorse(a *Animal) {
	p := g.Player
	if a.Tamed {
		g.mount(a)
		return
	}
	if !(p.Held.Kind == ItemFood && (p.Held.Block == Apple || p.Held.Block == Bread)) && !(p.Held.Kind == ItemBlock && p.Held.Block == WheatItem) && p.Held.Block != WheatItem {
		g.say("The horse snorts. Offer it wheat, bread or an apple.", 1.8)
		return
	}
	p.Inv[p.Held.Block]--
	p.EnsureHeld()
	a.Flee = 0
	g.burst(rl.Vector3Add(a.Pos, rl.NewVector3(0, 1.2, 0)), rl.NewColor(255, 120, 150, 255), 8)
	g.Audio.Play(g.Audio.Eat, 0.6)
	a.Visit++
	if a.Visit >= 2 {
		a.Tamed = true
		g.say("The horse is yours. Right click to ride; SHIFT to dismount.", 3)
		g.Audio.Play(g.Audio.Clear, 0.7)
		g.unlock(AchHorse)
	} else {
		g.say("The horse nuzzles your hand. One more should do it.", 1.8)
	}
}

// mount puts the player on a horse or boat.
func (g *Game) mount(a *Animal) {
	if g.isClient() {
		g.say("Only the host can ride (for now)", 1.5)
		return
	}
	g.Mount = a
	g.Player.Mounted = true
	g.Player.Flying = false
	a.Walking = false
	a.WanderT = 99
	g.say("SHIFT dismounts", 1.2)
}

func (g *Game) dismount() {
	if g.Mount == nil {
		return
	}
	a := g.Mount
	g.Mount = nil
	g.Player.Mounted = false
	side := rl.NewVector3(-a.Heading.Z, 0, a.Heading.X)
	g.Player.Pos = rl.Vector3Add(a.Pos, rl.Vector3Scale(side, 1.2))
	g.Player.Pos.Y = a.Pos.Y + 0.2
	if g.World.WaterAt(g.Player.Pos) {
		g.Player.Pos.Y += 0.5
	}
	a.WanderT = 1
}

// tickMount drives the mount from the player's steering and seats the player on it.
func (g *Game) tickMount(dt float32) {
	a := g.Mount
	if a == nil {
		return
	}
	p := g.Player
	if !a.Alive || rl.IsKeyPressed(rl.KeyLeftShift) || rl.IsKeyPressed(rl.KeyRightShift) {
		g.dismount()
		return
	}
	w := g.World
	move := p.MountMove
	speed := float32(0)
	inWater := w.WaterAt(rl.NewVector3(a.Pos.X, a.Pos.Y+0.2, a.Pos.Z))
	switch a.Kind {
	case AnimalHorse:
		speed = 9
		if p.Sprinting {
			speed = 13
		}
		if inWater {
			speed = 2.5
		}
		if p.MountJump && a.VelY == 0 && w.Solid(floorI(a.Pos.X), floorI(a.Pos.Y)-1, floorI(a.Pos.Z)) {
			a.VelY = 8
		}
	case AnimalBoat:
		speed = 8
		if !inWater && !w.WaterAt(a.Pos) {
			speed = 1.5 // dragging on land
		}
	}
	if move.X != 0 || move.Z != 0 {
		a.Heading = rl.Vector3Normalize(rl.NewVector3(move.X, 0, move.Z))
		a.Phase += dt * speed * 1.4
	}
	var delta rl.Vector3
	delta.X, delta.Z = move.X*speed*dt, move.Z*speed*dt
	if inWater {
		a.VelY = max(a.VelY-gravity*0.3*dt, -1.5)
		if w.WaterAt(rl.NewVector3(a.Pos.X, a.Pos.Y+a.Spec.Height*0.6, a.Pos.Z)) || a.Kind == AnimalBoat {
			a.VelY = min(a.VelY+16*dt, 2)
		}
	} else {
		a.VelY = max(a.VelY-gravity*dt, -30)
	}
	delta.Y = a.VelY * dt
	var res MoveResult
	a.Pos, res = w.MoveBox(a.Pos, a.Spec.Radius, a.Spec.Height, delta, a.Kind == AnimalHorse)
	a.Pos = WrapPos(a.Pos)
	if res.Ground || res.Ceiling {
		a.VelY = 0
	}
	seat := float32(1.35)
	if a.Kind == AnimalBoat {
		seat = 0.3
	}
	p.Pos = rl.NewVector3(a.Pos.X, a.Pos.Y+seat, a.Pos.Z)
	p.InWater, p.HeadWater = false, false
	if a.Kind == AnimalHorse && (move.X != 0 || move.Z != 0) && a.VelY == 0 {
		p.BobPhase += dt * speed * 0.8
		p.BobAmount = lerp(p.BobAmount, 0.6, dt*8)
		if int(a.Phase*2)%3 == 0 && a.StepFlag != int(a.Phase*2) {
			a.StepFlag = int(a.Phase * 2)
			g.Audio.Play(g.Audio.Steps[1], 0.35)
		}
	}
}

// placeBoat drops a boat onto water in front of the player.
func (g *Game) placeBoat() {
	p := g.Player
	o, d := p.Eye(), p.Forward()
	for t := float32(0.5); t < 7; t += 0.25 {
		q := rl.Vector3Add(o, rl.Vector3Scale(d, t))
		if g.World.WaterAt(q) {
			q.Y = float32(floorI(q.Y)) + 0.85
			b := NewAnimal(q, AnimalBoat)
			b.Heading = p.FlatForward()
			g.Animals = append(g.Animals, b)
			p.Inv[Boat]--
			p.EnsureHeld()
			g.Audio.Play(g.Audio.Splash, 0.6)
			return
		}
		if g.World.Solid(floorI(q.X), floorI(q.Y), floorI(q.Z)) {
			break
		}
	}
	g.say("Aim at water to launch the boat", 1.2)
}

// spawnCats puts stray cats near villages and in forests.
func (g *Game) spawnCats(n int) {
	w := g.World
	for i := 0; i < n*6 && n > 0; i++ {
		p := w.RandomFreePoint(g.Player.Pos, 10)
		lx, lz := wrapX(floorI(p.X)-originX), wrapZ(floorI(p.Z)-originZ)
		nearVillage := false
		for _, a := range g.Animals {
			if a.Kind == AnimalVillager && WrapDist(a.Home, p) < 20 {
				nearVillage = true
				break
			}
		}
		if !nearVillage && w.Biome[lz*worldW+lx] != BiomeForest {
			continue
		}
		c := NewAnimal(p, AnimalCat)
		c.Coat = rand.Intn(3)
		c.Walking = true
		g.Animals = append(g.Animals, c)
		n--
	}
}

// catTick: pets follow (unless told to sit), keep creepers away, and purr by the fire.
func (g *Game) catTick(a *Animal, dt float32) {
	if !a.Tamed {
		// Strays are shy: back off from a player who comes too close without fish.
		if WrapDist(g.Player.Pos, a.Pos) < 2.5 && a.Flee == 0 {
			if h := g.Player.Held; !(h.Kind == ItemFood && h.Block == Fish) {
				a.Flee = 1.5
			}
		}
		return
	}
	a.Flee = 0
	// Creepers will not come near a cat.
	for _, e := range g.Enemies {
		if e.Alive && e.Kind == KindCreeper && WrapDist(e.Pos, a.Pos) < 6 {
			away := WrapDelta(e.Pos, a.Pos)
			away.Y = 0
			if l := rl.Vector3Length(away); l > 0.01 {
				e.Heading = rl.Vector3Scale(away, 1/l)
				e.Fuse = 0
				e.Pos, _ = g.World.MoveBox(e.Pos, e.Spec.Radius, e.Spec.Height, rl.Vector3Scale(e.Heading, 3*dt), true)
			}
		}
	}
	if a.Sitting {
		a.Walking = false
		a.WanderT = 1
		return
	}
	p := g.Player
	d := WrapDelta(p.Pos, a.Pos)
	d.Y = 0
	dist := rl.Vector3Length(d)
	switch {
	case dist > 24:
		a.Pos = rl.Vector3Add(p.Pos, rl.NewVector3(-1.2, 0.2, 1.2))
	case dist > 3.5:
		a.Heading = rl.Vector3Scale(d, 1/dist)
		a.Walking = true
		a.WanderT = 0.3
	case dist < 2:
		a.Walking = false
		a.WanderT = 0.6
	}
}

// tameCat: a fish wins a stray over; right click again to make it sit or follow.
func (g *Game) tameCat(a *Animal) {
	p := g.Player
	if a.Tamed {
		if p.Held.Kind == ItemFood && p.Held.Block == Fish && a.HP < a.Spec.HP {
			p.Inv[Fish]--
			a.HP = a.Spec.HP
			p.EnsureHeld()
			g.say("Your cat purrs", 1.2)
			g.Audio.Play(g.Audio.Purr, 0.6)
			return
		}
		a.Sitting = !a.Sitting
		if a.Sitting {
			g.say("Your cat sits and waits here", 1.2)
		} else {
			g.say("Your cat follows you", 1.2)
		}
		g.Audio.Play(g.Audio.Purr, 0.5)
		return
	}
	if !(p.Held.Kind == ItemFood && p.Held.Block == Fish) {
		g.say("The cat watches you. It would like a fish.", 1.8)
		return
	}
	p.Inv[Fish]--
	p.EnsureHeld()
	a.Flee = 0
	a.Tamed = true
	g.burst(rl.Vector3Add(a.Pos, rl.NewVector3(0, 0.6, 0)), rl.NewColor(255, 120, 150, 255), 10)
	g.Audio.Play(g.Audio.Purr, 0.8)
	g.say("The cat is yours. It will follow you, and creepers keep their distance from it.", 3.5)
	g.unlock(AchCat)
}

// wolfTick: tamed wolves follow their owner and attack nearby hostiles; wild
// wolves wander and bite back when hurt (the base Update handles that).
func (g *Game) wolfTick(a *Animal, dt float32) {
	a.AtkCD = max(0, a.AtkCD-dt)
	if !a.Tamed {
		return
	}
	p := g.Player
	a.Flee = 0 // never turns on its owner
	// Attack hostiles near the owner.
	var threat *Enemy
	td := float32(99)
	for _, e := range g.Enemies {
		if e.Alive && e.Kind != KindCreeper {
			if d := WrapDist(e.Pos, a.Pos); d < td && WrapDist(e.Pos, p.Pos) < 12 {
				threat, td = e, d
			}
		}
	}
	if threat != nil && td < 10 {
		a.Heading = rl.Vector3Normalize(WrapDelta(threat.Pos, a.Pos))
		a.Heading.Y = 0
		a.Walking = true
		a.WanderT = 0.3
		if td < 1.6 && a.AtkCD == 0 {
			a.AtkCD = 0.9
			g.burst(rl.Vector3Add(threat.Pos, rl.NewVector3(0, 0.8, 0)), rl.NewColor(255, 90, 90, 255), 5)
			if threat.Hit(3) {
				g.killEnemy(threat, threat.Spec.Points/2)
			}
		}
		return
	}
	// Follow the owner: close the gap when far, teleport if very far.
	d := WrapDelta(p.Pos, a.Pos)
	d.Y = 0
	dist := rl.Vector3Length(d)
	switch {
	case dist > 24:
		a.Pos = rl.Vector3Add(p.Pos, rl.NewVector3(1.5, 0.2, 1.5))
	case dist > 4:
		a.Heading = rl.Vector3Scale(d, 1/dist)
		a.Walking = true
		a.WanderT = 0.3
	case dist < 2.5:
		a.Walking = false
		a.WanderT = 0.8
	}
}

// tameWolf feeds a wolf: a few pieces of meat make it loyal.
func (g *Game) tameWolf(a *Animal) {
	p := g.Player
	if p.Held.Kind != ItemFood || (p.Held.Block != Meat && p.Held.Block != CookedMeat && p.Held.Block != Fish) {
		g.say("The wolf eyes you. Offer it meat or fish.", 1.8)
		return
	}
	p.Inv[p.Held.Block]--
	p.EnsureHeld()
	a.Flee = 0
	a.HP = min(a.Spec.HP, a.HP+6)
	g.burst(rl.Vector3Add(a.Pos, rl.NewVector3(0, 0.8, 0)), rl.NewColor(255, 120, 150, 255), 8)
	g.Audio.Play(g.Audio.Eat, 0.6)
	if a.Tamed {
		g.say("Your wolf is well fed", 1.2)
		return
	}
	a.Visit++
	if a.Visit >= 2 {
		a.Tamed = true
		g.say("The wolf is yours. It will follow you and fight for you.", 3)
		g.Audio.Play(g.Audio.Clear, 0.7)
		g.unlock(AchWolf)
	} else {
		g.say("The wolf wags its tail. One more should do it.", 1.8)
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
		if a.Alive && a.Kind == AnimalWolf {
			g.wolfTick(a, dt)
		}
		if a.Alive && a.Kind == AnimalCat {
			g.catTick(a, dt)
		}
		if a.Alive && a.Kind >= AnimalKangaroo && a.Kind <= AnimalCrocodile {
			g.wildlifeTick(a, dt)
		}
		if a.Alive && a.Kind >= AnimalBronto {
			g.dinoTick(a, dt)
		}
		if a == g.Mount {
			alive++
			keep = append(keep, a)
			continue
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
		wild := 0
		for _, a := range g.Animals {
			if a.Alive && a.Kind >= AnimalKangaroo && a.Kind <= AnimalCrocodile {
				wild++
			}
		}
		if wild < 12 && rand.Float32() < 0.4 {
			g.spawnWildlife(2)
			g.spawnWaterLife(1)
		}
		if g.dinoCount() < 8 && rand.Float32() < 0.25 {
			g.spawnDinos(1)
		}
		horses := 0
		cats := 0
		for _, a := range g.Animals {
			if a.Alive && a.Kind == AnimalCat && !a.Tamed {
				cats++
			}
			if a.Alive && a.Kind == AnimalHorse {
				horses++
			}
		}
		if horses < 4 && rand.Float32() < 0.3 {
			g.spawnHorses(1)
		}
		if cats < 3 && rand.Float32() < 0.3 {
			g.spawnCats(1)
		}
		wolves := 0
		for _, a := range g.Animals {
			if a.Alive && a.Kind == AnimalWolf && !a.Tamed {
				wolves++
			}
		}
		if wolves < 3 && rand.Float32() < 0.3 {
			g.spawnWolves(1)
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
	case AnimalKoala, AnimalPlatypus:
		g.Score -= 110 // undo the kill bonus and fine the player
		g.say("Protected species!  -100", 2)
	case AnimalKangaroo:
		g.spawnDrop(c, Leather, 0)
	case AnimalWombat:
		g.spawnDrop(c, Leather, 0)
		g.spawnDrop(c, Leather, 0)
	case AnimalCrocodile:
		g.Score += 150
		for i := 0; i < 3; i++ {
			g.spawnDrop(c, Leather, 0)
		}
	case AnimalWolf:
		if a.Tamed {
			g.say("Your wolf has fallen", 2.5)
		}
	case AnimalCat:
		g.Score -= 50
		if a.Tamed {
			g.say("Your cat has died", 2.5)
		}
	case AnimalBoat:
		g.spawnDrop(c, Boat, 0)
		if g.Mount == a {
			g.dismount()
		}
	case AnimalHorse:
		g.spawnDrop(c, Leather, 0)
		if g.Mount == a {
			g.dismount()
		}
	case AnimalVillager:
		g.Score -= 200
		g.VillagerGrudge = 240 // seconds the village stays angry with you
		g.VillagersLost++
		g.say("You killed a villager  -200. The village will remember.", 3)
	case AnimalDino:
		g.Score += 800
		g.spawnDrop(c, Leather, 0)
		g.spawnDrop(c, Leather, 0)
		g.spawnDrop(c, Leather, 0)
		g.announce("TYRANNOSAUR SLAIN  +800", 3)
	case AnimalBronto:
		g.Score += 400
		for i := 0; i < 4; i++ {
			g.spawnDrop(c, Leather, 0)
		}
		g.announce("Brontosaurus slain  +400", 3)
	case AnimalRaptor:
		g.Score += 120
		g.spawnDrop(c, Leather, 0)
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
