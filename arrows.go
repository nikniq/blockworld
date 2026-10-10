package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Arrow is a skeleton's projectile.
type Arrow struct {
	Pos   rl.Vector3
	Vel   rl.Vector3
	Life  float32
	Owner uint32 // 0: a hostile's arrow; otherwise the player id + 1 that shot it
}

const playerArrowDamage = 4

const arrowDamage = 6

func (g *Game) shootArrow(from rl.Vector3, target rl.Vector3) {
	to := rl.Vector3Subtract(target, from)
	d := rl.Vector3Length(to)
	if d < 0.1 {
		return
	}
	dir := rl.Vector3Scale(to, 1/d)
	// Lob slightly upward to compensate for gravity over the flight.
	dir.Y += d * 0.012
	dir = rl.Vector3Normalize(dir)
	g.Arrows = append(g.Arrows, Arrow{Pos: from, Vel: rl.Vector3Scale(dir, 22), Life: 4})
	g.Audio.Play(g.Audio.Swing, 0.5)
}

// playerShoot fires the local player's bow (solo and host) or asks the host to (client).
func (g *Game) playerShoot() {
	p := g.Player
	from := rl.Vector3Add(p.Eye(), rl.Vector3Scale(p.Forward(), 0.5))
	vel := rl.Vector3Scale(p.Forward(), 28)
	g.Audio.Play(g.Audio.Swing, 0.6)
	if g.isClient() {
		g.sendToHost(&Msg{Shoot: &struct{ Pos, Vel rl.Vector3 }{from, vel}})
		return
	}
	g.Arrows = append(g.Arrows, Arrow{Pos: from, Vel: vel, Life: 4, Owner: 1})
}

// arrowHitsCreature applies a player's arrow to the first hostile or animal on its path.
func (g *Game) arrowHitsCreature(a *Arrow, ray rl.Ray, step float32) bool {
	if d, dd := g.hitDragon(ray, step); d != nil && dd <= step {
		g.burst(a.Pos, rl.NewColor(255, 120, 60, 255), 6)
		g.damageDragon(d, playerArrowDamage*2)
		return true
	}
	for _, e := range g.Enemies {
		if !e.Alive {
			continue
		}
		if c := rl.GetRayCollisionBox(ray, e.BB()); c.Hit && c.Distance <= step {
			g.burst(a.Pos, rl.NewColor(255, 80, 80, 255), 6)
			head := rl.GetRayCollisionSphere(ray, e.HeadCenter(), e.Spec.HeadR)
			dmg := playerArrowDamage
			if head.Hit {
				dmg *= 2
			}
			if e.Hit(dmg) {
				if a.Owner == 1 {
					g.killEnemy(e, e.Spec.Points)
				} else {
					g.killEnemyFor(e, e.Spec.Points, a.Owner-1)
				}
			}
			return true
		}
	}
	for _, an := range g.Animals {
		if !an.Alive {
			continue
		}
		if c := rl.GetRayCollisionBox(ray, an.BB()); c.Hit && c.Distance <= step {
			g.burst(a.Pos, rl.NewColor(255, 80, 80, 255), 6)
			if an.Hit(playerArrowDamage) {
				g.killAnimal(an)
			}
			return true
		}
	}
	return false
}

func (g *Game) updateArrows(dt float32) {
	keep := g.Arrows[:0]
	for i := range g.Arrows {
		a := &g.Arrows[i]
		a.Life -= dt
		a.Vel.Y -= 9 * dt
		step := rl.Vector3Scale(a.Vel, dt)
		next := rl.Vector3Add(a.Pos, step)
		if a.Life <= 0 {
			continue
		}
		ray := rl.NewRay(a.Pos, rl.Vector3Normalize(step))
		hit := false
		if a.Owner != 0 {
			hit = g.arrowHitsCreature(a, ray, rl.Vector3Length(step))
		} else {
			// A hostile's arrow: hit a player?
			for _, t := range g.targets() {
				if c := rl.GetRayCollisionBox(ray, t.Box); c.Hit && c.Distance <= rl.Vector3Length(step) {
					g.hurtTargetFrom(t.ID, int(float32(arrowDamage)*damageScale()+0.5), "was shot by a Skeleton", true, rl.Vector3Subtract(a.Pos, step), 4)
					hit = true
					break
				}
			}
		}
		if hit {
			continue
		}
		// Hit a block?
		if g.World.Solid(floorI(next.X), floorI(next.Y), floorI(next.Z)) {
			g.burst(a.Pos, rl.NewColor(200, 190, 170, 255), 4)
			continue
		}
		a.Pos = WrapPos(next)
		keep = append(keep, *a)
	}
	g.Arrows = keep
}

func (g *Game) drawArrows() {
	for i := range g.Arrows {
		a := &g.Arrows[i]
		v := rl.Vector3Normalize(a.Vel)
		yaw := float32(math.Atan2(float64(v.X), float64(v.Z)))
		pitch := float32(-math.Asin(float64(v.Y)))
		m := rl.MatrixMultiply(rl.MatrixMultiply(rl.MatrixRotateX(pitch), rl.MatrixRotateY(yaw)), rl.MatrixTranslate(a.Pos.X, a.Pos.Y, a.Pos.Z))
		rl.PushMatrix()
		rl.MultMatrix(m)
		rl.DrawCube(rl.Vector3{}, 0.06, 0.06, 0.7, rl.NewColor(190, 170, 130, 255))
		rl.DrawCube(rl.NewVector3(0, 0, 0.35), 0.1, 0.1, 0.12, rl.NewColor(120, 120, 125, 255))
		rl.PopMatrix()
	}
}
