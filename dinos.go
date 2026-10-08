package main

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Dinosaurs of several sizes. The original big biped is the tyrannosaur.
// Brontosaurs browse the swamps, stretching their necks up to eat leaves;
// raptors hunt in packs on the outback; a few tiny compys scurry about.

// DrawBronto: a huge four-legged body, a very long neck that reaches up or
// down (p.Pitch tilts it) and a long tapering tail.
func (s *Skins) DrawBronto(w *World, p *Pose) {
	s.init(w)
	model := modelMatrix(p)
	c := lightColor(w, rl.Vector3Add(p.Pos, rl.NewVector3(0, 2, 0)), p.Alpha)
	k := SkinBronto
	sw := swingAt(p.Phase, p.Amp) * 0.6
	const legH float32 = 1.6
	legs := [][3]float32{{0.6, 1.1, 1}, {-0.6, 1.1, -1}, {0.6, -1.1, -1}, {-0.6, -1.1, 1}}
	parts := []int{PartLegFL, PartLegFR, PartLegBL, PartLegBR}
	for i, l := range legs {
		s.drawPart(k, parts[i], rl.NewVector3(0.55, legH, 0.6), rl.NewVector3(l[0], legH, l[1]), 0.5, sw*l[2], 0, 0, model, c)
	}
	bodyY := legH + 0.1
	s.drawPart(k, PartBody, rl.NewVector3(1.6, 1.4, 3.6), rl.NewVector3(0, bodyY, 0), -0.5, 0, 0, 0, model, c)
	// Tail: three tapering segments swinging gently.
	wag := float32(math.Sin(float64(p.Phase*0.4))) * 0.15
	s.drawPart(k, PartExtra, rl.NewVector3(0.9, 0.8, 1.8), rl.NewVector3(0, bodyY+0.6, -2.4), 0, -0.1, wag, 0, model, c)
	s.drawPart(k, PartExtra, rl.NewVector3(0.55, 0.5, 1.8), rl.NewVector3(0, bodyY+0.7, -4.0), 0, -0.15, wag*2, 0, model, c)
	s.drawPart(k, PartExtra, rl.NewVector3(0.3, 0.3, 1.8), rl.NewVector3(0, bodyY+0.75, -5.6), 0, -0.2, wag*3, 0, model, c)
	// Neck: three segments pivoting from the shoulders; Pitch raises it toward the leaves.
	lift := clamp(-p.Pitch, -0.3, 1.1) // Pitch < 0 looks up
	n1 := rl.NewVector3(0, bodyY+0.6, 1.8)
	s.drawPart(k, PartArmL, rl.NewVector3(0.7, 0.7, 1.6), n1, 0, -0.35-lift*0.6, 0, 0, model, c)
	seg := func(prev rl.Vector3, ang, length float32) rl.Vector3 {
		return rl.NewVector3(prev.X, prev.Y+length*float32(math.Sin(float64(ang))), prev.Z+length*float32(math.Cos(float64(ang))))
	}
	a1 := 0.35 + lift*0.6
	n2 := seg(n1, a1, 1.5)
	s.drawPart(k, PartArmL, rl.NewVector3(0.55, 0.55, 1.6), n2, 0, -(a1 + lift*0.4), 0, 0, model, c)
	a2 := a1 + lift*0.4
	n3 := seg(n2, a2, 1.5)
	s.drawPart(k, PartArmR, rl.NewVector3(0.45, 0.45, 1.6), n3, 0, -(a2 + lift*0.3), 0, 0, model, c)
	a3 := a2 + lift*0.3
	hp := seg(n3, a3, 1.5)
	s.drawPart(k, PartHead, rl.NewVector3(0.5, 0.45, 0.8), hp, 0, -a3+p.Swing*0.3, 0, 0, model, c)
	if p.Flash > 0 {
		rl.DrawCubeV(rl.Vector3Add(p.Pos, rl.NewVector3(0, 2.2, 0)), rl.NewVector3(2.2, 4.5, 8), rl.Fade(rl.White, p.Flash*0.4))
	}
}

