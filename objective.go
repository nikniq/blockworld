package main

import (
	"fmt"
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// The objective: an ancient beacon tower stands far from spawn. Light it with
// three diamonds to conquer the world. The game continues afterwards.

// placeBeaconTower builds the tower on the surface far from the spawn.
func (w *World) placeBeaconTower(heights []int) {
	best := -1
	var bx, bz int
	for try := 0; try < 200; try++ {
		ang := rand.Float64() * 2 * math.Pi
		dist := float64(worldW) * (0.32 + rand.Float64()*0.12)
		x := wrapX(int(math.Cos(ang)*dist) - originX)
		z := wrapZ(int(math.Sin(ang)*dist) - originZ)
		h := heights[z*worldW+x]
		if h <= seaLevel+1 {
			continue
		}
		if h > best {
			best, bx, bz = h, x, z
		}
	}
	if best < 0 {
		bx, bz = wrapX(worldW/3), wrapZ(worldD/3)
		best = max(heights[bz*worldW+bx], seaLevel+2)
	}
	base := best
	const half, height = 2, 14
	for dz := -half - 1; dz <= half+1; dz++ {
		for dx := -half - 1; dx <= half+1; dx++ {
			w.flattenColumn(bx+dx, bz+dz, base)
			w.setLocal(bx+dx, base-1, bz+dz, StoneBrick)
		}
	}
	for y := base; y < base+height; y++ {
		for dz := -half; dz <= half; dz++ {
			for dx := -half; dx <= half; dx++ {
				edge := abs(dx) == half || abs(dz) == half
				b := Air
				if edge {
					b = StoneBrick
					if (y-base)%4 == 3 && (dx == 0 || dz == 0) {
						b = Glass
					}
				}
				w.setLocal(bx+dx, y, bz+dz, b)
			}
		}
	}
	// Doorway and a ladder up the inside of the far wall to the roof hatch.
	w.setLocal(bx, base, bz+half, DoorClosed)
	w.setLocal(bx, base+1, bz+half, Air)
	for y := base; y < base+height; y++ {
		w.setLocal(bx, y, bz-half+1, Ladder)
	}
	// Roof with a hatch over the ladder and the beacon in the middle.
	for dz := -half; dz <= half; dz++ {
		for dx := -half; dx <= half; dx++ {
			w.setLocal(bx+dx, base+height, bz+dz, StoneBrick)
		}
	}
	w.setLocal(bx, base+height, bz-half+1, Air)
	w.setLocal(bx, base+height, bz-half+1, Ladder)
	w.setLocal(bx, base+height+1, bz, Beacon)
	for _, d := range [][2]int{{-half, -half}, {half, -half}, {-half, half}, {half, half}} {
		w.setLocal(bx+d[0], base+height+1, bz+d[1], Torch)
	}
	w.Beacon = rl.NewVector3(float32(bx+originX)+0.5, float32(base+height+1), float32(bz+originZ)+0.5)
}

// findBeacon locates the beacon block (after loading a save).
func (w *World) findBeacon() {
	for i, b := range w.Blocks {
		if b == Beacon || b == BeaconLit {
			x := i%worldW + originX
			z := (i/worldW)%worldD + originZ
			y := i / (worldW * worldD)
			w.Beacon = rl.NewVector3(float32(x)+0.5, float32(y), float32(z)+0.5)
			w.BeaconLit = b == BeaconLit
			return
		}
	}
}

// lightBeacon is the win: costs three diamond ore.
func (g *Game) lightBeacon(x, y, z int) {
	p := g.Player
	if g.World.Get(x, y, z) != Beacon {
		return
	}
	if p.Inv[DiamondOre] < 3 && !settings.Creative {
		g.say(fmt.Sprintf("The beacon hungers for diamonds: you have %d of 3", p.Inv[DiamondOre]), 2.5)
		g.Audio.Play(g.Audio.Click, 0.6)
		return
	}
	if !settings.Creative {
		p.Inv[DiamondOre] -= 3
	}
	g.World.Set(x, y, z, BeaconLit)
	g.World.BeaconLit = true
	g.Score += 5000
	g.Won = true
	g.announce("THE BEACON IS LIT  -  Blockworld is yours!  +5000", 6)
	g.Audio.Play(g.Audio.Clear, 1)
	g.Audio.Play(g.Audio.Explode, 0.5)
	centre := rl.NewVector3(float32(x)+0.5, float32(y)+1, float32(z)+0.5)
	for i := 0; i < 80; i++ {
		c := rl.NewColor(uint8(120+rand.Intn(135)), uint8(120+rand.Intn(135)), uint8(200+rand.Intn(55)), 255)
		v := rl.NewVector3(rand.Float32()*2-1, 2+rand.Float32()*4, rand.Float32()*2-1)
		g.Sparks = append(g.Sparks, Spark{Pos: centre, Vel: rl.Vector3Scale(v, 2.5), Life: 1 + rand.Float32(), Col: c})
	}
	g.unlock(AchBeacon)
	p.EnsureHeld()
}

// beaconHint is the HUD line pointing at the beacon.
func (g *Game) beaconHint() string {
	w := g.World
	if w.Beacon.X == 0 && w.Beacon.Z == 0 && w.Beacon.Y == 0 {
		return ""
	}
	if w.BeaconLit {
		return "Beacon lit"
	}
	d := WrapDelta(w.Beacon, g.Player.Pos)
	dist := int(math.Sqrt(float64(d.X*d.X + d.Z*d.Z)))
	deg := math.Atan2(float64(d.X), float64(-d.Z)) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	dirs := [...]string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	return fmt.Sprintf("Beacon %dm %s", dist, dirs[int((deg+22.5)/45)%8])
}
