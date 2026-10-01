package main

import (
	"bytes"
	"encoding/gob"
	"math/rand"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Messages survive a gob round trip and the world compresses and restores exactly.
func TestNetCodec(t *testing.T) {
	rand.Seed(11)
	w := NewWorld()
	data := compressWorld(w.Blocks)
	if len(data) > 200000 {
		t.Fatalf("world payload too large: %d bytes", len(data))
	}
	back, err := decompressWorld(data)
	if err != nil || len(back) != len(w.Blocks) {
		t.Fatalf("decompress: %v", err)
	}
	for i := range back {
		if back[i] != w.Blocks[i] {
			t.Fatalf("block %d differs", i)
		}
	}
	snap := &Snapshot{SkyT: 0.4, Day: 2, Players: []PlayerState{{ID: 3, Name: "Ann", Pos: rl.NewVector3(1, 2, 3), Held: Item{ItemBlock, Torch}}},
		Enemies: []EnemyState{{ID: 9, Kind: KindCreeper, Alive: true, Fuse: 0.5}},
		Drops:   []DropState{{ID: 4, Block: Cobble, Count: 3}}}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	for _, m := range []Msg{{Snapshot: snap}, {Block: &struct {
		X, Y, Z int
		B       Block
	}{1, 2, 3, Glass}}, {Hit: &struct {
		Enemy    uint32
		Animal   uint32
		Dmg      int
		Headshot bool
		Knock    rl.Vector3
	}{Enemy: 9, Dmg: 3, Headshot: true}}} {
		if err := enc.Encode(&m); err != nil {
			t.Fatal(err)
		}
	}
	dec := gob.NewDecoder(&buf)
	var a, b, c Msg
	for _, m := range []*Msg{&a, &b, &c} {
		if err := dec.Decode(m); err != nil {
			t.Fatal(err)
		}
	}
	if a.Snapshot == nil || a.Snapshot.Players[0].Name != "Ann" || a.Snapshot.Enemies[0].Kind != KindCreeper || a.Snapshot.Drops[0].Count != 3 {
		t.Fatalf("snapshot lost data: %+v", a.Snapshot)
	}
	if b.Block == nil || b.Block.B != Glass || b.Snapshot != nil {
		t.Fatalf("block message wrong: %+v", b)
	}
	if c.Hit == nil || c.Hit.Enemy != 9 || !c.Hit.Headshot {
		t.Fatalf("hit message wrong: %+v", c)
	}
}

// Hostiles chase the nearest of several targets; the nav field seeds from all of them.
func TestMultipleTargets(t *testing.T) {
	rand.Seed(12)
	g := &Game{Audio: &Audio{}, CraftHover: -1, Remotes: map[uint32]*RemotePlayer{}}
	g.Reset()
	g.Player.Pos = rl.NewVector3(0.5, 14, 0.5)
	g.Remotes[7] = &RemotePlayer{PlayerState{ID: 7, Name: "Bob", Pos: rl.NewVector3(20.5, 14, 0.5), HP: 50}}
	if n := len(g.targets()); n != 2 {
		t.Fatalf("targets %d", n)
	}
	if g.nearestTarget(rl.NewVector3(18, 14, 0)).ID != 7 {
		t.Fatal("nearest target should be the remote player")
	}
	if g.nearestTarget(rl.NewVector3(2, 14, 0)).ID != 0 {
		t.Fatal("nearest target should be the local player")
	}
	seeds := []rl.Vector3{g.Player.Pos, g.Remotes[7].Pos}
	g.Nav.Update(seeds, g.World)
	x, z := g.Nav.cellOf(g.Remotes[7].Pos)
	if g.Nav.Dist[z*g.Nav.N+x] != 0 {
		t.Fatal("remote player's cell should be a seed")
	}
	g.Remotes[7].HP = 0
	if n := len(g.targets()); n != 1 {
		t.Fatalf("dead players are not targets: %d", n)
	}
}
