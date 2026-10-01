package main

import (
	"math/rand"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Most of the surface must be reachable from the spawn, and following the
// field must converge on the goal without ever climbing more than one block.
func TestNavFieldReachesSpawn(t *testing.T) {
	rand.Seed(1)
	w := NewWorld()
	nav := NewNavGrid(w)
	spawn := w.SpawnPoint()
	nav.Update(spawn, w)
	walkable, reachable := 0, 0
	for i, ok := range nav.Walk {
		if ok {
			walkable++
			if nav.Dist[i] >= 0 {
				reachable++
			}
		}
	}
	if walkable == 0 || float64(reachable)/float64(walkable) < 0.8 {
		t.Fatalf("only %d of %d walkable cells reachable", reachable, walkable)
	}
	// Walk from a far reachable cell.
	var start rl.Vector3
	found := false
	for i := 0; i < len(nav.Dist) && !found; i++ {
		if nav.Dist[i] > 40 {
			start = nav.center(i%nav.N, i/nav.N)
			found = true
		}
	}
	if !found {
		t.Skip("no far cell")
	}
	pos := start
	prevH := nav.Height[func() int { x, z := nav.cellOf(pos); return z*nav.N + x }()]
	for step := 0; step < 1000; step++ {
		d, ok := nav.Dir(pos)
		if !ok {
			break
		}
		pos = rl.Vector3Add(pos, rl.Vector3Scale(d, 0.4))
		x, z := nav.cellOf(pos)
		h := nav.Height[z*nav.N+x]
		if h-prevH > navStepUp {
			t.Fatalf("climbed %d blocks at step %d", h-prevH, step)
		}
		prevH = h
	}
	if rl.Vector3Distance(rl.NewVector3(pos.X, 0, pos.Z), rl.NewVector3(spawn.X, 0, spawn.Z)) > 1.5 {
		t.Fatalf("did not reach goal, ended at %v (started %v)", pos, start)
	}
}

// Placing a block bumps the world version and the field picks up the change.
func TestNavRefreshOnEdit(t *testing.T) {
	rand.Seed(2)
	w := NewWorld()
	nav := NewNavGrid(w)
	nav.Update(w.SpawnPoint(), w)
	x, z := 3, 3
	h := w.SurfaceY(x, z)
	// Level the neighbour column to the same height, then raise a 2-block pillar.
	for y := 0; y < worldH; y++ {
		b := Air
		if y < h {
			b = Stone
		}
		w.Set(x+1, y, z, b)
	}
	for y := h; y < h+2; y++ {
		w.Set(x, y, z, StoneBrick)
	}
	nav.Update(w.SpawnPoint(), w)
	lx, lz := x-originX, z-originZ
	if nav.Height[lz*nav.N+lx] != h+2 {
		t.Fatalf("nav height not refreshed: got %d want %d", nav.Height[lz*nav.N+lx], h+2)
	}
	if nav.canStep(lx+1, lz, lx, lz) {
		t.Fatal("enemies must not step up a 2-block pillar")
	}
}

// The DDA ray must report the block and entry face correctly.
func TestRayCastFace(t *testing.T) {
	w := NewWorld()
	// Clear a corridor and place a known block at its end.
	for z := 0; z <= 5; z++ {
		w.Set(5, 20, z, Air)
	}
	w.Set(5, 20, 5, Stone)
	h := w.RayCast(rl.NewVector3(5.5, 20.5, 0.5), rl.NewVector3(0, 0, 1), 20)
	if !h.Hit || h.X != 5 || h.Y != 20 || h.Z != 5 || h.NZ != -1 {
		t.Fatalf("unexpected hit %+v", h)
	}
	if h.Dist < 4.4 || h.Dist > 4.6 {
		t.Fatalf("bad distance %v", h.Dist)
	}
}

// A box dropped onto the terrain lands on the surface and reports ground.
func TestMoveBoxLands(t *testing.T) {
	w := NewWorld()
	s := w.SpawnPoint()
	pos := rl.NewVector3(s.X, s.Y+3, s.Z)
	var res MoveResult
	for i := 0; i < 20; i++ {
		pos, res = w.MoveBox(pos, 0.3, 1.8, rl.NewVector3(0, -0.5, 0), false)
	}
	if !res.Ground || pos.Y != s.Y {
		t.Fatalf("expected to land at %v, got %v ground=%v", s.Y, pos.Y, res.Ground)
	}
}
