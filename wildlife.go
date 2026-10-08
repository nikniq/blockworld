package main

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Australian wildlife: an Outback biome of red sand and eucalyptus, with
// kangaroos, emus, wombats, koalas in the trees, platypuses in the rivers,
// crocodiles at the water's edge, and a kookaburra laughing at dawn.

// ---------- drawing ----------

// DrawKangaroo: upright body on big hind legs, a long tail and tall ears; it hops.
func (s *Skins) DrawKangaroo(w *World, p *Pose) {
	s.init(w)
	model := modelMatrix(p)
	c := lightColor(w, rl.Vector3Add(p.Pos, rl.NewVector3(0, 0.8, 0)), p.Alpha)
	k := SkinKangaroo
	crouch := float32(math.Abs(math.Sin(float64(p.Phase)))) * 0.25 * p.Amp
	legAng := -0.6 + crouch*1.2
	for _, sg := range []float32{-1, 1} {
		s.drawPart(k, PartLegL, rl.NewVector3(0.22, 0.55, 0.45), rl.NewVector3(sg*0.2, 0.5, -0.05), 0.5, legAng, 0, 0, model, c)
		s.drawPart(k, PartLegFL, rl.NewVector3(0.2, 0.12, 0.55), rl.NewVector3(sg*0.2, 0.06, 0.15), 0, 0, 0, 0, model, c) // feet
	}
	bodyY := 0.5 + crouch*0.1
	s.drawPart(k, PartBody, rl.NewVector3(0.5, 0.8, 0.45), rl.NewVector3(0, bodyY, 0), -0.5, 0.35-crouch*0.4, 0, 0, model, c)
	s.drawPart(k, PartExtra, rl.NewVector3(0.22, 0.2, 1.1), rl.NewVector3(0, bodyY+0.1, -0.3), 0, -0.45, 0, 0, model, c) // tail
	for _, sg := range []float32{-1, 1} {
		s.drawPart(k, PartArmL, rl.NewVector3(0.12, 0.35, 0.12), rl.NewVector3(sg*0.27, bodyY+0.6, 0.2), 0.5, -1.0, 0, 0, model, c)
	}
	headY := bodyY + 0.95
	s.drawPart(k, PartHead, rl.NewVector3(0.3, 0.32, 0.5), rl.NewVector3(0, headY, 0.2), 0, -p.Pitch, 0, 0, model, c)
	for _, sg := range []float32{-1, 1} {
		s.drawPart(k, PartLegBL, rl.NewVector3(0.08, 0.3, 0.12), rl.NewVector3(sg*0.11, headY+0.3, 0.1), 0, -0.2, 0, sg*0.3, model, c) // ears
	}
	if p.Flash > 0 {
		rl.DrawCubeV(rl.Vector3Add(p.Pos, rl.NewVector3(0, 0.9, 0)), rl.NewVector3(0.8, 1.9, 1.4), rl.Fade(rl.White, p.Flash*0.5))
	}
}

// DrawEmu: a round body on long legs with a long neck and a tuft of a tail.
func (s *Skins) DrawEmu(w *World, p *Pose) {
	s.init(w)
	model := modelMatrix(p)
	c := lightColor(w, rl.Vector3Add(p.Pos, rl.NewVector3(0, 0.8, 0)), p.Alpha)
	k := SkinEmu
	sw := swingAt(p.Phase, p.Amp) * 1.3
	for _, sg := range []float32{-1, 1} {
		s.drawPart(k, PartLegL, rl.NewVector3(0.1, 0.85, 0.1), rl.NewVector3(sg*0.16, 0.85, 0), 0.5, sw*sg, 0, 0, model, c)
	}
	s.drawPart(k, PartBody, rl.NewVector3(0.55, 0.5, 0.8), rl.NewVector3(0, 0.85, 0), -0.5, 0, 0, 0, model, c)
	s.drawPart(k, PartExtra, rl.NewVector3(0.4, 0.3, 0.35), rl.NewVector3(0, 1.3, -0.45), 0, 0.5, 0, 0, model, c)                // tail tuft
	s.drawPart(k, PartArmL, rl.NewVector3(0.16, 0.9, 0.16), rl.NewVector3(0, 1.3, 0.3), -0.5, -0.25+p.Pitch*0.5, 0, 0, model, c) // neck
	s.drawPart(k, PartHead, rl.NewVector3(0.22, 0.2, 0.42), rl.NewVector3(0, 2.1, 0.45), 0, -p.Pitch, 0, 0, model, c)
	if p.Flash > 0 {
		rl.DrawCubeV(rl.Vector3Add(p.Pos, rl.NewVector3(0, 1.1, 0)), rl.NewVector3(0.7, 2.3, 1.2), rl.Fade(rl.White, p.Flash*0.5))
	}
}