// DrawRaptor: a small, fast biped with a long stiff tail and a sickle claw.
func (s *Skins) DrawRaptor(w *World, p *Pose) {
	s.init(w)
	model := modelMatrix(p)
	c := lightColor(w, rl.Vector3Add(p.Pos, rl.NewVector3(0, 0.6, 0)), p.Alpha)
	k := SkinRaptor
	sw := swingAt(p.Phase, p.Amp) * 1.4
	const legH float32 = 0.7
	for _, sg := range []float32{-1, 1} {
		s.drawPart(k, PartLegL, rl.NewVector3(0.16, legH, 0.2), rl.NewVector3(sg*0.17, legH, -0.05), 0.5, sw*sg, 0, 0, model, c)
		s.drawPart(k, PartLegFL, rl.NewVector3(0.08, 0.14, 0.1), rl.NewVector3(sg*0.17, legH*0.45, 0.14), 0.5, 0.8+sw*sg, 0, 0, model, c) // claw
	}
	bodyY := legH + 0.05
	s.drawPart(k, PartBody, rl.NewVector3(0.4, 0.42, 0.9), rl.NewVector3(0, bodyY, 0), -0.5, 0.1, 0, 0, model, c)
	s.drawPart(k, PartExtra, rl.NewVector3(0.18, 0.16, 1.3), rl.NewVector3(0, bodyY+0.25, -0.5), 0, 0.05, float32(math.Sin(float64(p.Phase)))*0.1, 0, model, c)
	s.drawPart(k, PartArmL, rl.NewVector3(0.25, 0.25, 0.4), rl.NewVector3(0, bodyY+0.35, 0.55), 0, -0.5, 0, 0, model, c) // neck
	s.drawPart(k, PartHead, rl.NewVector3(0.28, 0.26, 0.6), rl.NewVector3(0, bodyY+0.55, 0.9), 0, -p.Pitch+p.Swing*0.4, 0, 0, model, c)
	for _, sg := range []float32{-1, 1} {
		s.drawPart(k, PartArmR, rl.NewVector3(0.08, 0.3, 0.08), rl.NewVector3(sg*0.2, bodyY+0.15, 0.3), 0.5, -1.0, 0, 0, model, c)
	}
	if p.Flash > 0 {
		rl.DrawCubeV(rl.Vector3Add(p.Pos, rl.NewVector3(0, 0.7, 0)), rl.NewVector3(0.6, 1.5, 2.2), rl.Fade(rl.White, p.Flash*0.5))
	}
}

// ---------- behaviour ----------

