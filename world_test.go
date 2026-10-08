package main

import (
	"math/rand"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// A generated world must contain a sea, caves and every ore, with bedrock at the bottom.
func TestGenerateFeatures(t *testing.T) {
	rand.Seed(3)
	w := NewWorld()
	counts := map[Block]int{}
	for _, b := range w.Blocks {
		counts[b]++
	}
	for _, b := range []Block{Water, CoalOre, IronOre, GoldOre, DiamondOre, Log, Leaves, Sand, StoneBrick} {
		if counts[b] == 0 {
			t.Errorf("no %s generated", blocks[b].Name)
		}
	}
	for z := 0; z < worldD; z++ {
		for x := 0; x < worldW; x++ {
			if w.getLocal(x, 0, z) != Bedrock {
				t.Fatalf("no bedrock at %d,%d", x, z)
			}
		}
	}
	// Caves: air pockets well below the ground.
	caves := 0
	for z := 0; z < worldD; z++ {
		for x := 0; x < worldW; x++ {
			g := w.Ground[z*worldW+x]
			for y := 2; y < g-3; y++ {
				if w.getLocal(x, y, z) == Air {
					caves++
				}
			}
		}
	}
	if caves < 500*areaScale {
		t.Errorf("only %d underground air cells; caves missing", caves)
	}
	// The spawn is dry and on solid ground.
	s := w.SpawnPoint()
	if w.WaterAt(s) || !w.Solid(floorI(s.X), floorI(s.Y)-1, floorI(s.Z)) {
		t.Fatalf("bad spawn %v", s)
	}
}

// Hostile spawn points are never under water or under a tree canopy.
func TestRandomFreePointDry(t *testing.T) {
	rand.Seed(4)
	w := NewWorld()
	for i := 0; i < 100; i++ {
		p := w.RandomFreePoint(w.SpawnPoint(), 10)
		if w.WaterAt(p) || w.WaterAt(rl.NewVector3(p.X, p.Y-0.5, p.Z)) {
			t.Fatalf("spawn point %v is in water", p)
		}
		if !w.PointFree(p, 0.4) {
			t.Fatalf("spawn point %v is inside blocks", p)
		}
	}
}

// Water is not solid, so an entity sinks into it; the ground height ignores it.
func TestWaterNotSolid(t *testing.T) {
	w := NewWorld()
	x, z := 40, 40
	for y := 0; y < worldH; y++ {
		b := Air
		switch {
		case y == 0:
			b = Bedrock
		case y < 5:
			b = Stone
		case y <= seaLevel:
			b = Water
		}
		w.Set(x, y, z, b)
	}
	if w.Solid(x, 7, z) || !w.IsWater(x, 7, z) {
		t.Fatal("water must not block movement")
	}
	if w.SurfaceY(x, z) != 5 {
		t.Fatalf("ground height %d, want 5", w.SurfaceY(x, z))
	}
	lx, lz := x-originX, z-originZ
	if w.Height[lz*worldW+lx] != seaLevel+1 {
		t.Fatalf("column top %d, want %d", w.Height[lz*worldW+lx], seaLevel+1)
	}
}

// Hard blocks need the right pickaxe tier and get faster with better tools.
func TestMineTimeTiers(t *testing.T) {
	p := NewPlayer(rl.Vector3{})
	p.Held = Item{Kind: ItemPickaxe}
	p.PickTier = TierWood
	wood := p.MineTime(Stone)
	if wood <= 0 || wood >= blocks[Stone].MineTime {
		t.Fatalf("wooden pickaxe should speed up stone: %v", wood)
	}
	if p.MineTime(DiamondOre) >= 0 {
		t.Fatal("diamond ore must need an iron pickaxe")
	}
	p.PickTier = TierIron
	if p.MineTime(DiamondOre) < 0 {
		t.Fatal("iron pickaxe must harvest diamond ore")
	}
	p.Held = Item{ItemBlock, Planks}
	if p.MineTime(Stone) != blocks[Stone].MineTime {
		t.Fatal("holding a block mines at hand speed")
	}
	if p.MineTime(Dirt) != blocks[Dirt].MineTime || p.MineTime(Bedrock) >= 0 {
		t.Fatal("soft blocks ignore tools; bedrock is unbreakable")
	}
}

// Crafting consumes ingredients, upgrades tools once, and the hotbar follows the inventory.
func TestCraftingAndHotbar(t *testing.T) {
	p := NewPlayer(rl.Vector3{})
	var pick *Recipe
	for i := range recipes {
		if recipes[i].Pick == TierStone {
			pick = &recipes[i]
		}
	}
	if pick.CanCraft(p) {
		t.Fatal("should lack cobblestone")
	}
	p.Inv[Cobble] = 3
	if !pick.CanCraft(p) {
		t.Fatal("should be craftable")
	}
	pick.Craft(p)
	if p.PickTier != TierStone || p.Inv[Cobble] != 0 || p.Inv[Planks] != 14 {
		t.Fatalf("craft result: tier %d cobble %d planks %d", p.PickTier, p.Inv[Cobble], p.Inv[Planks])
	}
	if pick.Useful(p) {
		t.Fatal("owned tools are no longer useful")
	}
	hb := p.Hotbar()
	if len(hb) != 5 || hb[3] != (Item{ItemBlock, Planks}) || hb[4] != (Item{ItemBlock, Torch}) {
		t.Fatalf("hotbar %v", hb)
	}
	p.HoldBlock(Planks)
	p.Inv[Planks] = 0
	p.EnsureHeld()
	if p.Held.Kind != ItemPickaxe {
		t.Fatal("empty stack must fall back to the pickaxe")
	}
}

// Chunk meshing produces textured, ambient-occluded faces only where blocks meet air.
func TestEmitFace(t *testing.T) {
	var m meshBuf
	m.emitFace(&faces[0], Grass, 0, 0, 0, unitBox, 1, [4]int{3, 3, 3, 3}, fullLight)
	if len(m.verts) != 18 || len(m.uvs) != 12 || len(m.cols) != 24 {
		t.Fatalf("face sizes %d %d %d", len(m.verts), len(m.uvs), len(m.cols))
	}
	for i := 1; i < 18; i += 3 {
		if m.verts[i] != 1 {
			t.Fatalf("top face vertex not at y=1: %v", m.verts)
		}
	}
	var dark meshBuf
	dark.emitFace(&faces[0], Grass, 0, 0, 0, unitBox, 1, [4]int{0, 0, 0, 0}, fullLight)
	if dark.cols[2] >= m.cols[2] {
		t.Fatal("occluded corners must be darker")
	}
	u0, v0, u1, v1 := tileUV(Grass, 0)
	if u0 >= u1 || v0 >= v1 || u1 > 1 || v1 > 1 {
		t.Fatalf("bad uv %v %v %v %v", u0, v0, u1, v1)
	}
}

// The texture atlas has an opaque tile for every solid block and translucent water.
func TestAtlas(t *testing.T) {
	img := buildAtlas()
	for b := Block(1); b < numBlocks; b++ {
		r := TileRect(b, 1)
		c := img.RGBAAt(int(r.X)+8, int(r.Y)+8)
		if blocks[b].Trans {
			if c.A == 255 {
				t.Errorf("%s should be translucent", blocks[b].Name)
			}
		} else if blocks[b].Tiny || blocks[b].Item || b == Leaves || b == SpruceLeaves {
			continue // plants, items and leaves are alpha-cutout tiles
		} else if c.A != 255 {
			t.Errorf("%s tile is transparent", blocks[b].Name)
		}
	}
}

// Sunlight reaches open ground, caves are dark, and a torch lights its surroundings
// with falloff; removing it puts the light out again.
func TestLighting(t *testing.T) {
	rand.Seed(5)
	w := NewWorld()
	// Surface cell at the spawn is fully sunlit.
	s := w.SpawnPoint()
	lx, lz := floorI(s.X)-originX, floorI(s.Z)-originZ
	if w.sunLocal(lx, floorI(s.Y), lz) != 15 {
		t.Fatalf("spawn sunlight %d", w.sunLocal(lx, floorI(s.Y), lz))
	}
	// Dig a sealed room three blocks under the spawn, walling it in stone so no cave leaks light in.
	x, z := floorI(s.X), floorI(s.Z)
	y := floorI(s.Y) - 5
	for dx := -2; dx <= 2; dx++ {
		for dz := -2; dz <= 2; dz++ {
			for dy := -1; dy <= 1; dy++ {
				w.Set(x+dx, y+dy, z+dz, Stone)
			}
		}
	}
	for dx := -1; dx <= 1; dx++ {
		for dz := -1; dz <= 1; dz++ {
			w.Set(x+dx, y, z+dz, Air)
		}
	}
	w.flushLight()
	if w.sunLocal(lx, y, lz) != 0 || w.blockLocal(lx, y, lz) != 0 {
		t.Fatalf("sealed room should be dark: sun %d blk %d", w.sunLocal(lx, y, lz), w.blockLocal(lx, y, lz))
	}
	w.Set(x, y, z, Torch)
	w.flushLight()
	if w.blockLocal(lx, y, lz) != 14 || w.blockLocal(lx+1, y, lz) != 13 {
		t.Fatalf("torch light %d, neighbour %d", w.blockLocal(lx, y, lz), w.blockLocal(lx+1, y, lz))
	}
	if w.Luminance(rl.NewVector3(s.X, float32(y), s.Z), 1) <= w.Luminance(rl.NewVector3(s.X+1, float32(y), s.Z), 1) {
		t.Fatal("luminance should fall off from the torch")
	}
	w.Set(x, y, z, Air)
	w.flushLight()
	if w.blockLocal(lx+1, y, lz) != 0 {
		t.Fatalf("light should go out, got %d", w.blockLocal(lx+1, y, lz))
	}
	// Opening the roof lets the sun in.
	for yy := y + 1; yy < worldH; yy++ {
		w.Set(x, yy, z, Air)
	}
	w.flushLight()
	if w.sunLocal(lx, y, lz) != 15 || w.sunLocal(lx+1, y, lz) != 14 {
		t.Fatalf("sun through hole: %d %d", w.sunLocal(lx, y, lz), w.sunLocal(lx+1, y, lz))
	}
}

// A torch is aimable but not solid: entities walk through it and bullets fly past.
func TestTorchRay(t *testing.T) {
	w := NewWorld()
	for z := 0; z <= 5; z++ {
		for y := 18; y < 24; y++ {
			w.Set(5, y, z, Air)
		}
	}
	w.Set(5, 20, 5, Torch)
	o, d := rl.NewVector3(5.5, 20.5, 0.5), rl.NewVector3(0, 0, 1)
	if h := w.RayCastAny(o, d, 20); !h.Hit || h.Z != 5 {
		t.Fatalf("aim ray should stop at the torch: %+v", h)
	}
	if h := w.RayCast(o, d, 20); h.Hit && h.Z == 5 {
		t.Fatal("bullet ray must pass through the torch")
	}
	if w.Solid(5, 20, 5) {
		t.Fatal("torch must not be solid")
	}
}

// Saving and loading round-trips the world, the player and the animals.
func TestSaveLoad(t *testing.T) {
	t.Setenv("BLOCKWORLD_SAVE", t.TempDir()+"/world.sav")
	rand.Seed(6)
	g := &Game{Audio: &Audio{}, CraftHover: -1}
	g.Reset()
	g.World.Set(3, 20, 3, TNT)
	g.World.Set(3, 21, 3, Torch)
	p := g.Player
	p.Pos = rl.NewVector3(3.5, 22, 3.5)
	p.HP, p.Reserve, p.PickTier = 42, 99, TierIron
	p.Inv[DiamondOre] = 7
	g.Score, g.Night = 1234, 2
	g.Sky.T, g.Sky.Day = 0.5, 3
	animals := 0
	for _, a := range g.Animals {
		if a.Alive {
			animals++
		}
	}
	if !saveExists() == false {
		t.Fatal("no save expected yet")
	}
	if !g.save() || !saveExists() {
		t.Fatal("save failed")
	}
	g2 := &Game{Audio: &Audio{}, CraftHover: -1}
	if !g2.load() {
		t.Fatal("load failed")
	}
	if g2.World.Get(3, 20, 3) != TNT || g2.World.Get(3, 21, 3) != Torch {
		t.Fatal("blocks not restored")
	}
	if g2.World.blockLocal(3-originX, 21, 3-originZ) != 14 {
		t.Fatal("lighting not rebuilt after load")
	}
	q := g2.Player
	if q.Pos != p.Pos || q.HP != 42 || q.Reserve != 99 || q.PickTier != TierIron || q.Inv[DiamondOre] != 7 {
		t.Fatalf("player not restored: %+v", q)
	}
	if g2.Score != 1234 || g2.Night != 2 || g2.Sky.Day != 3 || g2.Sky.T != 0.5 {
		t.Fatal("game state not restored")
	}
	if len(g2.Animals) != animals {
		t.Fatalf("animals %d, want %d", len(g2.Animals), animals)
	}
	deleteSave()
	if saveExists() {
		t.Fatal("save should be deleted")
	}
}

// TNT lit by a blast chains, meat is food rather than a block, and eating is a hotbar item.
func TestTNTAndFood(t *testing.T) {
	rand.Seed(7)
	g := &Game{Audio: &Audio{}, CraftHover: -1}
	g.Reset()
	x, z := 10, 10
	y := g.World.SurfaceY(x, z)
	g.World.Set(x, y, z, TNT)
	g.World.Set(x+1, y, z, TNT)
	g.primeTNT(x, y, z, 0.1)
	if len(g.Primed) != 1 || g.World.Get(x, y, z) != Air {
		t.Fatal("priming should lift the block into an entity")
	}
	for i := 0; i < 3; i++ {
		g.updatePrimed(0.05)
	}
	if g.World.Get(x+1, y, z) == TNT {
		t.Fatal("neighbouring TNT should have been lit by the blast")
	}
	if len(g.Primed) != 1 {
		t.Fatalf("chained TNT must stay lit until it explodes, primed=%d", len(g.Primed))
	}
	for i := 0; i < 40; i++ {
		g.updatePrimed(0.05)
	}
	if len(g.Primed) != 0 {
		t.Fatal("chained TNT should have exploded")
	}
	for _, d := range g.Drops {
		if d.Ammo == 0 && d.Block == Air {
			t.Fatal("blast dropped an Air item")
		}
	}
	if g.World.Get(x, y-1, z) != Air {
		t.Fatal("blast should crater the ground")
	}
	if Meat.Placeable() || Apple.Placeable() || CookedMeat.Placeable() {
		t.Fatal("food is not placeable")
	}
	p := g.Player
	p.Inv[Meat] = 1
	hb := p.Hotbar()
	if hb[len(hb)-1] != (Item{ItemFood, Meat}) {
		t.Fatalf("meat should be a food item on the hotbar: %v", hb)
	}
}

// Lava lies in the deep caves and emits light; saplings grow into trees; ladders are climbable, not solid.
func TestLavaSaplingLadder(t *testing.T) {
	rand.Seed(8)
	w := NewWorld()
	lava := 0
	for i, b := range w.Blocks {
		if b == Lava {
			lava++
			y := i / (worldW * worldD)
			if y > 5 {
				t.Fatalf("lava at height %d", y)
			}
		}
	}
	if lava == 0 {
		t.Fatal("no lava generated")
	}
	if !Lava.Liquid() || Lava.Placeable() || blocks[Lava].Solid {
		t.Fatal("lava must be a non-solid, unplaceable liquid")
	}
	// A sapling on dirt in the open grows when told to.
	x, z := 8, 8
	y := w.SurfaceY(x, z)
	for yy := y; yy < y+8; yy++ {
		w.Set(x, yy, z, Air)
	}
	w.Set(x, y-1, z, Dirt)
	w.Set(x, y, z, Sapling)
	if !w.GrowTree(x, y, z) || w.Get(x, y, z) != Log || w.Get(x, y+3, z) != Log {
		t.Fatal("sapling should grow into a trunk")
	}
	leaves := 0
	for dy := 2; dy < 7; dy++ {
		for dx := -2; dx <= 2; dx++ {
			if w.Get(x+dx, y+dy, z) == Leaves {
				leaves++
			}
		}
	}
	if leaves == 0 {
		t.Fatal("tree should have a canopy")
	}
	if w.Solid(x, y, z) == false {
		t.Fatal("log is solid")
	}
	for zz := 10; zz <= 20; zz++ {
		w.Set(20, 20, zz, Air)
	}
	w.Set(20, 20, 20, Ladder)
	if w.Solid(20, 20, 20) || !blocks[Ladder].Tiny {
		t.Fatal("ladders are climbable tiny blocks, not walls")
	}
	if h := w.RayCastAny(rl.NewVector3(20.5, 20.5, 10.5), rl.NewVector3(0, 0, 1), 20); !h.Hit || h.Z != 20 {
		t.Fatalf("ladder should be aimable: %+v", h)
	}
}

// Armour absorbs combat damage but not environmental damage; respawning drops the
// inventory as stacks and keeps tools.
func TestArmorAndRespawn(t *testing.T) {
	rand.Seed(9)
	g := &Game{Audio: &Audio{}, CraftHover: -1}
	g.Reset()
	p := g.Player
	p.ArmorTier = 2
	p.Hurt(20, "test", true)
	if p.HP != 100-11 {
		t.Fatalf("iron armour should absorb 45%%: HP %d", p.HP)
	}
	p.Hurt(20, "lava", false)
	if p.HP != 100-31 {
		t.Fatalf("environmental damage ignores armour: HP %d", p.HP)
	}
	p.Inv[Cobble] = 12
	p.Reserve = 20
	p.PickTier = TierIron
	p.Pos = rl.NewVector3(20.5, 30, 20.5)
	p.Hurt(500, "was slain by a Zombie", true)
	g.checkDeath()
	if g.State != StateGameOver || g.Deaths != 1 || p.Cause != "was slain by a Zombie" {
		t.Fatalf("expected death: state %d deaths %d cause %q", g.State, g.Deaths, p.Cause)
	}
	g.respawn()
	if g.State != StatePlaying || p.HP != maxHealth || p.Inv[Cobble] != 0 || p.PickTier != TierIron {
		t.Fatal("respawn should restore health, drop blocks and keep tools")
	}
	if p.Pos != g.Spawn {
		t.Fatalf("respawned at %v, want %v", p.Pos, g.Spawn)
	}
	stack, ammo := 0, 0
	for _, d := range g.Drops {
		if d.Block == Cobble {
			stack += d.Count
		}
		ammo += d.Ammo
	}
	if stack != 12 || ammo != 32 {
		t.Fatalf("dropped stack %d ammo %d", stack, ammo)
	}
}

// Dungeons generate underground with a spawner and loot; crates pay out when broken.
func TestDungeonsAndLoot(t *testing.T) {
	rand.Seed(10)
	g := &Game{Audio: &Audio{}, CraftHover: -1}
	g.Reset()
	w := g.World
	spawners, crates := 0, 0
	for i, b := range w.Blocks {
		y := i / (worldW * worldD)
		switch b {
		case Spawner:
			spawners++
			if y > 17 {
				t.Fatalf("spawner too high at y=%d", y)
			}
		case Crate:
			crates++
		}
	}
	if spawners == 0 || crates == 0 {
		t.Fatalf("spawners %d crates %d", spawners, crates)
	}
	x, z := 30, 30
	y := w.SurfaceY(x, z)
	w.Set(x, y, z, Crate)
	before := g.Score
	g.breakBlock(x, y, z)
	ammo := 0
	for _, d := range g.Drops {
		ammo += d.Ammo
	}
	if ammo < 16 || g.Score < before+150 || w.Get(x, y, z) != Air {
		t.Fatalf("crate loot: ammo %d score %d", ammo, g.Score-before)
	}
	// A giant blocked by a wall smashes it.
	e := NewEnemy(rl.NewVector3(float32(x)+0.5, float32(y), float32(z)+0.5), KindGiant, 5)
	e.Heading = rl.NewVector3(1, 0, 0)
	for dy := 0; dy < 4; dy++ {
		w.Set(x+1, y+dy, z, Stone)
	}
	g.giantSmash(e)
	if w.Get(x+1, y, z) != Air || w.Get(x+1, y+3, z) != Air {
		t.Fatal("giant should have smashed the wall")
	}
}

// D must strafe toward the camera's right: forward x up, as raylib's camera defines it.
func TestRightIsRight(t *testing.T) {
	for _, yaw := range []float32{0, 0.7, 1.9, 3.1, 4.4} {
		p := &Player{Yaw: yaw}
		want := rl.Vector3CrossProduct(p.FlatForward(), rl.NewVector3(0, 1, 0))
		if rl.Vector3Distance(p.Right(), want) > 1e-5 {
			t.Fatalf("yaw %.1f: Right()=%v want %v", yaw, p.Right(), want)
		}
	}
}

// The local player state must be available offline (third-person view uses it).
func TestLocalStateOffline(t *testing.T) {
	g := &Game{Audio: &Audio{}, CraftHover: -1}
	g.Reset()
	st := g.localState()
	if st.Pos != g.Player.Pos || st.ID != 0 {
		t.Fatalf("bad offline state %+v", st)
	}
}

// Creative mode: no damage, unlimited placeable blocks on the hotbar, instant mining.
func TestCreativeMode(t *testing.T) {
	settings.Creative = true
	defer func() { settings.Creative = false }()
	p := NewPlayer(rl.Vector3{})
	p.Hurt(50, "test", true)
	if p.HP != maxHealth {
		t.Fatal("creative players take no damage")
	}
	found := false
	for _, it := range p.Hotbar() {
		if it == (Item{ItemBlock, DiamondOre}) {
			found = true
		}
	}
	if !found {
		t.Fatal("creative hotbar should list every placeable block")
	}
	if p.MineTime(Stone) > 0.1 {
		t.Fatal("creative mining is instant")
	}
	g := &Game{Audio: &Audio{}, CraftHover: -1}
	g.Reset()
	for _, tt := range g.targets() {
		if tt.ID == 0 && len(g.targets()) > 1 {
			t.Fatal("a creative player should not be a target while villagers exist")
		}
	}
}

// Crops grow in stages, wheat bakes into bread, and the bow fires arrows that hurt hostiles.
func TestFarmingAndBow(t *testing.T) {
	rand.Seed(13)
	g := &Game{Audio: &Audio{}, CraftHover: -1}
	g.Reset()
	w := g.World
	x, z := 12, 12
	y := w.SurfaceY(x, z)
	for yy := y; yy < y+3; yy++ {
		w.Set(x, yy, z, Air)
	}
	w.Set(x, y-1, z, Dirt)
	w.Set(x, y, z, Seeds)
	grown := false
	for i := 0; i < 400 && !grown; i++ {
		g.GrowCD = 0
		g.growSaplings(0.1)
		grown = w.Get(x, y, z) == Wheat
	}
	if !grown {
		t.Fatal("seeds should ripen into wheat")
	}
	p := g.Player
	p.Inv[WheatItem] = 3
	for i := range recipes {
		if recipes[i].Out == Bread && recipes[i].CanCraft(p) {
			recipes[i].Craft(p)
		}
	}
	if p.Inv[Bread] != 1 || blocks[Bread].Food == 0 {
		t.Fatal("three wheat should bake one loaf")
	}
	// Bow: an arrow flying into a zombie hurts it (clear the air along its path first).
	for zz := 15; zz <= 22; zz++ {
		for yy := 29; yy <= 33; yy++ {
			w.Set(20, yy, zz, Air)
		}
	}
	e := NewEnemy(rl.NewVector3(20.5, 30, 20.5), KindZombie, 1)
	g.Enemies = append(g.Enemies, e)
	g.Arrows = append(g.Arrows, Arrow{Pos: rl.NewVector3(20.5, 31, 17), Vel: rl.NewVector3(0, 0, 30), Life: 2, Owner: 1})
	hp := e.HP
	for i := 0; i < 10; i++ {
		g.updateArrows(0.02)
	}
	if e.HP >= hp {
		t.Fatalf("arrow should hurt the zombie: %d -> %d", hp, e.HP)
	}
	p.Inv[Bow] = 1
	hasBow := false
	for _, it := range p.Hotbar() {
		if it.Kind == ItemBow {
			hasBow = true
		}
	}
	if !hasBow {
		t.Fatal("bow should appear on the hotbar")
	}
}

// A dinosaur roams at world start, ignores the player until hurt, then charges and bites.
func TestDinosaur(t *testing.T) {
	rand.Seed(14)
	g := &Game{Audio: &Audio{}, CraftHover: -1}
	g.Reset()
	if g.dinosaurs() != 1 {
		t.Fatalf("dinosaurs at start: %d", g.dinosaurs())
	}
	var d *Animal
	for _, a := range g.Animals {
		if a.Kind == AnimalDino {
			d = a
		}
	}
	p := g.Player
	// Put it next to the player on flat ground and provoke it.
	d.Pos = rl.NewVector3(p.Pos.X+1.5, p.Pos.Y, p.Pos.Z)
	if bite := d.Update(0.1, g.World, g.localTarget()); bite != 0 {
		t.Fatal("a calm dinosaur does not bite")
	}
	d.Hit(1)
	bitten := 0
	for i := 0; i < 40; i++ {
		bitten += d.Update(0.1, g.World, g.localTarget())
	}
	if bitten < 18 {
		t.Fatalf("provoked dinosaur should bite: %d", bitten)
	}
	if d.Flee < 5 {
		t.Fatal("dinosaur should stay angry for a while")
	}
}

// Doors block when closed and pass when open; mineshafts generate; trades consume and pay.
func TestDoorsTraderMineshaft(t *testing.T) {
	rand.Seed(15)
	g := &Game{Audio: &Audio{}, CraftHover: -1}
	g.Reset()
	w := g.World
	w.Set(30, 30, 30, DoorClosed)
	if !w.Solid(30, 30, 30) || !blocks[DoorClosed].Tiny {
		t.Fatal("closed door must block but draw as a panel")
	}
	w.Set(30, 30, 30, DoorOpen)
	if w.Solid(30, 30, 30) || blocks[DoorOpen].Drops != DoorClosed {
		t.Fatal("open door must let things through and drop a door")
	}
	planks := 0
	for i, b := range w.Blocks {
		if b == Planks && i/(worldW*worldD) < 20 {
			planks++
		}
	}
	if planks < 100 {
		t.Fatalf("mineshafts should leave plank floors underground: %d", planks)
	}
	p := g.Player
	p.Inv[GoldOre] = 2
	tr := &trades[0]
	if !tr.can(p) {
		t.Fatal("two gold should afford the ammo trade")
	}
	before := p.Reserve
	tr.apply(p)
	if p.Inv[GoldOre] != 0 || p.Reserve != before+24 {
		t.Fatal("trade should take the gold and pay ammo")
	}
}

// The world wraps: blocks, surfaces, light and pathfinding continue across the edges.
func TestWorldWraps(t *testing.T) {
	rand.Seed(16)
	w := NewWorld()
	east, west := originX+worldW-1, originX
	// Terrain height (ignoring trees and plants) is continuous across the seam: periodic noise.
	ground := func(x, z int) int {
		for y := worldH - 1; y >= 0; y-- {
			switch b := w.Get(x, y, z); b {
			case Air, Water, Log, BirchLog, Leaves, SpruceLeaves, Cactus:
				continue
			default:
				if blocks[b].Tiny {
					continue
				}
				return y
			}
		}
		return 0
	}
	jumps, samples := 0, 0
	for z := originZ; z < originZ+worldD; z += 3 {
		samples++
		if d := ground(east, z) - ground(west, z); d > 3 || d < -3 {
			jumps++ // ruins and dungeons may legitimately sit on the seam
		}
	}
	if jumps*10 > samples {
		t.Fatalf("%d of %d samples jump across the seam: terrain is not periodic", jumps, samples)
	}
	// Reading and writing past the edge lands on the far side.
	w.Set(originX+worldW+2, 30, 5, GoldBlock)
	if w.Get(originX+2, 30, 5) != GoldBlock || w.Get(originX+worldW+2, 30, 5) != GoldBlock {
		t.Fatal("blocks should wrap in x")
	}
	w.Set(7, 31, originZ-3, Glass)
	if w.Get(7, 31, originZ+worldD-3) != Glass {
		t.Fatal("blocks should wrap in z")
	}
	if !w.Solid(originX-1, w.SurfaceY(east, 0)-1, 0) {
		t.Fatal("solidity should wrap")
	}
	// Deltas and positions.
	a := rl.NewVector3(float32(east)+0.5, 10, 0)
	b := rl.NewVector3(float32(west)+0.5, 10, 0)
	if d := WrapDist(a, b); d > 1.01 {
		t.Fatalf("neighbours across the seam are %v apart", d)
	}
	if p := WrapPos(rl.NewVector3(float32(originX+worldW)+3, 1, float32(originZ)-1)); p.X != float32(originX)+3 || p.Z != float32(originZ+worldD)-1 {
		t.Fatalf("WrapPos %v", p)
	}
	// Pathfinding seeds and steps across the seam.
	nav := NewNavGrid(w)
	nav.Update([]rl.Vector3{b}, w)
	x, z := nav.cellOf(a)
	if nav.Dist[z*nav.N+x] < 0 || nav.Dist[z*nav.N+x] > 3 {
		t.Fatalf("east edge should be a step from the west edge: dist %d", nav.Dist[z*nav.N+x])
	}
}

// Villages generate with beds, villagers spawn at them, quests complete, and zombies target villagers.
func TestVillages(t *testing.T) {
	rand.Seed(17)
	g := &Game{Audio: &Audio{}, CraftHover: -1}
	g.Reset()
	villagers := 0
	var v *Animal
	for _, a := range g.Animals {
		if a.Kind == AnimalVillager {
			villagers++
			if v == nil {
				v = a
			}
		}
	}
	if villagers < 4 {
		t.Fatalf("expected villagers in generated villages, got %d", villagers)
	}
	doors, beds := 0, 0
	for _, b := range g.World.Blocks {
		switch b {
		case DoorClosed, DoorOpen:
			doors++
		case Bed:
			beds++
		}
	}
	if doors < 4 || beds < villagers {
		t.Fatalf("houses need doors and beds: %d doors, %d beds", doors, beds)
	}
	// A zombie next to a villager chases it; damage routes to the villager.
	g.Player.Pos = rl.NewVector3(v.Pos.X+60, 14, v.Pos.Z)
	ts := g.targets()
	found := false
	for _, tt := range ts {
		if tt.ID&villagerIDBit != 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("villagers should be hostile targets")
	}
	hp := v.HP
	g.hurtTargetFrom(villagerIDBit|v.ID, 5, "zombie", true, v.Pos, 0)
	if v.HP != hp-5 {
		t.Fatalf("villager should take damage: %d -> %d", hp, v.HP)
	}
	// Quest: give the items and hand them over.
	v.Quest = 1 // 12 cobblestone for bread and apples
	g.Player.Inv[Cobble] = 12
	g.Talking = v
	done, ready, _ := g.questStatus(v)
	if done || !ready {
		t.Fatal("quest should be ready to complete")
	}
	q := &quests[v.Quest]
	g.Player.Inv[q.Need] -= q.NeedN
	q.Give(g.Player)
	v.QuestDone = true
	if g.Player.Inv[Cobble] != 0 || g.Player.Inv[Bread] != 3 {
		t.Fatal("quest should consume cobblestone and pay bread")
	}
	if done, _, _ := g.questStatus(v); !done {
		t.Fatal("quest should be marked done")
	}
}