// DrawCroc: a long low body, a toothy snout and a swinging tail; legs splay to the sides.
func (s *Skins) DrawCroc(w *World, p *Pose) {
	s.init(w)
	model := modelMatrix(p)
	c := lightColor(w, rl.Vector3Add(p.Pos, rl.NewVector3(0, 0.3, 0)), p.Alpha)
	k := SkinCroc
	wag := float32(math.Sin(float64(p.Phase*0.8))) * 0.35 * max(p.Amp, 0.3)
	s.drawPart(k, PartBody, rl.NewVector3(0.65, 0.35, 1.5), rl.NewVector3(0, 0.2, 0), -0.5, 0, 0, 0, model, c)
	s.drawPart(k, PartHead, rl.NewVector3(0.45, 0.25, 0.8), rl.NewVector3(0, 0.25, 1.1), 0, p.Swing*0.4, 0, 0, model, c)
	s.drawPart(k, PartExtra, rl.NewVector3(0.4, 0.25, 0.9), rl.NewVector3(0, 0.3, -1.1), 0, 0, wag, 0, model, c)
	s.drawPart(k, PartExtra, rl.NewVector3(0.22, 0.16, 0.9), rl.NewVector3(0, 0.3, -1.95), 0, 0, wag*2, 0, model, c)
	for i, z := range []float32{0.5, -0.5} {
		for _, sg := range []float32{-1, 1} {
			lift := float32(math.Sin(float64(p.Phase+float32(i)*1.5))) * 0.3 * p.Amp * sg
			s.drawPart(k, PartLegFL, rl.NewVector3(0.45, 0.14, 0.16), rl.NewVector3(sg*0.35, 0.18, z), 0, lift, 0, sg*0.6, model, c)
		}
	}
	if p.Flash > 0 {
		rl.DrawCubeV(rl.Vector3Add(p.Pos, rl.NewVector3(0, 0.3, 0)), rl.NewVector3(1.2, 0.7, 4.0), rl.Fade(rl.White, p.Flash*0.5))
	}
}

// DrawPlatypus: a flat little body with a bill and a paddle tail.
func (s *Skins) DrawPlatypus(w *World, p *Pose) {
	s.init(w)
	model := modelMatrix(p)
	c := lightColor(w, rl.Vector3Add(p.Pos, rl.NewVector3(0, 0.2, 0)), p.Alpha)
	k := SkinPlatypus
	sw := swingAt(p.Phase, p.Amp)
	s.drawPart(k, PartBody, rl.NewVector3(0.35, 0.2, 0.6), rl.NewVector3(0, 0.08, 0), -0.5, 0, 0, 0, model, c)
	s.drawPart(k, PartHead, rl.NewVector3(0.3, 0.07, 0.32), rl.NewVector3(0, 0.13, 0.45), 0, 0, 0, 0, model, c)       // bill
	s.drawPart(k, PartExtra, rl.NewVector3(0.3, 0.07, 0.4), rl.NewVector3(0, 0.12, -0.48), 0, sw*0.3, 0, 0, model, c) // tail
	for _, d := range [][2]float32{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}} {
		s.drawPart(k, PartLegFL, rl.NewVector3(0.14, 0.06, 0.14), rl.NewVector3(d[0]*0.22, 0.05, d[1]*0.2), 0, sw*d[0]*d[1], 0, 0, model, c)
	}
}

// ---------- wildlife behaviour ----------

// wildlifeTick adds species quirks on top of the base wandering.
func (g *Game) wildlifeTick(a *Animal, dt float32) {
	w := g.World
	switch a.Kind {
	case AnimalKangaroo:
		// Hop: bounce while moving.
		if (a.Walking || a.Flee > 0) && a.VelY == 0 && !w.WaterAt(a.Pos) {
			if w.Solid(floorI(a.Pos.X), floorI(a.Pos.Y)-1, floorI(a.Pos.Z)) {
				a.VelY = 4.2
			}
		}
	case AnimalKoala:
		// Mostly sits; a tamed-wolf style follow is not needed.
		if a.Walking && rand.Float32() < 0.02 {
			a.Walking = false
			a.WanderT = 3 + rand.Float32()*4
		}
	case AnimalPlatypus:
		// Stays in the water; heads back if it strands.
		if !w.WaterAt(a.Pos) && !w.WaterAt(rl.NewVector3(a.Pos.X, a.Pos.Y-0.5, a.Pos.Z)) {
			a.WanderT = 0
			if a.Home.Y != 0 {
				to := WrapDelta(a.Home, a.Pos)
				to.Y = 0
				if l := rl.Vector3Length(to); l > 0.1 {
					a.Heading = rl.Vector3Scale(to, 1/l)
					a.Walking = true
				}
			}
		}
	case AnimalCrocodile:
		// Lurks by the water and attacks anyone who comes close.
		t := g.nearestTarget(a.Pos)
		d := WrapDist(t.Pos, a.Pos)
		if d < 6 && a.Flee == 0 && t.ID&villagerIDBit == 0 {
			a.Flee = 4 // charge (Spec.Damage > 0 turns Flee into aggression)
		}
		if a.Flee == 0 && a.Home.Y != 0 && WrapDist(a.Pos, a.Home) > 10 {
			to := WrapDelta(a.Home, a.Pos)
			to.Y = 0
			a.Heading = rl.Vector3Normalize(to)
			a.Walking = true
			a.WanderT = 1
		}
	}
}

