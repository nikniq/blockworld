package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Arrow is a skeleton's projectile.
type Arrow struct {
	Pos  rl.Vector3
	Vel  rl.Vector3
	Life float32
}

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

func (g *Game) updateArrows(dt float32) {
	p := g.Player
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
		// Hit the player?
		ray := rl.NewRay(a.Pos, rl.Vector3Normalize(step))
		if c := rl.GetRayCollisionBox(ray, p.Box()); c.Hit && c.Distance <= rl.Vector3Length(step) {
			p.Damage(arrowDamage)
			g.Audio.Play(g.Audio.Hurt, 0.8)
			continue
		}
		// Hit a block?
		if g.World.Solid(floorI(next.X), floorI(next.Y), floorI(next.Z)) {
			g.burst(a.Pos, rl.NewColor(200, 190, 170, 255), 4)
			continue
		}
		a.Pos = next
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
