package main

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// The world is a fixed voxel volume. Block coordinates run from originX/originZ
// (inclusive) to originX+worldW / originZ+worldD (exclusive) on X/Z and 0..worldH on Y.
const (
	worldW    = 64
	worldH    = 32
	worldD    = 64
	chunkSize = 16
	originX   = -worldW / 2
	originZ   = -worldD / 2
)

type Block uint8

const (
	Air Block = iota
	Grass
	Dirt
	Stone
	Sand
	Wood
	Leaves
	Planks
	Brick
	Bedrock
	numBlocks
)

type blockInfo struct {
	Name              string
	Top, Side, Bottom rl.Color
	MineTime          float32 // seconds to break; < 0 means unbreakable
	Drops             Block
}

func col(r, g, b uint8) rl.Color { return rl.NewColor(r, g, b, 255) }

var blocks = [numBlocks]blockInfo{
	Air:     {Name: "Air", MineTime: -1},
	Grass:   {"Grass", col(106, 170, 64), col(118, 96, 60), col(118, 96, 60), 0.35, Dirt},
	Dirt:    {"Dirt", col(122, 92, 58), col(122, 92, 58), col(122, 92, 58), 0.3, Dirt},
	Stone:   {"Stone", col(128, 128, 132), col(122, 122, 126), col(115, 115, 120), 0.9, Stone},
	Sand:    {"Sand", col(222, 208, 150), col(214, 200, 142), col(205, 190, 135), 0.3, Sand},
	Wood:    {"Wood", col(160, 130, 78), col(102, 76, 44), col(160, 130, 78), 0.6, Wood},
	Leaves:  {"Leaves", col(62, 138, 52), col(56, 126, 48), col(50, 110, 42), 0.2, Leaves},
	Planks:  {"Planks", col(184, 146, 86), col(176, 138, 80), col(168, 130, 74), 0.5, Planks},
	Brick:   {"Brick", col(160, 84, 72), col(152, 78, 66), col(140, 70, 60), 1.0, Brick},
	Bedrock: {"Bedrock", col(40, 40, 46), col(40, 40, 46), col(40, 40, 46), -1, Bedrock},
}

type chunk struct {
	mesh   rl.Mesh
	loaded bool
	dirty  bool
	verts  []float32
	norms  []float32
	cols   []uint8
}

// World holds the voxel volume, per-column surface heights and chunk meshes.
type World struct {
	Blocks  []Block
	Height  []int // per column (z*worldW+x): feet level of the surface, i.e. highest solid + 1
	Version int   // bumped on every block change; the nav grid watches it
	Half    float32
	chunks  []*chunk
	ncx     int
	ncz     int
	mat     rl.Material
	matOK   bool
}

func NewWorld() *World {
	w := &World{
		Blocks: make([]Block, worldW*worldH*worldD),
		Height: make([]int, worldW*worldD),
		Half:   worldW / 2,
		ncx:    worldW / chunkSize,
		ncz:    worldD / chunkSize,
	}
	for i := 0; i < w.ncx*w.ncz; i++ {
		w.chunks = append(w.chunks, &chunk{dirty: true})
	}
	w.generate(rand.Int())
	for z := 0; z < worldD; z++ {
		for x := 0; x < worldW; x++ {
			w.recomputeHeight(x, z)
		}
	}
	return w
}

// ---------- generation ----------

func hash2(x, y, seed int) float32 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + uint32(seed)*1274126177
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float32(h&0xffff) / 65535
}

// vnoise is smooth value noise in [0,1].
func vnoise(x, y float32, seed int) float32 {
	xi, yi := int(math.Floor(float64(x))), int(math.Floor(float64(y)))
	fx, fy := x-float32(xi), y-float32(yi)
	fx, fy = fx*fx*(3-2*fx), fy*fy*(3-2*fy)
	a, b := hash2(xi, yi, seed), hash2(xi+1, yi, seed)
	c, d := hash2(xi, yi+1, seed), hash2(xi+1, yi+1, seed)
	return lerp(lerp(a, b, fx), lerp(c, d, fx), fy)
}

