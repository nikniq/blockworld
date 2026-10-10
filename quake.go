package main

import (
	"fmt"
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Earthquakes. Every so often the ground near a player rumbles for a few
// seconds, then a fault tears across the land: one side heaves up a block,
// the other drops, and a fissure opens along the line. Built blocks,
// villages, flags and the beacon are left alone; everything underwater is
// too. The host runs the shift and its block updates reach clients the
// usual way; clients only shake and show the epicentre.

const (
	quakeEvery  = 540.0 // mean seconds between quakes
	quakeWarn   = 5.0   // seconds of tremor before the ground moves
	quakeDur    = 7.0   // seconds the fault takes to spread to full radius
	quakeRadius = 28.0  // blocks from the epicentre that move
	quakeDepth  = 6     // blocks below the surface that move with it
	maxFaults   = 6
)

type Quake struct {
	Epi    rl.Vector3 // epicentre on the surface
	Dir    float32    // fault line bearing (radians)
	T      float32    // seconds since the warning
	Done   float32    // radius already shifted
	Active bool
}

// Fault marks a past quake for the map.
type Fault struct {
	Pos rl.Vector3
	Dir float32
	Day int
}

// tickQuake schedules and runs the event (host and solo only).
func (g *Game) tickQuake(dt float32) {
	if settings.Difficulty == 0 && !g.Quake.Active {
		return
	}
	q := &g.Quake
	if !q.Active {
		g.QuakeCD -= dt
		if g.QuakeCD > 0 {
			return
		}
		g.QuakeCD = quakeEvery*0.6 + rand.Float32()*quakeEvery*0.8
		ts := g.targets()
		if len(ts) == 0 {
			return
		}
		ref := ts[rand.Intn(len(ts))].Pos
		ang := rand.Float64() * 2 * math.Pi
		dist := 10 + rand.Float64()*30
		x, z := ref.X+float32(math.Cos(ang)*dist), ref.Z+float32(math.Sin(ang)*dist)
		g.startQuake(WrapPos(rl.NewVector3(x, float32(g.World.SurfaceY(floorI(x), floorI(z))), z)), rand.Float32()*math.Pi)
		return
	}
	q.T += dt
	if q.T < quakeWarn {
		g.Shake = max(g.Shake, 0.35)
		return
	}
	g.Shake = max(g.Shake, 0.9)
	g.quakeShift(min(quakeRadius, (q.T-quakeWarn)/quakeDur*quakeRadius))
	if q.T >= quakeWarn+quakeDur {
		q.Active = false
		g.announce("The earthquake subsides. A fault line scars the land.", 4)
		g.unlock(AchQuake)
	}
}

// startQuake begins the tremor at an epicentre with a fault bearing.
func (g *Game) startQuake(epi rl.Vector3, dir float32) {
	g.Quake = Quake{Epi: epi, Dir: dir, Active: true}
	g.Faults = append(g.Faults, Fault{Pos: epi, Dir: dir, Day: g.Sky.Day})
	if len(g.Faults) > maxFaults {
		g.Faults = g.Faults[len(g.Faults)-maxFaults:]
	}
	g.announce("The ground rumbles... EARTHQUAKE!  "+g.quakeBearing(), 4)
	g.Audio.Play(g.Audio.Thunder, 1)
	g.sendFx(Fx{Kind: FxQuake, Pos: epi, Shake: dir})
}

func (g *Game) quakeBearing() string {
	d := WrapDelta(g.Quake.Epi, g.Player.Pos)
	dist := int(math.Sqrt(float64(d.X*d.X + d.Z*d.Z)))
	deg := math.Atan2(float64(d.X), float64(-d.Z)) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	dirs := [...]string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	return fmt.Sprintf("epicentre %dm %s", dist, dirs[int((deg+22.5)/45)%8])
}

// quakeShift moves every column between the radius already done and r.
func (g *Game) quakeShift(r float32) {
	q := &g.Quake
	if r <= q.Done {
		return
	}
	w := g.World
	ex, ez := floorI(q.Epi.X), floorI(q.Epi.Z)
	dx, dz := float32(math.Cos(float64(q.Dir))), float32(math.Sin(float64(q.Dir)))
	n := int(math.Ceil(float64(r)))
	for oz := -n; oz <= n; oz++ {
		for ox := -n; ox <= n; ox++ {
			d := float32(math.Sqrt(float64(ox*ox + oz*oz)))
			if d <= q.Done || d > r {
				continue
			}
			x, z := ex+ox, ez+oz
			ground := w.SurfaceY(x, z)
			if ground <= seaLevel+1 || ground < quakeDepth+3 {
				continue // under water or too deep
			}
			if g.quakeProtected(x, z) {
				continue
			}
			// Signed distance from the fault line.
			side := float32(ox)*dz - float32(oz)*dx
			switch {
			case d > quakeRadius*0.8 && rand.Float32() < (d-quakeRadius*0.8)/(quakeRadius*0.2):
				continue // the shift fades out at the edge
			case abs32(side) < 1.2 && d < quakeRadius*0.85:
				g.quakeFissure(x, z, ground)
			case side > 0:
				g.shiftColumn(x, z, ground, 1)
			default:
				g.shiftColumn(x, z, ground, -1)
			}
		}
	}
	q.Done = r
	g.unburyAll(q.Epi, r+2)
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// quakeProtected reports whether a column is built on or near a settlement.
func (g *Game) quakeProtected(x, z int) bool {
	w := g.World
	p := rl.NewVector3(float32(x)+0.5, 0, float32(z)+0.5)
	for _, v := range g.Villages {
		if d := WrapDelta(v.Centre, p); d.X*d.X+d.Z*d.Z < 24*24 {
			return true
		}
	}
	for _, f := range g.Flags {
		if d := WrapDelta(f.Pos, p); d.X*d.X+d.Z*d.Z < 6*6 {
			return true
		}
	}
	if b := w.Beacon; b.Y > 0 {
		if d := WrapDelta(b, p); d.X*d.X+d.Z*d.Z < 12*12 {
			return true
		}
	}
	ground := w.SurfaceY(x, z)
	for y := max(1, ground-quakeDepth-1); y < worldH; y++ {
		if !naturalBlock(w.Get(x, y, z)) {
			return true
		}
	}
	return false
}

// naturalBlock reports whether a block is part of the landscape rather than
// something built or placed.
func naturalBlock(b Block) bool {
	switch b {
	case Planks, StoneBrick, Glass, Torch, TNT, Wool, Bed, Ladder, DoorClosed, DoorOpen,
		Beacon, BeaconLit, FlagPost, ChestBlock, Spawner, Crate, Bedrock, Cobble, GoldBlock,
		WheatGrowing, Wheat:
		return false
	}
	return true
}

// shiftColumn moves the top quakeDepth blocks of a column up or down one.
func (g *Game) shiftColumn(x, z, ground, dir int) {
	w := g.World
	top := ground
	for y := ground; y < worldH; y++ { // include anything standing on it (trees, grass)
		if w.Get(x, y, z) != Air {
			top = y + 1
		}
	}
	y0 := ground - quakeDepth
	if dir > 0 {
		if top >= worldH-1 {
			return
		}
		for y := top; y >= y0; y-- {
			w.Set(x, y+1, z, w.Get(x, y, z))
		}
		w.Set(x, y0, z, w.Get(x, y0-1, z))
	} else {
		for y := y0; y <= top; y++ {
			w.Set(x, y-1, z, w.Get(x, y, z))
		}
		w.Set(x, top, z, Air)
	}
}

// quakeFissure opens a crack a few blocks deep along the fault.
func (g *Game) quakeFissure(x, z, ground int) {
	w := g.World
	depth := 3 + rand.Intn(4)
	for y := ground + 3; y >= max(2, ground-depth); y-- {
		if b := w.Get(x, y, z); b != Air && b != Water && b != Lava {
			w.Set(x, y, z, Air)
		}
	}
	if rand.Float32() < 0.08 {
		w.Set(x, max(2, ground-depth-1), z, Lava)
	}
}

// unburyAll lifts anything the uplift just enclosed back onto the surface.
func (g *Game) unburyAll(centre rl.Vector3, r float32) {
	if !g.Headless {
		g.unbury(&g.Player.Pos, centre, r)
	}
	for _, a := range g.Animals {
		if a.Alive {
			g.unbury(&a.Pos, centre, r)
		}
	}
	for _, e := range g.Enemies {
		if e.Alive {
			g.unbury(&e.Pos, centre, r)
		}
	}
}

func (g *Game) unbury(pos *rl.Vector3, centre rl.Vector3, r float32) {
	if WrapDist(*pos, centre) > r {
		return
	}
	x, z := floorI(pos.X), floorI(pos.Z)
	for i := 0; i < 4; i++ {
		y := floorI(pos.Y + 0.1)
		if !blocks[g.World.Get(x, y, z)].Solid && !blocks[g.World.Get(x, y+1, z)].Solid {
			return
		}
		pos.Y = float32(y + 1)
	}
}

// drawFaultMarkers shows the active epicentre and past fault lines on a map.
func (g *Game) drawFaultMarkers(toMap func(x, z float32) (float32, float32), scale float32, labels bool) {
	yellow := rl.NewColor(255, 230, 90, 255)
	for _, f := range g.Faults {
		dx, dz := float32(math.Cos(float64(f.Dir)))*quakeRadius*0.85, float32(math.Sin(float64(f.Dir)))*quakeRadius*0.85
		x1, y1 := toMap(f.Pos.X-dx, f.Pos.Z-dz)
		x2, y2 := toMap(f.Pos.X+dx, f.Pos.Z+dz)
		if abs32(x2-x1) > 200*scale || abs32(y2-y1) > 200*scale {
			continue // crosses the wrap seam
		}
		rl.DrawLineEx(rl.NewVector2(x1, y1), rl.NewVector2(x2, y2), 1.5*scale, yellow)
		if labels {
			x, y := toMap(f.Pos.X, f.Pos.Z)
			rl.DrawText(fmt.Sprintf("fault day %d", f.Day), int32(x)+8, int32(y)+4, 16, yellow)
		}
	}
	if g.Quake.Active {
		x, y := toMap(g.Quake.Epi.X, g.Quake.Epi.Z)
		pulse := float32(math.Sin(rl.GetTime()*10))*0.5 + 0.5
		rl.DrawCircleLines(int32(x), int32(y), (4+4*pulse)*scale, yellow)
		if labels {
			rl.DrawText("EARTHQUAKE", int32(x)+10, int32(y)-8, 16, yellow)
		}
	}
}
