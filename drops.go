package main

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Drop is an item lying in the world: a block (or ammo) that pops out of mined
// blocks and slain hostiles and is collected by walking over it.
type Drop struct {
	Pos   rl.Vector3
	Vel   rl.Vector3
	Block Block
	Count int // blocks in the stack (0 means one)
	Ammo  int
	Age   float32
	Spin  float32
}

func (g *Game) spawnDrop(pos rl.Vector3, b Block, ammo int) {
	v := rl.NewVector3(rand.Float32()*2-1, 2.5+rand.Float32()*1.5, rand.Float32()*2-1)
	g.Drops = append(g.Drops, Drop{Pos: pos, Vel: v, Block: b, Ammo: ammo, Spin: rand.Float32() * 6})
}

func (g *Game) updateDrops(dt float32) {
	p := g.Player
	keep := g.Drops[:0]
	for i := range g.Drops {
		d := &g.Drops[i]
		d.Age += dt
		d.Spin += dt * 2
		if g.World.WaterAt(d.Pos) {
			d.Vel.Y = lerp(d.Vel.Y, 0.6, dt*3) // floats up
			d.Vel.X *= 1 - dt*2
			d.Vel.Z *= 1 - dt*2
		} else {
			d.Vel.Y = max(d.Vel.Y-gravity*dt, -25)
		}
		var res MoveResult
		d.Pos, res = g.World.MoveBox(d.Pos, 0.12, 0.25, rl.Vector3Scale(d.Vel, dt), false)
		if res.Ground {
			d.Vel.Y = 0
			d.Vel.X *= 1 - dt*8
			d.Vel.Z *= 1 - dt*8
		}
		if res.Wall {
			d.Vel.X, d.Vel.Z = 0, 0
		}
		if d.Age > 240 {
			continue
		}
		flat := rl.Vector3Distance(rl.NewVector3(d.Pos.X, 0, d.Pos.Z), rl.NewVector3(p.Pos.X, 0, p.Pos.Z))
		if d.Age > 0.5 && flat < 1.3 && math.Abs(float64(p.Pos.Y-d.Pos.Y)) < 1.8 {
			if d.Ammo > 0 {
				p.Reserve += d.Ammo
			} else {
				p.Inv[d.Block] += max(1, d.Count)
			}
			g.Audio.Play(g.Audio.Pickup, 0.5)
			continue
		}
		keep = append(keep, *d)
	}
	g.Drops = keep
}

func (g *Game) drawDrops() {
	for i := range g.Drops {
		d := &g.Drops[i]
		bob := float32(math.Sin(float64(d.Spin*1.5))) * 0.05
		pos := rl.NewVector3(d.Pos.X, d.Pos.Y+0.18+bob, d.Pos.Z)
		if d.Ammo > 0 {
			rl.DrawCube(pos, 0.28, 0.2, 0.28, rl.NewColor(240, 190, 50, 255))
			rl.DrawCubeWires(pos, 0.28, 0.2, 0.28, rl.NewColor(120, 80, 10, 255))
			continue
		}
		m := rl.MatrixMultiply(rl.MatrixMultiply(rl.MatrixScale(0.3, 0.3, 0.3), rl.MatrixRotateY(d.Spin)), rl.MatrixTranslate(pos.X, pos.Y, pos.Z))
		g.World.DrawBlockAt(d.Block, m)
	}
}