// spawnWildlife places outback and river animals according to the biome.
func (g *Game) spawnWildlife(n int) {
	w := g.World
	for i := 0; i < n*6 && n > 0; i++ {
		p := w.RandomFreePoint(g.Player.Pos, 8)
		lx, lz := wrapX(floorI(p.X)-originX), wrapZ(floorI(p.Z)-originZ)
		biome := w.Biome[lz*worldW+lx]
		ground := w.Get(floorI(p.X), floorI(p.Y)-1, floorI(p.Z))
		var kind AnimalKind
		switch {
		case biome == BiomeOutback && (ground == RedSand || ground == Grass):
			kind = []AnimalKind{AnimalKangaroo, AnimalKangaroo, AnimalEmu, AnimalWombat}[rand.Intn(4)]
		case (biome == BiomeForest || biome == BiomeOutback) && ground == Grass && rand.Float32() < 0.3:
			kind = AnimalKoala
		default:
			continue
		}
		a := NewAnimal(p, kind)
		a.Walking = kind != AnimalKoala
		if kind == AnimalKoala {
			// Up a tree if there is one nearby.
			for dx := -4; dx <= 4; dx++ {
				for dz := -4; dz <= 4; dz++ {
					x, z := floorI(p.X)+dx, floorI(p.Z)+dz
					top := w.SurfaceY(x, z)
					if b := w.Get(x, top-1, z); b == Log || b == EucLog || b == Leaves || b == EucLeaves {
						a.Pos = rl.NewVector3(float32(x)+0.5, float32(top), float32(z)+0.5)
					}
				}
			}
		}
		g.Animals = append(g.Animals, a)
		n--
	}
}

// spawnWaterLife puts platypuses in rivers and lakes and crocodiles on warm shores.
func (g *Game) spawnWaterLife(n int) {
	w := g.World
	for i := 0; i < n*40 && n > 0; i++ {
		lx, lz := rand.Intn(worldW), rand.Intn(worldD)
		h := w.Height[lz*worldW+lx]
		if h <= w.Ground[lz*worldW+lx] || h > seaLevel+1 {
			continue // not a water column
		}
		pos := rl.NewVector3(float32(lx+originX)+0.5, float32(h)-0.3, float32(lz+originZ)+0.5)
		if WrapDist(pos, g.Player.Pos) < 10 {
			continue
		}
		biome := w.Biome[lz*worldW+lx]
		kind := AnimalPlatypus
		if (biome == BiomeOutback || biome == BiomePlains) && rand.Float32() < 0.4 {
			kind = AnimalCrocodile
			// Crocodiles lie on the bank next to the water.
			for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				x, z := lx+d[0], lz+d[1]
				if g2 := w.Ground[wrapZ(z)*worldW+wrapX(x)]; g2 > seaLevel && g2 <= seaLevel+2 {
					pos = rl.NewVector3(float32(x+originX)+0.5, float32(g2), float32(z+originZ)+0.5)
					break
				}
			}
		}
		a := NewAnimal(pos, kind)
		a.Home = pos
		a.Walking = true
		g.Animals = append(g.Animals, a)
		n--
	}
}

// kookaburra laughs once at dawn when forest or outback is near.
func (g *Game) kookaburra() {
	w := g.World
	lx, lz := wrapX(floorI(g.Player.Pos.X)-originX), wrapZ(floorI(g.Player.Pos.Z)-originZ)
	if b := w.Biome[lz*worldW+lx]; b == BiomeForest || b == BiomeOutback || b == BiomePlains {
		g.Audio.Play(g.Audio.Laugh, 0.55)
		g.say("A kookaburra laughs at the dawn", 2)
	}
}