func (w *World) generate(seed int) {
	// Terrain.
	for z := 0; z < worldD; z++ {
		for x := 0; x < worldW; x++ {
			fx, fz := float32(x), float32(z)
			n := 0.55*vnoise(fx/16, fz/16, seed) + 0.3*vnoise(fx/7+50, fz/7+50, seed+1) + 0.15*vnoise(fx/3+90, fz/3+90, seed+2)
			h := 6 + int(n*13)
			// Flatten the middle so the player spawn is open.
			dx, dz := float32(x+originX), float32(z+originZ)
			if d := float32(math.Sqrt(float64(dx*dx + dz*dz))); d < 9 {
				t := clamp(d/9, 0, 1)
				h = int(lerp(10, float32(h), t*t))
			}
			for y := 0; y < h; y++ {
				b := Stone
				switch {
				case y == 0:
					b = Bedrock
				case y == h-1:
					b = Grass
					if h <= 8 {
						b = Sand
					}
				case y >= h-4:
					b = Dirt
					if h <= 8 {
						b = Sand
					}
				}
				w.setLocal(x, y, z, b)
			}
		}
	}
	// Trees.
	for i := 0; i < 70; i++ {
		x, z := rand.Intn(worldW-4)+2, rand.Intn(worldD-4)+2
		if dx, dz := x+originX, z+originZ; dx*dx+dz*dz < 64 {
			continue
		}
		h := w.surfaceLocal(x, z)
		if w.getLocal(x, h-1, z) != Grass {
			continue
		}
		th := 4 + rand.Intn(3)
		for y := h; y < h+th; y++ {
			w.setLocal(x, y, z, Wood)
		}
		top := h + th - 1
		for dy := -2; dy <= 1; dy++ {
			r := 2
			if dy == 1 {
				r = 1
			}
			for dz := -r; dz <= r; dz++ {
				for dx := -r; dx <= r; dx++ {
					if abs(dx) == 2 && abs(dz) == 2 {
						continue
					}
					if w.getLocal(x+dx, top+dy, z+dz) == Air {
						w.setLocal(x+dx, top+dy, z+dz, Leaves)
					}
				}
			}
		}
	}
	// Brick ruins for cover.
	for _, c := range [][2]int{{-18, -18}, {18, -18}, {-18, 18}, {18, 18}, {0, -22}, {0, 22}, {-24, 0}, {24, 0}} {
		x, z := c[0]-originX, c[1]-originZ
		h := w.surfaceLocal(x, z)
		for dz := -2; dz <= 2; dz++ {
			for dx := -2; dx <= 2; dx++ {
				edge := abs(dx) == 2 || abs(dz) == 2
				if !edge {
					continue
				}
				// Leave door gaps in the middle of two sides.
				if (dx == 0 && dz == 2) || (dz == 0 && dx == -2) {
					continue
				}
				for y := h - 1; y < h+3; y++ {
					if y >= 0 {
						w.setLocal(x+dx, y, z+dz, Brick)
					}
				}
			}
		}
		// Floor under the ruin so it doesn't float on slopes.
		for dz := -2; dz <= 2; dz++ {
			for dx := -2; dx <= 2; dx++ {
				for y := 0; y < h; y++ {
					if w.getLocal(x+dx, y, z+dz) == Air {
						w.setLocal(x+dx, y, z+dz, Stone)
					}
				}
				for y := h; y < h+3; y++ {
					if abs(dx) < 2 && abs(dz) < 2 {
						w.setLocal(x+dx, y, z+dz, Air)
					}
				}
			}
		}
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// ---------- block access ----------

func inLocal(x, y, z int) bool {
	return x >= 0 && z >= 0 && y >= 0 && x < worldW && z < worldD && y < worldH
}

func (w *World) getLocal(x, y, z int) Block {
	if !inLocal(x, y, z) {
		return Air
	}
	return w.Blocks[(y*worldD+z)*worldW+x]
}

func (w *World) setLocal(x, y, z int, b Block) {
	if inLocal(x, y, z) {
		w.Blocks[(y*worldD+z)*worldW+x] = b
	}
}

func (w *World) surfaceLocal(x, z int) int {
	for y := worldH - 1; y >= 0; y-- {
		if w.getLocal(x, y, z) != Air {
			return y + 1
		}
	}
	return 0
}

func (w *World) recomputeHeight(x, z int) {
	w.Height[z*worldW+x] = w.surfaceLocal(x, z)
}

// Get returns the block at world coordinates (Air outside the volume).
func (w *World) Get(x, y, z int) Block { return w.getLocal(x-originX, y, z-originZ) }

// InBounds reports whether world block coordinates are inside the volume.
func (w *World) InBounds(x, y, z int) bool { return inLocal(x-originX, y, z-originZ) }

// Solid is the collision query: everything below y=0 and outside X/Z bounds is a wall.
func (w *World) Solid(x, y, z int) bool {
	if y < 0 {
		return true
	}
	if y >= worldH {
		return false
	}
	lx, lz := x-originX, z-originZ
	if lx < 0 || lz < 0 || lx >= worldW || lz >= worldD {
		return true
	}
	return w.Blocks[(y*worldD+lz)*worldW+lx] != Air
}

// Set changes a block, updates the column height and marks chunk meshes dirty.
func (w *World) Set(x, y, z int, b Block) {
	lx, lz := x-originX, z-originZ
	if !inLocal(lx, y, lz) {
		return
	}
	w.Blocks[(y*worldD+lz)*worldW+lx] = b
	w.recomputeHeight(lx, lz)
	w.Version++
	cx, cz := lx/chunkSize, lz/chunkSize
	mark := func(i, j int) {
		if i >= 0 && j >= 0 && i < w.ncx && j < w.ncz {
			w.chunks[j*w.ncx+i].dirty = true
		}
	}
	mark(cx, cz)
	if lx%chunkSize == 0 {
		mark(cx-1, cz)
	}
	if lx%chunkSize == chunkSize-1 {
		mark(cx+1, cz)
	}
	if lz%chunkSize == 0 {
		mark(cx, cz-1)
	}
	if lz%chunkSize == chunkSize-1 {
		mark(cx, cz+1)
	}
}

// SurfaceY returns the feet level of the surface at a world column.
func (w *World) SurfaceY(x, z int) int {
	lx, lz := x-originX, z-originZ
	if lx < 0 || lz < 0 || lx >= worldW || lz >= worldD {
		return 0
	}
	return w.Height[lz*worldW+lx]
}

func floorI(v float32) int { return int(math.Floor(float64(v))) }

// SpawnPoint is the player start, on the surface at the world centre.
func (w *World) SpawnPoint() rl.Vector3 {
	return rl.NewVector3(0.5, float32(w.SurfaceY(0, 0)), 0.5)
}

// ---------- rendering ----------

type faceDef struct {
	dx, dy, dz int
	corners    [4][3]float32
	shade      float32
}

var faces = [6]faceDef{
	{0, 1, 0, [4][3]float32{{0, 1, 0}, {0, 1, 1}, {1, 1, 1}, {1, 1, 0}}, 1.0},
	{0, -1, 0, [4][3]float32{{0, 0, 0}, {1, 0, 0}, {1, 0, 1}, {0, 0, 1}}, 0.45},
	{1, 0, 0, [4][3]float32{{1, 0, 0}, {1, 1, 0}, {1, 1, 1}, {1, 0, 1}}, 0.82},
	{-1, 0, 0, [4][3]float32{{0, 0, 0}, {0, 0, 1}, {0, 1, 1}, {0, 1, 0}}, 0.76},
	{0, 0, 1, [4][3]float32{{0, 0, 1}, {1, 0, 1}, {1, 1, 1}, {0, 1, 1}}, 0.68},
	{0, 0, -1, [4][3]float32{{0, 0, 0}, {0, 1, 0}, {1, 1, 0}, {1, 0, 0}}, 0.62},
}

func (w *World) buildChunk(ci, cj int, c *chunk) {
	c.verts, c.norms, c.cols = c.verts[:0], c.norms[:0], c.cols[:0]
	for lz := cj * chunkSize; lz < (cj+1)*chunkSize; lz++ {
		for lx := ci * chunkSize; lx < (ci+1)*chunkSize; lx++ {
			top := w.Height[lz*worldW+lx]
			for y := 0; y < top; y++ {
				b := w.getLocal(lx, y, lz)
				if b == Air {
					continue
				}
				info := &blocks[b]
				// Subtle per-block tint variation gives a textured look.
				v := 0.92 + hash2(lx, y*131+lz, 7)*0.12
				for _, f := range faces {
					if w.getLocal(lx+f.dx, y+f.dy, lz+f.dz) != Air {
						continue
					}
					base := info.Side
					if f.dy > 0 {
						base = info.Top
					} else if f.dy < 0 {
						base = info.Bottom
					}
					s := f.shade * v
					r, g, bl := uint8(float32(base.R)*s), uint8(float32(base.G)*s), uint8(float32(base.B)*s)
					wx, wz := float32(lx+originX), float32(lz+originZ)
					for _, i := range [6]int{0, 1, 2, 0, 2, 3} {
						cr := f.corners[i]
						c.verts = append(c.verts, wx+cr[0], float32(y)+cr[1], wz+cr[2])
						c.norms = append(c.norms, float32(f.dx), float32(f.dy), float32(f.dz))
						c.cols = append(c.cols, r, g, bl, 255)
					}
				}
			}
		}
	}
	if c.loaded {
		rl.UnloadMesh(&c.mesh)
		c.loaded = false
	}
	c.mesh = rl.Mesh{}
	if len(c.verts) == 0 {
		return
	}
	c.mesh.VertexCount = int32(len(c.verts) / 3)
	c.mesh.TriangleCount = c.mesh.VertexCount / 3
	c.mesh.Vertices = &c.verts[0]
	c.mesh.Normals = &c.norms[0]
	c.mesh.Colors = &c.cols[0]
	rl.UploadMesh(&c.mesh, false)
	c.loaded = true
}

// Draw renders all chunks, skipping ones fully behind the camera.
func (w *World) Draw(cam rl.Camera3D) {
	if !w.matOK {
		w.mat = rl.LoadMaterialDefault()
		w.matOK = true
	}
	fwd := rl.Vector3Normalize(rl.Vector3Subtract(cam.Target, cam.Position))
	for cj := 0; cj < w.ncz; cj++ {
		for ci := 0; ci < w.ncx; ci++ {
			c := w.chunks[cj*w.ncx+ci]
			if c.dirty {
				w.buildChunk(ci, cj, c)
				c.dirty = false
			}
			if !c.loaded {
				continue
			}
			centre := rl.NewVector3(float32(ci*chunkSize+chunkSize/2+originX), worldH/2, float32(cj*chunkSize+chunkSize/2+originZ))
			if rl.Vector3DotProduct(rl.Vector3Subtract(centre, cam.Position), fwd) < -24 {
				continue
			}
			rl.DrawMesh(c.mesh, w.mat, rl.MatrixIdentity())
		}
	}
}

// ---------- collision ----------

type MoveResult struct {
	Ground, Ceiling, Wall bool
}

func (w *World) boxSolid(x0, y0, z0, x1, y1, z1 float32) bool {
	const eps = 1e-4
	for y := floorI(y0); y <= floorI(y1-eps); y++ {
		for z := floorI(z0); z <= floorI(z1-eps); z++ {
			for x := floorI(x0); x <= floorI(x1-eps); x++ {
				if w.Solid(x, y, z) {
					return true
				}
			}
		}
	}
	return false
}

// MoveBox moves an axis-aligned box (feet at pos, half width hw, height h) by
// delta, sliding along solid blocks. With stepUp, a blocked horizontal move
// may climb one block when standing on the ground.
func (w *World) MoveBox(pos rl.Vector3, hw, h float32, delta rl.Vector3, stepUp bool) (rl.Vector3, MoveResult) {
	const eps = 0.001
	var res MoveResult
	solid := func(p rl.Vector3) bool {
		return w.boxSolid(p.X-hw, p.Y, p.Z-hw, p.X+hw, p.Y+h, p.Z+hw)
	}

	// Vertical.
	oldY := pos.Y
	pos.Y += delta.Y
	if solid(pos) {
		if delta.Y < 0 {
			for y := floorI(oldY); y >= floorI(pos.Y); y-- {
				if w.boxSolid(pos.X-hw, float32(y), pos.Z-hw, pos.X+hw, float32(y+1), pos.Z+hw) {
					pos.Y = float32(y + 1)
					break
				}
			}
			res.Ground = true
		} else {
			for y := floorI(oldY + h); y <= floorI(pos.Y+h); y++ {
				if w.boxSolid(pos.X-hw, float32(y), pos.Z-hw, pos.X+hw, float32(y+1), pos.Z+hw) {
					pos.Y = float32(y) - h - eps
					break
				}
			}
			res.Ceiling = true
		}
	} else if delta.Y <= 0 {
		// Standing exactly on a surface counts as grounded.
		probe := pos
		probe.Y -= 0.01
		res.Ground = solid(probe)
	}

	// Horizontal, one axis at a time.
	moveAxis := func(dx, dz float32) {
		if dx == 0 && dz == 0 {
			return
		}
		try := pos
		try.X += dx
		try.Z += dz
		if !solid(try) {
			pos = try
			return
		}
		if stepUp && res.Ground {
			up := try
			up.Y += 1
			if !solid(up) {
				pos = up
				return
			}
		}
		res.Wall = true
		if dx > 0 {
			pos.X = float32(floorI(try.X+hw)) - hw - eps
		} else if dx < 0 {
			pos.X = float32(floorI(try.X-hw)+1) + hw + eps
		}
		if dz > 0 {
			pos.Z = float32(floorI(try.Z+hw)) - hw - eps
		} else if dz < 0 {
			pos.Z = float32(floorI(try.Z-hw)+1) + hw + eps
		}
	}
	moveAxis(delta.X, 0)
	moveAxis(0, delta.Z)
	return pos, res
}

// PointFree reports whether a box of the given half width standing at pos is clear of blocks.
func (w *World) PointFree(pos rl.Vector3, radius float32) bool {
	return !w.boxSolid(pos.X-radius, pos.Y, pos.Z-radius, pos.X+radius, pos.Y+1.8, pos.Z+radius)
}

// ---------- ray queries ----------

// RayHit describes the first solid block along a ray and the face it entered through.
type RayHit struct {
	Hit        bool
	Dist       float32
	X, Y, Z    int // block
	NX, NY, NZ int // face normal (points to the adjacent air cell)
}

// RayCast walks the voxel grid (Amanatides & Woo) up to maxD units.
func (w *World) RayCast(o, d rl.Vector3, maxD float32) RayHit {
	d = rl.Vector3Normalize(d)
	x, y, z := floorI(o.X), floorI(o.Y), floorI(o.Z)
	step := [3]int{sign(d.X), sign(d.Y), sign(d.Z)}
	dir := [3]float32{d.X, d.Y, d.Z}
	orig := [3]float32{o.X, o.Y, o.Z}
	cell := [3]int{x, y, z}
	var tMax, tDelta [3]float32
	for i := 0; i < 3; i++ {
		if dir[i] == 0 {
			tMax[i] = float32(math.Inf(1))
			tDelta[i] = float32(math.Inf(1))
			continue
		}
		tDelta[i] = float32(math.Abs(float64(1 / dir[i])))
		var next float32
		if step[i] > 0 {
			next = float32(cell[i]+1) - orig[i]
		} else {
			next = orig[i] - float32(cell[i])
		}
		tMax[i] = next * tDelta[i]
	}
	var normal [3]int
	t := float32(0)
	for t <= maxD {
		if cell[1] < 0 || cell[1] >= worldH && step[1] >= 0 {
			break
		}
		if w.Solid(cell[0], cell[1], cell[2]) {
			return RayHit{true, t, cell[0], cell[1], cell[2], normal[0], normal[1], normal[2]}
		}
		axis := 0
		if tMax[1] < tMax[axis] {
			axis = 1
		}
		if tMax[2] < tMax[axis] {
			axis = 2
		}
		t = tMax[axis]
		tMax[axis] += tDelta[axis]
		cell[axis] += step[axis]
		normal = [3]int{}
		normal[axis] = -step[axis]
	}
	return RayHit{}
}

func sign(v float32) int {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}

// RayDistance returns the distance to the first block hit, or +Inf.
func (w *World) RayDistance(ray rl.Ray) float32 {
	if h := w.RayCast(ray.Position, ray.Direction, 200); h.Hit {
		return h.Dist
	}
	return float32(math.Inf(1))
}

// RandomFreePoint finds a surface spawn point at least minDist from `from`.
func (w *World) RandomFreePoint(from rl.Vector3, minDist float32) rl.Vector3 {
	for i := 0; i < 200; i++ {
		lx, lz := rand.Intn(worldW-4)+2, rand.Intn(worldD-4)+2
		h := w.Height[lz*worldW+lx]
		p := rl.NewVector3(float32(lx+originX)+0.5, float32(h), float32(lz+originZ)+0.5)
		if h+2 < worldH && w.getLocal(lx, h+1, lz) == Air && w.PointFree(p, 0.5) &&
			rl.Vector3Distance(rl.NewVector3(p.X, 0, p.Z), rl.NewVector3(from.X, 0, from.Z)) >= minDist {
			return p
		}
	}
	return rl.NewVector3(0.5, float32(w.SurfaceY(0, 0)), float32(originZ+3)+0.5)
}

func clamp(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