// dinoTick: brontosaurs browse leaves; raptors hunt in packs; compys scatter.
func (g *Game) dinoTick(a *Animal, dt float32) {
	w := g.World
	switch a.Kind {
	case AnimalBronto:
		a.BiteCD = max(0, a.BiteCD-dt)
		// Look for leaves within reach of the neck and eat them.
		if a.BiteCD == 0 {
			fwd := a.Heading
			for dy := 5; dy >= 2; dy-- {
				for dist := float32(2); dist <= 5; dist++ {
					x := floorI(a.Pos.X + fwd.X*dist)
					z := floorI(a.Pos.Z + fwd.Z*dist)
					y := floorI(a.Pos.Y) + dy
					if b := w.Get(x, y, z); b == Leaves || b == FruitLeaves || b == EucLeaves || b == SpruceLeaves {
						w.Set(x, y, z, Air)
						a.BiteCD = 4 + rand.Float32()*4
						a.Browse = 2.5
						a.Walking = false
						a.WanderT = 3
						if b == FruitLeaves && rand.Float32() < 0.5 {
							g.spawnDrop(rl.NewVector3(float32(x)+0.5, float32(y)+0.5, float32(z)+0.5), Apple, 0)
						}
						g.burst(rl.NewVector3(float32(x)+0.5, float32(y)+0.5, float32(z)+0.5), rl.NewColor(80, 150, 60, 255), 8)
						return
					}
				}
			}
		}
		a.Browse = max(0, a.Browse-dt)
		// Prefer to stay in the swamp.
		lx, lz := wrapX(floorI(a.Pos.X)-originX), wrapZ(floorI(a.Pos.Z)-originZ)
		if w.Biome[lz*worldW+lx] != BiomeSwamp && a.Home.Y != 0 && a.WanderT <= 0.6 {
			to := WrapDelta(a.Home, a.Pos)
			to.Y = 0
			if l := rl.Vector3Length(to); l > 1 {
				a.Heading = rl.Vector3Scale(to, 1/l)
				a.Walking = true
				a.WanderT = 3
			}
		}
	case AnimalRaptor:
		// Packs: hunt any player or animal within range, together.
		t := g.nearestTarget(a.Pos)
		d := WrapDist(t.Pos, a.Pos)
		if d < 14 && a.Flee == 0 && t.ID&villagerIDBit == 0 && !g.Sky.IsNight() == false || (d < 9 && a.Flee == 0 && t.ID&villagerIDBit == 0) {
			a.Flee = 6 // charge
			for _, o := range g.Animals {
				if o != a && o.Alive && o.Kind == AnimalRaptor && WrapDist(o.Pos, a.Pos) < 12 {
					o.Flee = max(o.Flee, 5) // the pack joins in
				}
			}
		}
		// Hop over low obstacles.
		if (a.Walking || a.Flee > 0) && a.VelY == 0 && rand.Float32() < 0.02 {
			a.VelY = 5
		}
	case AnimalCompy:
		// Skittish: scatter from anything big nearby.
		t := g.nearestTarget(a.Pos)
		if WrapDist(t.Pos, a.Pos) < 5 && a.Flee == 0 {
			a.Flee = 3
		}
	}
}

// spawnDinos places dinosaurs by biome: brontosaur herds in swamps, raptor packs
// and compys on the outback.
func (g *Game) spawnDinos(n int) {
	w := g.World
	for i := 0; i < n*40 && n > 0; i++ {
		p := w.RandomFreePoint(g.Player.Pos, 30)
		lx, lz := wrapX(floorI(p.X)-originX), wrapZ(floorI(p.Z)-originZ)
		switch w.Biome[lz*worldW+lx] {
		case BiomeSwamp:
			if !w.PointFree(p, 1.2) {
				continue
			}
			// A small herd.
			for k := 0; k < 2+rand.Intn(2); k++ {
				q := rl.Vector3Add(p, rl.NewVector3(float32(k)*4-4, 0, float32(rand.Intn(5)-2)))
				q.Y = float32(w.SurfaceY(floorI(q.X), floorI(q.Z)))
				b := NewAnimal(q, AnimalBronto)
				b.Home = q
				b.Walking = true
				g.Animals = append(g.Animals, b)
			}
			n--
		case BiomeOutback:
			if rand.Float32() < 0.5 {
				for k := 0; k < 3; k++ {
					q := rl.Vector3Add(p, rl.NewVector3(float32(k)*2-2, 0, float32(k%2)*2))
					q.Y = float32(w.SurfaceY(floorI(q.X), floorI(q.Z)))
					r := NewAnimal(q, AnimalRaptor)
					r.Walking = true
					g.Animals = append(g.Animals, r)
				}
			} else {
				for k := 0; k < 4; k++ {
					q := rl.Vector3Add(p, rl.NewVector3(float32(rand.Intn(5)-2), 0, float32(rand.Intn(5)-2)))
					q.Y = float32(w.SurfaceY(floorI(q.X), floorI(q.Z)))
					cmp := NewAnimal(q, AnimalCompy)
					cmp.Walking = true
					g.Animals = append(g.Animals, cmp)
				}
			}
			n--
		}
	}
}

func (g *Game) dinoCount() int {
	n := 0
	for _, a := range g.Animals {
		if a.Alive && (a.Kind == AnimalBronto || a.Kind == AnimalRaptor || a.Kind == AnimalCompy) {
			n++
		}
	}
	return n
}
