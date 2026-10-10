package main

import (
	"fmt"
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Two dragons: a fire dragon roosting on the highest peak and a frost dragon
// over the snowy taiga. They circle their roosts, dive and breathe fire at
// anyone who comes near, and drop a hoard when slain.

type DragonState int

const (
	DragonCircle DragonState = iota
	DragonAttack
	DragonDying
	DragonDead
)

type Dragon struct {
	ID     int
	Name   string
	Frost  bool
	Pos    rl.Vector3
	Vel    rl.Vector3
	Roost  rl.Vector3
	HP     int
	MaxHP  int
	State  DragonState
	Angle  float32 // circling angle
	Flap   float32
	FireCD float32
	BiteCD float32
	RoarCD float32
	Anger  float32 // seconds it stays hostile after being hurt
	Flash  float32
	DeathT float32
	Target uint32 // player id it is attacking
}

type Fireball struct {
	Pos   rl.Vector3
	Vel   rl.Vector3
	Life  float32
	Frost bool
}

const (
	dragonHP     = 150
	dragonAlert  = 40.0 // blocks from the roost that wake it
	dragonChase  = 70.0 // beyond this it gives up
	dragonSpeed  = 10.0
	dragonHeight = 14.0 // circling altitude above the roost
)

// spawnDragons picks two roosts far apart: the highest point, and the highest snowy point.
func (g *Game) spawnDragons() {
	w := g.World
	g.Dragons = nil
	best, snowBest := -1, -1
	var bx, bz, sx, sz int
	for try := 0; try < 4000; try++ {
		lx, lz := rand.Intn(worldW), rand.Intn(worldD)
		h := w.Ground[lz*worldW+lx]
		if h <= seaLevel+2 {
			continue
		}
		if dx, dz := lx+originX, lz+originZ; dx*dx+dz*dz < 60*60 {
			continue // not over the spawn
		}
		if h > best {
			best, bx, bz = h, lx, lz
		}
		if w.getLocal(lx, h-1, lz) == Snow && h > snowBest {
			snowBest, sx, sz = h, lx, lz
		}
	}
	if best < 0 {
		return
	}
	if snowBest < 0 || (sx == bx && sz == bz) {
		// No snow: put the frost dragon on the far side of the world instead.
		sx, sz = wrapX(bx+worldW/2), wrapZ(bz+worldD/2)
		snowBest = w.Ground[sz*worldW+sx]
	}
	mk := func(id int, name string, frost bool, lx, h, lz int) {
		roost := rl.NewVector3(float32(lx+originX)+0.5, float32(h), float32(lz+originZ)+0.5)
		d := &Dragon{ID: id, Name: name, Frost: frost, Roost: roost, HP: dragonHP, MaxHP: dragonHP, Angle: rand.Float32() * 6.28}
		d.Pos = rl.NewVector3(roost.X+22, roost.Y+dragonHeight, roost.Z)
		g.Dragons = append(g.Dragons, d)
	}
	mk(1, "Fire Dragon", false, bx, best, bz)
	mk(2, "Frost Dragon", true, sx, snowBest, sz)
}

// tickDragons runs the dragons and their fireballs (host and solo).
func (g *Game) tickDragons(dt float32) {
	for _, d := range g.Dragons {
		g.tickDragon(d, dt)
	}
	// Fireballs.
	keep := g.Fireballs[:0]
	for i := range g.Fireballs {
		f := &g.Fireballs[i]
		f.Life -= dt
		f.Vel.Y -= 3 * dt
		next := rl.Vector3Add(f.Pos, rl.Vector3Scale(f.Vel, dt))
		hit := f.Life <= 0 || g.World.Solid(floorI(next.X), floorI(next.Y), floorI(next.Z))
		for _, t := range g.targets() {
			if WrapDist(next, rl.Vector3Add(t.Pos, rl.NewVector3(0, 0.9, 0))) < 1.3 {
				hit = true
			}
		}
		if hit {
			g.blast(next, 1.6, 34)
			if f.Frost {
				g.burst(next, rl.NewColor(180, 230, 255, 255), 20)
				for _, t := range g.targets() {
					if t.ID == 0 && WrapDist(next, t.Pos) < 4 {
						g.Player.Knock = rl.Vector3Scale(g.Player.Knock, 0.2)
						g.Player.FrostT = 3 // slowed
					}
				}
			}
			continue
		}
		f.Pos = WrapPos(next)
		g.Sparks = append(g.Sparks, Spark{Pos: f.Pos, Vel: rl.NewVector3(rand.Float32()-0.5, rand.Float32(), rand.Float32()-0.5), Life: 0.25, Col: fireColor(f.Frost)})
		keep = append(keep, *f)
	}
	g.Fireballs = keep
}

func fireColor(frost bool) rl.Color {
	if frost {
		return rl.NewColor(150, 220, 255, 255)
	}
	return rl.NewColor(255, 140, 40, 255)
}

func (g *Game) tickDragon(d *Dragon, dt float32) {
	d.Flash = max(0, d.Flash-dt*3)
	d.FireCD = max(0, d.FireCD-dt)
	d.BiteCD = max(0, d.BiteCD-dt)
	d.RoarCD = max(0, d.RoarCD-dt)
	d.Anger = max(0, d.Anger-dt)
	d.Flap += dt * 6
	switch d.State {
	case DragonDead:
		return
	case DragonDying:
		d.DeathT += dt
		d.Vel.Y -= 12 * dt
		d.Pos = rl.Vector3Add(d.Pos, rl.Vector3Scale(d.Vel, dt))
		if g.World.Solid(floorI(d.Pos.X), floorI(d.Pos.Y), floorI(d.Pos.Z)) || d.DeathT > 6 {
			g.dragonFalls(d)
		}
		return
	}
	// Who is near? Players only (dragons ignore the villages' wars).
	var target *Target
	var td float32 = 1e9
	for _, t := range g.targets() {
		if t.ID&villagerIDBit != 0 {
			continue
		}
		if dist := WrapDist(t.Pos, d.Roost); dist < td {
			tt := t
			target, td = &tt, dist
		}
	}
	if d.State == DragonCircle {
		if target != nil && (td < dragonAlert || d.Anger > 0 && WrapDist(target.Pos, d.Pos) < dragonChase) {
			d.State = DragonAttack
			d.Target = target.ID
			g.announce("The "+d.Name+" has seen you!", 3)
			g.Audio.Play(g.Audio.Roar, 1)
		}
	}
	switch d.State {
	case DragonCircle:
		d.Angle += dt * dragonSpeed / 24
		want := rl.NewVector3(d.Roost.X+float32(math.Cos(float64(d.Angle)))*24, d.Roost.Y+dragonHeight+float32(math.Sin(float64(d.Angle*2)))*2, d.Roost.Z+float32(math.Sin(float64(d.Angle)))*24)
		g.dragonSteer(d, want, dragonSpeed, dt)
		if d.RoarCD == 0 {
			d.RoarCD = 20 + rand.Float32()*20
			if WrapDist(d.Pos, g.Player.Pos) < 90 {
				g.Audio.Play(g.Audio.Roar, 0.6*clamp(1-WrapDist(d.Pos, g.Player.Pos)/90, 0, 1))
			}
		}
	case DragonAttack:
		if target == nil || (WrapDist(target.Pos, d.Roost) > dragonChase && d.Anger == 0) {
			d.State = DragonCircle
			return
		}
		tp := Near(target.Pos, d.Pos)
		flat := rl.Vector3Subtract(tp, d.Pos)
		flat.Y = 0
		hd := rl.Vector3Length(flat)
		// Swoop: dive to bite when the fire is on cooldown, else hover above and breathe.
		var want rl.Vector3
		if d.FireCD > 1.2 && hd < 12 {
			want = rl.NewVector3(tp.X, tp.Y+1.5, tp.Z)
		} else {
			want = rl.NewVector3(tp.X-flat.X/max(hd, 1)*8, tp.Y+9, tp.Z-flat.Z/max(hd, 1)*8)
		}
		g.dragonSteer(d, want, dragonSpeed*1.2, dt)
		if hd < 16 && d.FireCD == 0 && math.Abs(float64(d.Pos.Y-tp.Y)) < 14 {
			d.FireCD = 3.5
			from := rl.Vector3Add(d.Pos, rl.Vector3Scale(rl.Vector3Normalize(d.Vel), 2.2))
			to := rl.Vector3Add(tp, rl.NewVector3(0, 1, 0))
			dir := rl.Vector3Normalize(rl.Vector3Subtract(to, from))
			g.Fireballs = append(g.Fireballs, Fireball{Pos: from, Vel: rl.Vector3Scale(dir, 18), Life: 3, Frost: d.Frost})
			g.Audio.Play(g.Audio.Burn, 0.9)
		}
		if rl.Vector3Distance(d.Pos, rl.Vector3Add(tp, rl.NewVector3(0, 1, 0))) < 3 && d.BiteCD == 0 {
			d.BiteCD = 2
			g.hurtTargetFrom(target.ID, int(22*damageScale()), "was devoured by the "+d.Name, true, d.Pos, 9)
		}
	}
}

// dragonSteer eases the dragon toward a point, keeping it above the ground.
func (g *Game) dragonSteer(d *Dragon, want rl.Vector3, speed float32, dt float32) {
	to := rl.Vector3Subtract(Near(want, d.Pos), d.Pos)
	if l := rl.Vector3Length(to); l > 0.01 {
		to = rl.Vector3Scale(to, speed/l)
	}
	d.Vel = rl.Vector3Add(d.Vel, rl.Vector3Scale(rl.Vector3Subtract(to, d.Vel), min(dt*1.5, 1)))
	d.Pos = rl.Vector3Add(d.Pos, rl.Vector3Scale(d.Vel, dt))
	ground := float32(g.World.SurfaceY(floorI(d.Pos.X), floorI(d.Pos.Z)))
	if d.Pos.Y < ground+2 {
		d.Pos.Y = ground + 2
		d.Vel.Y = max(d.Vel.Y, 0)
	}
	d.Pos = WrapPos(d.Pos)
}

// BB is the dragon's hit box.
func (d *Dragon) BB() rl.BoundingBox {
	return rl.NewBoundingBox(rl.Vector3Subtract(d.Pos, rl.NewVector3(2.2, 1.2, 2.2)), rl.Vector3Add(d.Pos, rl.NewVector3(2.2, 1.4, 2.2)))
}

// hitDragon finds the first living dragon along a ray within maxD (boxes brought near the origin).
func (g *Game) hitDragon(ray rl.Ray, maxD float32) (*Dragon, float32) {
	var best *Dragon
	bd := maxD
	for _, d := range g.Dragons {
		if d.State == DragonDead || d.State == DragonDying {
			continue
		}
		bb := d.BB()
		n := Near(d.Pos, ray.Position)
		off := rl.Vector3Subtract(n, d.Pos)
		bb = rl.NewBoundingBox(rl.Vector3Add(bb.Min, off), rl.Vector3Add(bb.Max, off))
		if c := rl.GetRayCollisionBox(ray, bb); c.Hit && c.Distance < bd {
			best, bd = d, c.Distance
		}
	}
	return best, bd
}

// damageDragon applies a hit; dragons remember who hurt them.
func (g *Game) damageDragon(d *Dragon, dmg int) {
	if d.State == DragonDead || d.State == DragonDying {
		return
	}
	d.HP -= dmg
	d.Flash = 1
	d.Anger = 40
	if d.State == DragonCircle {
		d.State = DragonAttack
		g.announce("The "+d.Name+" turns on you!", 3)
		g.Audio.Play(g.Audio.Roar, 1)
	}
	if d.HP <= 0 {
		d.State = DragonDying
		d.DeathT = 0
		d.Vel = rl.NewVector3(d.Vel.X*0.3, -2, d.Vel.Z*0.3)
		g.announce("The "+d.Name+" is falling!", 3)
		g.Audio.Play(g.Audio.Roar, 1)
	}
}

func (g *Game) dragonFalls(d *Dragon) {
	d.State = DragonDead
	at := rl.NewVector3(d.Pos.X, d.Pos.Y+1, d.Pos.Z)
	g.blast(at, 2.2, 10)
	for i := 0; i < 5; i++ {
		g.spawnDrop(at, DiamondOre, 0)
	}
	for i := 0; i < 3; i++ {
		g.spawnDrop(at, GoldOre, 0)
	}
	g.spawnDrop(at, 0, 48)
	g.Score += 2500
	g.announce(fmt.Sprintf("%s SLAIN  +2500  -  its hoard lies where it fell", d.Name), 5)
	g.unlock(AchDragon)
	alive := 0
	for _, o := range g.Dragons {
		if o.State != DragonDead {
			alive++
		}
	}
	if alive == 0 {
		g.Score += 2500
		g.announce("Both dragons are dead. The skies are yours.  +2500", 6)
	}
}

// ---------- drawing ----------

// DrawDragon draws a dragon: long body, neck and head, tail, two flapping wings, tucked legs.
func (s *Skins) DrawDragon(w *World, p *Pose, frost bool, flap float32) {
	s.init(w)
	model := modelMatrix(p)
	c := lightColor(w, p.Pos, p.Alpha)
	k := SkinDragon
	if frost {
		k = SkinFrostDragon
	}
	wing := float32(math.Sin(float64(flap))) * 0.6
	s.drawPart(k, PartBody, rl.NewVector3(1.2, 1.0, 3.2), rl.NewVector3(0, 0, 0), 0, 0, 0, 0, model, c)
	s.drawPart(k, PartArmL, rl.NewVector3(0.8, 0.7, 1.6), rl.NewVector3(0, 0.3, 2.2), 0, -0.4+p.Pitch, 0, 0, model, c) // neck
	s.drawPart(k, PartHead, rl.NewVector3(0.9, 0.7, 1.5), rl.NewVector3(0, 0.9, 3.4), 0, -0.1+p.Pitch+p.Swing*0.5, 0, 0, model, c)
	s.drawPart(k, PartExtra, rl.NewVector3(0.7, 0.6, 2.2), rl.NewVector3(0, 0.1, -2.6), 0, 0.15, float32(math.Sin(float64(flap*0.5)))*0.2, 0, model, c)
	s.drawPart(k, PartExtra, rl.NewVector3(0.4, 0.35, 2.2), rl.NewVector3(0, 0.3, -4.7), 0, 0.25, float32(math.Sin(float64(flap*0.5)))*0.4, 0, model, c)
	for _, sg := range []float32{-1, 1} {
		// Wings pivot at the shoulder and flap about the body axis.
		s.drawPart(k, PartArmR, rl.NewVector3(3.2, 0.12, 2.2), rl.NewVector3(sg*0.6, 0.45, 0.4), 0, 0, 0, sg*(0.3+wing), model, c)
		s.drawPart(k, PartArmR, rl.NewVector3(2.4, 0.1, 1.6), rl.NewVector3(sg*3.6, 0.45+sg*0, 0.3), 0, 0, 0, sg*(0.3+wing)*1.6, model, c)
		s.drawPart(k, PartLegL, rl.NewVector3(0.3, 0.8, 0.3), rl.NewVector3(sg*0.45, -0.5, -0.6), 0.5, 1.0, 0, 0, model, c)
		s.drawPart(k, PartLegL, rl.NewVector3(0.25, 0.6, 0.25), rl.NewVector3(sg*0.4, -0.5, 1.0), 0.5, 1.1, 0, 0, model, c)
	}
	if p.Flash > 0 {
		rl.DrawCubeV(p.Pos, rl.NewVector3(3, 2, 7), rl.Fade(rl.White, p.Flash*0.4))
	}
}

func (g *Game) drawDragons(cam rl.Camera3D) {
	for _, d := range g.Dragons {
		if d.State == DragonDead {
			continue
		}
		pos := Near(d.Pos, cam.Position)
		if rl.Vector3Distance(pos, cam.Position) > 200 {
			continue
		}
		yaw := float32(math.Atan2(float64(d.Vel.X), float64(d.Vel.Z)))
		pitch := float32(0)
		if l := rl.Vector3Length(d.Vel); l > 0.1 {
			pitch = -float32(math.Asin(float64(d.Vel.Y / l)))
		}
		pose := Pose{Pos: pos, Yaw: yaw, Pitch: pitch * 0.5, Scale: 1.4, Lum: g.World.Luminance(pos, g.Sky.Light()), Alpha: 1, Flash: d.Flash}
		if d.State == DragonDying {
			pose.Death = clamp(d.DeathT*0.5, 0, 1)
		}
		if d.FireCD > 3.2 {
			pose.Swing = 1 // jaws open as it breathes
		}
		skins.DrawDragon(g.World, &pose, d.Frost, d.Flap)
		if d.HP < d.MaxHP && d.State != DragonDying {
			top := pos.Y + 3
			frac := float32(d.HP) / float32(d.MaxHP)
			rl.DrawCubeV(rl.NewVector3(pos.X, top, pos.Z), rl.NewVector3(4, 0.12, 0.12), rl.NewColor(0, 0, 0, 180))
			rl.DrawCubeV(rl.NewVector3(pos.X-(1-frac)*2, top, pos.Z), rl.NewVector3(frac*4, 0.14, 0.14), rl.Red)
		}
	}
	for i := range g.Fireballs {
		f := &g.Fireballs[i]
		pos := Near(f.Pos, cam.Position)
		rl.DrawSphere(pos, 0.45, fireColor(f.Frost))
		rl.DrawSphere(pos, 0.7, rl.Fade(fireColor(f.Frost), 0.35))
	}
}

// dragonHint names the nearest living dragon for the HUD.
func (g *Game) dragonHint() string {
	var best *Dragon
	bd := float32(1e9)
	for _, d := range g.Dragons {
		if d.State == DragonDead {
			continue
		}
		if dist := WrapDist(d.Pos, g.Player.Pos); dist < bd {
			best, bd = d, dist
		}
	}
	if best == nil {
		return ""
	}
	if bd > 120 {
		return ""
	}
	state := "circling"
	if best.State == DragonAttack {
		state = "HUNTING YOU"
	}
	return fmt.Sprintf("%s %dm, %s", best.Name, int(bd), state)
}
