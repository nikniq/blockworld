package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	navStepUp   = 1 // blocks an enemy can climb in one step
	navDropDown = 3 // blocks an enemy will willingly drop
)

// NavGrid is a flow field over the terrain surface (highest solid block per column): every walkable column
// stores its walking distance to the player, so enemies route over hills and
// around trees and ruins by stepping "downhill". It is rebuilt when the player
// changes column or the world changes.
type NavGrid struct {
	N       int
	Origin  float32
	Walk    []bool
	Height  []int
	Dist    []int32
	queue   []int32
	seeds   []int32 // seed cells of the last build
	valid   bool
	version int
}

func NewNavGrid(w *World) *NavGrid {
	n := worldW
	g := &NavGrid{
		N:      n,
		Origin: originX,
		Walk:   make([]bool, n*n),
		Height: make([]int, n*n),
		Dist:   make([]int32, n*n),
		queue:  make([]int32, 0, n*n),
	}
	g.refresh(w)
	return g
}

func (g *NavGrid) refresh(w *World) {
	g.version = w.Version
	for z := 0; z < g.N; z++ {
		for x := 0; x < g.N; x++ {
			h := w.Ground[z*worldW+x]
			g.Height[z*g.N+x] = h
			// Needs two blocks of headroom above the ground (water is wadeable).
			g.Walk[z*g.N+x] = h+1 < worldH && !blocks[w.getLocal(x, h, z)].Solid && !blocks[w.getLocal(x, h+1, z)].Solid
		}
	}
	g.valid = false
}

func (g *NavGrid) cellOf(p rl.Vector3) (int, int) {
	return wrapI(floorI(p.X-g.Origin), g.N), wrapI(floorI(p.Z-g.Origin), g.N)
}

func (g *NavGrid) center(x, z int) rl.Vector3 {
	return rl.NewVector3(g.Origin+float32(x)+0.5, 0, g.Origin+float32(z)+0.5)
}

func (g *NavGrid) walkable(x, z int) bool {
	x, z = wrapI(x, g.N), wrapI(z, g.N)
	return g.Walk[z*g.N+x]
}

// canStep reports whether an agent may move from cell a to adjacent cell b.
func (g *NavGrid) canStep(ax, az, bx, bz int) bool {
	if !g.walkable(ax, az) || !g.walkable(bx, bz) {
		return false
	}
	ax, az, bx, bz = wrapI(ax, g.N), wrapI(az, g.N), wrapI(bx, g.N), wrapI(bz, g.N)
	dh := g.Height[bz*g.N+bx] - g.Height[az*g.N+ax]
	return dh <= navStepUp && -dh <= navDropDown
}

// Update rebuilds the field if the world changed or any target moved to
// another cell. With several targets the field leads to the nearest one.
func (g *NavGrid) Update(targets []rl.Vector3, w *World) {
	if w.Version != g.version {
		g.refresh(w)
	}
	cells := make([]int32, 0, len(targets))
	for _, t := range targets {
		tx, tz := g.cellOf(t)
		cells = append(cells, int32(tz*g.N+tx))
	}
	same := g.valid && len(cells) == len(g.seeds)
	for i := 0; same && i < len(cells); i++ {
		same = cells[i] == g.seeds[i]
	}
	if same {
		return
	}
	g.seeds, g.valid = cells, true
	for i := range g.Dist {
		g.Dist[i] = -1
	}
	g.queue = g.queue[:0]
	seed := func(x, z int) {
		x, z = wrapI(x, g.N), wrapI(z, g.N)
		if g.walkable(x, z) && g.Dist[z*g.N+x] < 0 {
			g.Dist[z*g.N+x] = 0
			g.queue = append(g.queue, int32(z*g.N+x))
		}
	}
	for _, c := range cells {
		tx, tz := int(c)%g.N, int(c)/g.N
		if g.walkable(tx, tz) {
			seed(tx, tz)
			continue
		}
		// Target is somewhere unwalkable (inside a tree canopy, on a pillar):
		// seed the nearest ring of walkable cells around it.
		before := len(g.queue)
		for r := 1; r < g.N && len(g.queue) == before; r++ {
			for dz := -r; dz <= r; dz++ {
				for dx := -r; dx <= r; dx++ {
					seed(tx+dx, tz+dz)
				}
			}
		}
	}
	// Breadth-first search outward; an edge A->B exists when B can step to A.
	for head := 0; head < len(g.queue); head++ {
		i := int(g.queue[head])
		x, z := i%g.N, i/g.N
		d := g.Dist[i] + 1
		for _, n := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, nz := wrapI(x+n[0], g.N), wrapI(z+n[1], g.N)
			if g.canStep(nx, nz, x, z) && g.Dist[nz*g.N+nx] < 0 {
				g.Dist[nz*g.N+nx] = d
				g.queue = append(g.queue, int32(nz*g.N+nx))
			}
		}
	}
}

// Dir returns the flat unit direction an agent at pos should walk to follow
// the field toward the target. ok is false when pos is already at the goal
// or the goal is unreachable.
func (g *NavGrid) Dir(pos rl.Vector3) (rl.Vector3, bool) {
	if !g.valid {
		return rl.Vector3{}, false
	}
	x, z := g.cellOf(pos)
	cur := g.Dist[z*g.N+x]
	best := cur
	bx, bz := x, z
	for dz := -1; dz <= 1; dz++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dz == 0 {
				continue
			}
			nx, nz := wrapI(x+dx, g.N), wrapI(z+dz, g.N)
			if !g.walkable(nx, nz) {
				continue
			}
			if g.walkable(x, z) && !g.canStep(x, z, nx, nz) {
				continue
			}
			// Do not cut corners: both orthogonal cells must be steppable too.
			if dx != 0 && dz != 0 && (!g.canStep(x, z, x+dx, z) || !g.canStep(x, z, x, z+dz)) {
				continue
			}
			d := g.Dist[nz*g.N+nx]
			if d < 0 {
				continue
			}
			if best < 0 || d < best {
				best = d
				bx, bz = nx, nz
			}
		}
	}
	if bx == x && bz == z {
		return rl.Vector3{}, false
	}
	to := WrapDelta(g.center(bx, bz), pos)
	to.Y = 0
	l := rl.Vector3Length(to)
	if l < 1e-4 {
		return rl.Vector3{}, false
	}
	return rl.Vector3Scale(to, 1/l), true
}
