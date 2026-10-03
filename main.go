package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var settings = defaultSettings()

type State int

const (
	StateMenu State = iota
	StatePlaying
	StatePaused
	StateCrafting
	StateGameOver
	StateJoin
)

var playerName string

// Primed is a block of TNT with a lit fuse.
type Primed struct {
	Pos  rl.Vector3 // bottom centre
	VelY float32
	Fuse float32
}

type Tracer struct {
	Start, End rl.Vector3
	Life       float32
}

type Spark struct {
	Pos  rl.Vector3
	Vel  rl.Vector3
	Life float32
	Col  rl.Color
}

type Game struct {
	State       State
	World       *World
	Nav         *NavGrid
	Audio       *Audio
	Sky         *Sky
	Player      *Player
	Enemies     []*Enemy
	Animals     []*Animal
	Primed      []Primed // lit TNT
	Arrows      []Arrow
	Drops       []Drop
	AnimalCD    float32
	GrowCD      float32 // sapling growth tick
	ThirdPerson bool
	Spawn       rl.Vector3 // respawn point (world spawn or the last bed used)
	Deaths      int
	ShowHelp    bool
	ShowMap     bool
	RainCD      float32
	SpawnerCD   float32

	Net            *Net
	JoinText       string
	JoinErr        string
	JoinField      int // 0 address, 1 name
	netTestCell    [3]int
	netTestWas     Block
	Remotes        map[uint32]*RemotePlayer
	RemoteHostiles int // hostile count reported by the host (client only)
	nextID         uint32
	SleepT         float32 // fade while sleeping
	Tracers        []Tracer
	Sparks         []Spark

	Night      int // nights that have fallen so far
	WasNight   bool
	SpawnLeft  int // hostiles still to arrive tonight
	SpawnCD    float32
	Score      int
	Kills      int
	HitMark    float32
	Msg        string
	MsgT       float32
	Flash      float32 // muzzle flash timer
	Shake      float32 // camera shake from explosions
	CraftHover int

	HighScore int
	NewHigh   bool

	mapTex rl.Texture2D
	mapOK  bool
	mapVer int
	mapCD  float32
}

func NewGame() *Game {
	g := &Game{Audio: NewAudio(), HighScore: loadHighScore(), CraftHover: -1, Remotes: map[uint32]*RemotePlayer{}}
	g.Reset()
	g.State = StateMenu
	return g
}

// Reset generates a fresh world and starts a run.
func (g *Game) Reset() {
	if g.World != nil {
		g.World.Unload()
	}
	g.World = NewWorld()
	g.Nav = NewNavGrid(g.World)
	g.Sky = NewSky()
	g.Player = NewPlayer(g.World.SpawnPoint())
	g.Spawn = g.World.SpawnPoint()
	g.Deaths = 0
	g.Enemies = nil
	g.Animals = nil
	g.Primed = nil
	g.Arrows = nil
	g.Drops = nil
	g.Tracers = nil
	g.Sparks = nil
	g.Night = 0
	g.WasNight = false
	g.SpawnLeft = 0
	g.Score = 0
	g.Kills = 0
	g.NewHigh = false
	g.mapVer = -1
	g.State = StatePlaying
	g.AnimalCD = 25
	g.spawnAnimals(16)
	g.say("Day 1  -  mine, craft and build before dark  (E = crafting)", 5)
}

func (g *Game) say(s string, t float32) { g.Msg, g.MsgT = s, t }

// announce shows a message locally and to every connected player.
func (g *Game) announce(s string, t float32) {
	g.say(s, t)
	g.sendFx(Fx{Kind: FxMessage, Text: s})
}

// assignIDs gives every entity a stable id for multiplayer snapshots.
func (g *Game) assignIDs() {
	for _, e := range g.Enemies {
		if e.ID == 0 {
			g.nextID++
			e.ID = g.nextID
		}
	}
	for _, a := range g.Animals {
		if a.ID == 0 {
			g.nextID++
			a.ID = g.nextID
		}
	}
	for i := range g.Drops {
		if g.Drops[i].ID == 0 {
			g.nextID++
			g.Drops[i].ID = g.nextID
		}
	}
}

// nightfall starts a night: a first rush of hostiles, the rest trickle in.
func (g *Game) nightfall() {
	g.Night++
	total := int(float32(min(6+g.Night*3, 36)) * hostileScale())
	first := total * 2 / 5
	g.spawnHostiles(first)
	g.SpawnLeft = total - first
	g.SpawnCD = 6
	if total == 0 {
		g.announce(fmt.Sprintf("NIGHT %d  -  peaceful", g.Night), 3)
		return
	}
	if g.Night%5 == 0 {
		if q, ok := g.World.RandomDarkPoint(g.Player.Pos, 20); ok {
			g.Enemies = append(g.Enemies, NewEnemy(q, KindGiant, g.Night))
			g.announce(fmt.Sprintf("NIGHT %d  -  the ground shakes. A GIANT walks.", g.Night), 4)
			g.Audio.Play(g.Audio.Explode, 0.5)
			return
		}
	}
	g.announce(fmt.Sprintf("NIGHT %d  -  the dead rise (%d hostiles)", g.Night, total), 3)
	g.Audio.Play(g.Audio.Wave, 0.8)
}

func (g *Game) dawn() {
	if g.Night == 0 || settings.Difficulty == 0 {
		return
	}
	bonus := 400 * g.Night
	g.Score += bonus
	g.SpawnLeft = 0
	g.announce(fmt.Sprintf("You survived night %d!  +%d", g.Night, bonus), 3.5)
	if g.isHost() {
		g.Net.broadcast(&Msg{Score: &struct {
			Points int
			Kill   bool
		}{bonus, false}}, 0)
	}
	g.Audio.Play(g.Audio.Clear, 0.7)
}

func (g *Game) spawnHostiles(n int) {
	for i := 0; i < n; i++ {
		p, ok := g.World.RandomDarkPoint(g.Player.Pos, 16)
		if !ok {
			continue // the whole surface is lit: nothing rises
		}
		g.Enemies = append(g.Enemies, NewEnemy(p, g.pickKind(), g.Night))
	}
}

// pickKind chooses a hostile type. Spiders and creepers appear from night 2,
// skeletons from night 3, brutes from night 4.
func (g *Game) pickKind() EnemyKind {
	r := rand.Float32()
	switch {
	case g.Night >= 4 && r < 0.12:
		return KindBrute
	case g.Night >= 3 && r < 0.3:
		return KindSkeleton
	case g.Night >= 2 && r < 0.32:
		return KindCreeper
	case g.Night >= 2 && r < 0.55:
		return KindSpider
	}
	return KindZombie
}

// nightsSurvived counts nights that reached dawn.
func (g *Game) nightsSurvived() int {
	if g.Sky.IsNight() {
		return max(g.Night-1, 0)
	}
	return g.Night
}

func (g *Game) aliveEnemies() int {
	n := 0
	for _, e := range g.Enemies {
		if e.Alive {
			n++
		}
	}
	return n
}

// killEnemy scores a kill for the local player and rolls the loot drop.
func (g *Game) killEnemy(e *Enemy, pts int) {
	g.Kills++
	g.Score += pts
	g.killEnemyRaw(e)
}

// killEnemyRaw handles the death itself: sound, loot and messages.
func (g *Game) killEnemyRaw(e *Enemy) {
	g.Audio.Play(g.Audio.Die, 0.7)
	c := rl.NewVector3(e.Pos.X, e.Pos.Y+0.5, e.Pos.Z)
	switch e.Kind {
	case KindZombie:
		if rand.Float32() < 0.4 {
			g.spawnDrop(c, 0, 6)
		}
	case KindSpider:
		if rand.Float32() < 0.3 {
			g.spawnDrop(c, 0, 4)
		}
	case KindCreeper:
		g.spawnDrop(c, CoalOre, 0)
	case KindBrute:
		g.spawnDrop(c, 0, 12)
		g.spawnDrop(c, IronOre, 0)
	case KindSkeleton:
		if rand.Float32() < 0.5 {
			g.spawnDrop(c, 0, 8)
		}
	case KindGiant:
		for i := 0; i < 3; i++ {
			g.spawnDrop(c, DiamondOre, 0)
		}
		g.spawnDrop(c, 0, 24)
		g.announce("GIANT SLAIN  +1500", 3)
	}
}

func (g *Game) fire() {
	p := g.Player
	eye := p.Eye()
	dir := p.Forward()
	// Small spread while moving / in recoil.
	spread := 0.004 + p.BobAmount*0.01
	dir.X += (rand.Float32()*2 - 1) * spread
	dir.Y += (rand.Float32()*2 - 1) * spread
	dir.Z += (rand.Float32()*2 - 1) * spread
	dir = rl.Vector3Normalize(dir)
	ray := rl.NewRay(eye, dir)

	wall := g.World.RayCast(eye, dir, 200)
	wallD := float32(math.Inf(1))
	if wall.Hit {
		wallD = wall.Dist
	}
	bestD := wallD
	var target *Enemy
	var animal *Animal
	headshot := false
	for _, e := range g.Enemies {
		if !e.Alive {
			continue
		}
		c := rl.GetRayCollisionBox(ray, e.BB())
		if c.Hit && c.Distance < bestD {
			bestD = c.Distance
			target = e
			h := rl.GetRayCollisionSphere(ray, e.HeadCenter(), e.Spec.HeadR)
			headshot = h.Hit
		}
	}
	for _, a := range g.Animals {
		if !a.Alive {
			continue
		}
		if c := rl.GetRayCollisionBox(ray, a.BB()); c.Hit && c.Distance < bestD {
			bestD = c.Distance
			animal = a
			target = nil
		}
	}
	end := rl.Vector3Add(eye, rl.Vector3Scale(dir, min(bestD, 200)))
	// Tracer starts from a "muzzle" slightly right/below the eye.
	muzzle := rl.Vector3Add(eye, rl.Vector3Add(rl.Vector3Scale(p.Right(), 0.25), rl.NewVector3(0, -0.2, 0)))
	muzzle = rl.Vector3Add(muzzle, rl.Vector3Scale(dir, 0.6))
	g.Tracers = append(g.Tracers, Tracer{muzzle, end, 0.06})
	g.Flash = 0.05
	g.Audio.Play(g.Audio.Shoot, 0.9)
	g.sendFx(Fx{Kind: FxShot, Pos: eye})

	if g.isClient() {
		// The host resolves damage; we only show feedback.
		if target != nil || animal != nil {
			g.HitMark = 0.15
			g.Audio.Play(g.Audio.Hit, 0.6)
			g.burst(end, rl.NewColor(255, 80, 80, 255), 10)
			h := &struct {
				Enemy    uint32
				Animal   uint32
				Dmg      int
				Headshot bool
				Knock    rl.Vector3
			}{Dmg: headshotDamage(headshot), Headshot: headshot}
			if target != nil {
				h.Enemy = target.ID
			} else {
				h.Animal = animal.ID
			}
			g.sendToHost(&Msg{Hit: h})
		} else if wall.Hit {
			g.burst(end, rl.NewColor(220, 220, 200, 255), 6)
			if g.World.Get(wall.X, wall.Y, wall.Z) == TNT {
				g.sendToHost(&Msg{Prime: &struct{ X, Y, Z int }{wall.X, wall.Y, wall.Z}})
			}
		}
		return
	}

	if target != nil {
		dmg := 1
		col := rl.NewColor(255, 80, 80, 255)
		if headshot {
			dmg = 3
			col = rl.Yellow
		}
		g.HitMark = 0.15
		if headshot {
			g.Audio.Play(g.Audio.Head, 0.7)
		} else {
			g.Audio.Play(g.Audio.Hit, 0.6)
		}
		if target.Hit(dmg) {
			pts := target.Spec.Points
			if headshot {
				pts += pts / 2
				g.say("HEADSHOT", 0.8)
			}
			g.killEnemy(target, pts)
		}
		g.burst(end, col, 10)
	} else if animal != nil {
		g.HitMark = 0.15
		g.Audio.Play(g.Audio.Hit, 0.5)
		g.burst(end, rl.NewColor(255, 80, 80, 255), 8)
		if animal.Hit(headshotDamage(false)) {
			g.killAnimal(animal)
		}
	} else if wall.Hit {
		g.burst(end, rl.NewColor(220, 220, 200, 255), 6)
		if g.World.Get(wall.X, wall.Y, wall.Z) == TNT {
			g.primeTNT(wall.X, wall.Y, wall.Z, 2.5)
		}
	}
}

func headshotDamage(head bool) int {
	if head {
		return 3
	}
	return 1
}

// attack swings the sword at the closest hostile in front of the player.
func (g *Game) attack() {
	p := g.Player
	g.Audio.Play(g.Audio.Swing, 0.5)
	eye := p.Eye()
	ray := rl.NewRay(eye, p.Forward())
	var target *Enemy
	best := float32(swordReach)
	for _, e := range g.Enemies {
		if !e.Alive {
			continue
		}
		bb := e.BB()
		bb.Min = rl.Vector3Subtract(bb.Min, rl.NewVector3(0.25, 0.1, 0.25))
		bb.Max = rl.Vector3Add(bb.Max, rl.NewVector3(0.25, 0.1, 0.25))
		if c := rl.GetRayCollisionBox(ray, bb); c.Hit && c.Distance < best {
			best = c.Distance
			target = e
		}
	}
	var animal *Animal
	for _, a := range g.Animals {
		if !a.Alive {
			continue
		}
		if c := rl.GetRayCollisionBox(ray, a.BB()); c.Hit && c.Distance < best {
			best = c.Distance
			animal = a
			target = nil
		}
	}
	if g.isClient() {
		if target != nil || animal != nil {
			g.HitMark = 0.15
			g.Audio.Play(g.Audio.Hit, 0.7)
			g.burst(rl.Vector3Add(eye, rl.Vector3Scale(ray.Direction, best)), rl.NewColor(255, 90, 90, 255), 8)
			h := &struct {
				Enemy    uint32
				Animal   uint32
				Dmg      int
				Headshot bool
				Knock    rl.Vector3
			}{Dmg: p.SwordDamage()}
			if target != nil {
				h.Enemy = target.ID
				h.Knock = rl.Vector3Scale(p.FlatForward(), 0.9)
			} else {
				h.Animal = animal.ID
			}
			g.sendToHost(&Msg{Hit: h})
		} else if a := p.Aim; a.Hit && g.World.Get(a.X, a.Y, a.Z) == TNT {
			g.sendToHost(&Msg{Prime: &struct{ X, Y, Z int }{a.X, a.Y, a.Z}})
		}
		return
	}
	if animal != nil {
		g.HitMark = 0.15
		g.Audio.Play(g.Audio.Hit, 0.5)
		g.burst(rl.Vector3Add(eye, rl.Vector3Scale(ray.Direction, best)), rl.NewColor(255, 90, 90, 255), 8)
		if animal.Hit(p.SwordDamage()) {
			g.killAnimal(animal)
		}
		return
	}
	if target == nil {
		if a := p.Aim; a.Hit && g.World.Get(a.X, a.Y, a.Z) == TNT {
			g.primeTNT(a.X, a.Y, a.Z, 3)
		}
		return
	}
	g.HitMark = 0.15
	g.Audio.Play(g.Audio.Hit, 0.7)
	g.burst(rl.Vector3Add(eye, rl.Vector3Scale(ray.Direction, best)), rl.NewColor(255, 90, 90, 255), 8)
	// Knockback.
	kb := rl.Vector3Scale(p.FlatForward(), 0.9)
	target.Pos, _ = g.World.MoveBox(target.Pos, target.Spec.Radius, target.Spec.Height, kb, false)
	target.Fuse = max(0, target.Fuse-0.5)
	if target.Hit(p.SwordDamage()) {
		g.killEnemy(target, target.Spec.Points+target.Spec.Points/4)
	}
}

// explode is a creeper blast: hurts everything nearby and tears a crater in the terrain.
func (g *Game) explode(pos rl.Vector3) {
	pos.Y += 0.8
	g.blast(pos, 2.6, 50)
}

// primeTNT lights a TNT block: it becomes a falling entity that explodes after the fuse.
func (g *Game) primeTNT(x, y, z int, fuse float32) {
	if g.World.Get(x, y, z) != TNT {
		return
	}
	g.World.Set(x, y, z, Air)
	g.Primed = append(g.Primed, Primed{Pos: rl.NewVector3(float32(x)+0.5, float32(y), float32(z)+0.5), VelY: 2, Fuse: fuse})
	g.Audio.Play(g.Audio.Fuse, 0.8)
}

func (g *Game) updatePrimed(dt float32) {
	// Blasts may light more TNT (appending to g.Primed) while we iterate, so
	// work on a copy and keep whatever was added afterwards.
	current := append([]Primed(nil), g.Primed...)
	g.Primed = g.Primed[:0]
	var keep []Primed
	for i := range current {
		t := &current[i]
		t.Fuse -= dt
		t.VelY = max(t.VelY-gravity*dt, -25)
		var res MoveResult
		t.Pos, res = g.World.MoveBox(t.Pos, 0.45, 0.95, rl.NewVector3(0, t.VelY*dt, 0), false)
		if res.Ground || res.Ceiling {
			t.VelY = 0
		}
		if t.Fuse <= 0 {
			g.blast(rl.NewVector3(t.Pos.X, t.Pos.Y+0.5, t.Pos.Z), 3.8, 75)
			continue
		}
		keep = append(keep, *t)
	}
	g.Primed = append(keep, g.Primed...)
}

// blast damages the player, hostiles and animals within range, cratering the
// terrain and lighting any TNT caught in it.
func (g *Game) blast(pos rl.Vector3, r, dmg float32) {
	reach := r * 2.5
	for _, t := range g.targets() {
		if d := rl.Vector3Distance(pos, rl.Vector3Add(t.Pos, rl.NewVector3(0, 0.9, 0))); d < reach {
			g.hurtTarget(t.ID, int(dmg*(1-d/reach)*damageScale()), "was blown up", true)
		}
	}
	for _, e := range g.Enemies {
		if e.Alive && rl.Vector3Distance(pos, e.Pos) < r*1.6 {
			if e.Hit(int(dmg / 8)) {
				g.killEnemy(e, e.Spec.Points/2)
			}
		}
	}
	for _, a := range g.Animals {
		if a.Alive && rl.Vector3Distance(pos, a.Pos) < r*1.6 {
			if a.Hit(int(dmg / 8)) {
				g.killAnimal(a)
			}
		}
	}
	cx, cy, cz := floorI(pos.X), floorI(pos.Y), floorI(pos.Z)
	n := int(r) + 1
	for dz := -n; dz <= n; dz++ {
		for dy := -n; dy <= n; dy++ {
			for dx := -n; dx <= n; dx++ {
				d := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
				if d > r+rand.Float32()*0.6 {
					continue
				}
				x, y, z := cx+dx, cy+dy, cz+dz
				b := g.World.Get(x, y, z)
				if b == Air || b.Liquid() || b == Bedrock {
					continue
				}
				if b == TNT {
					g.primeTNT(x, y, z, 0.3+rand.Float32()*0.6)
					continue
				}
				g.World.Set(x, y, z, Air)
				if drop := blocks[b].Drops; drop != Air && rand.Float32() < 0.3 {
					g.spawnDrop(rl.NewVector3(float32(x)+0.5, float32(y)+0.5, float32(z)+0.5), drop, 0)
				}
			}
		}
	}
	for dz := -n; dz <= n; dz++ {
		for dx := -n; dx <= n; dx++ {
			g.settle(cx+dx, cy+n+1, cz+dz)
		}
	}
	g.burst(pos, rl.NewColor(255, 160, 40, 255), int(r*15))
	g.burst(pos, rl.NewColor(90, 90, 90, 255), int(r*12))
	g.Shake = min(1.5, r/2.6)
	g.Audio.Play(g.Audio.Explode, 1)
	g.sendFx(Fx{Kind: FxExplosion, Pos: pos, N: int(r * 15), Shake: min(1.5, r/2.6)})
}

func (g *Game) burst(pos rl.Vector3, col rl.Color, n int) {
	for i := 0; i < n; i++ {
		v := rl.NewVector3(rand.Float32()*2-1, rand.Float32()*2, rand.Float32()*2-1)
		g.Sparks = append(g.Sparks, Spark{pos, rl.Vector3Scale(v, 3), 0.3 + rand.Float32()*0.2, col})
	}
}

// blockOccupied reports whether an entity stands in the given block cell.
func (g *Game) blockOccupied(x, y, z int) bool {
	cell := rl.NewBoundingBox(rl.NewVector3(float32(x), float32(y), float32(z)), rl.NewVector3(float32(x+1), float32(y+1), float32(z+1)))
	if rl.CheckCollisionBoxes(cell, g.Player.Box()) {
		return true
	}
	for _, e := range g.Enemies {
		if e.Alive && rl.CheckCollisionBoxes(cell, e.BB()) {
			return true
		}
	}
	return false
}

// settle lets sand and gravel above a removed block fall, like Minecraft.
func (g *Game) settle(x, y, z int) {
	w := g.World
	for ; y < worldH; y++ {
		b := w.Get(x, y, z)
		if b != Sand && b != Gravel {
			return
		}
		ny := y
		for ny > 0 && !w.Solid(x, ny-1, z) && !g.blockOccupied(x, ny-1, z) {
			ny--
		}
		if ny != y {
			w.Set(x, y, z, Air)
			w.Set(x, ny, z, b)
		}
	}
}

// updateBuilder handles mining (hold left click) and placing (right click).
func (g *Game) updateBuilder(dt float32) {
	p, w := g.Player, g.World
	if usePressed() && p.Held.Kind == ItemFood && p.Inv[p.Held.Block] > 0 {
		if p.HP < maxHealth {
			p.Inv[p.Held.Block]--
			heal := blocks[p.Held.Block].Food
			p.HP = min(maxHealth, p.HP+heal)
			p.Swing = 1
			g.say(fmt.Sprintf("+%d HEALTH", heal), 1)
			g.Audio.Play(g.Audio.Eat, 0.7)
			p.EnsureHeld()
		}
	}
	p.Aim = w.RayCastAny(p.Eye(), p.Forward(), reachDist)
	if !p.Aim.Hit || !w.InBounds(p.Aim.X, p.Aim.Y, p.Aim.Z) {
		p.Aim.Hit = false
		p.Mining = false
		return
	}
	a := p.Aim
	if attackDown() && p.Held.Kind != ItemSword {
		b := w.Get(a.X, a.Y, a.Z)
		info := &blocks[b]
		need := p.MineTime(b)
		if need >= 0 {
			if !p.Mining {
				p.Mining = true
				p.MineT = 0
			}
			p.MineT += dt
			p.Swing = 1
			if p.MineT >= need {
				p.Mining = false
				if g.isClient() {
					g.sendToHost(&Msg{Break: &struct{ X, Y, Z int }{a.X, a.Y, a.Z}})
					g.Audio.Play(g.Audio.Dig, 0.8)
				} else {
					g.breakBlock(a.X, a.Y, a.Z)
				}
			}
		} else {
			p.Mining = false
			if info.MineTime >= 0 && attackPressed() {
				g.say(fmt.Sprintf("%s needs a%s %s pickaxe", info.Name, article(tierNames[info.MinTier]), tierNames[info.MinTier]), 1.5)
			}
		}
	} else {
		p.Mining = false
	}
	if usePressed() && w.Get(a.X, a.Y, a.Z) == Bed {
		g.Spawn = rl.NewVector3(float32(a.X)+0.5, float32(a.Y)+0.5, float32(a.Z)+0.5)
		if g.isClient() {
			g.say("Spawn point set. Only the host can skip the night", 2)
			return
		}
		g.trySleep()
		return
	}
	if usePressed() && p.Held.Kind == ItemBlock && p.Inv[p.Held.Block] > 0 {
		x, y, z := a.X+a.NX, a.Y+a.NY, a.Z+a.NZ
		target := w.Get(x, y, z)
		held := &blocks[p.Held.Block]
		free := target == Air || (target.Liquid() && !held.Tiny)
		if p.Held.Block == Sapling {
			below := w.Get(x, y-1, z)
			free = free && (below == Grass || below == Dirt)
		}
		if w.InBounds(x, y, z) && free && !(held.Solid && g.blockOccupied(x, y, z)) {
			if g.isClient() {
				g.sendToHost(&Msg{Place: &struct {
					X, Y, Z int
					B       Block
				}{x, y, z, p.Held.Block}})
			} else {
				w.Set(x, y, z, p.Held.Block)
				g.sendFx(Fx{Kind: FxPlace, Pos: rl.NewVector3(float32(x), float32(y), float32(z))})
			}
			p.Inv[p.Held.Block]--
			p.Swing = 1
			g.Audio.Play(g.Audio.Place, 0.7)
			p.EnsureHeld()
		}
	}
}

// breakBlock removes a block, drops its item and lets sand above it fall.
func (g *Game) breakBlock(x, y, z int) {
	b := g.World.Get(x, y, z)
	if b == Air {
		return
	}
	info := &blocks[b]
	g.World.Set(x, y, z, Air)
	centre := rl.NewVector3(float32(x)+0.5, float32(y)+0.5, float32(z)+0.5)
	if info.Drops != Air {
		g.spawnDrop(centre, info.Drops, 0)
	}
	if b == Leaves && rand.Float32() < 0.15 {
		g.spawnDrop(centre, Sapling, 0)
	}
	switch b {
	case Crate:
		g.spawnDrop(centre, 0, 16)
		loot := [][2]int{{int(DiamondOre), 2}, {int(GoldOre), 3}, {int(IronOre), 4}, {int(TNT), 2}, {int(Meat), 3}, {int(Torch), 8}}
		pick := loot[rand.Intn(len(loot))]
		for i := 0; i < pick[1]; i++ {
			g.spawnDrop(centre, Block(pick[0]), 0)
		}
		g.Score += 150
		g.say("Loot!", 1.2)
		g.Audio.Play(g.Audio.Craft, 0.8)
	case Spawner:
		g.Score += 500
		g.spawnDrop(centre, DiamondOre, 0)
		g.say("Spawner destroyed  +500", 2)
		g.Audio.Play(g.Audio.Clear, 0.7)
	}
	g.burst(centre, info.Side, 12)
	g.Audio.Play(g.Audio.Dig, 0.8)
	g.sendFx(Fx{Kind: FxDig, Pos: centre, Color: info.Side, N: 12})
	g.settle(x, y+1, z)
}

// soak drives a stress session (BLOCKWORLD_SOAK=1): rapid mining, building,
// day/night cycling, hostiles and explosions, to shake out crashes.
func (g *Game) soak(frame int) bool {
	if frame == 1 {
		g.Reset()
	}
	if frame > 4000 {
		return true
	}
	p, w := g.Player, g.World
	g.Sky.T += 0.0025
	if frame%5 == 0 {
		x, z := floorI(p.Pos.X)+rand.Intn(7)-3, floorI(p.Pos.Z)+rand.Intn(7)-3
		y := w.SurfaceY(x, z) - 1 - rand.Intn(3)
		if y > 0 && w.Get(x, y, z) != Bedrock && !g.blockOccupied(x, y, z) {
			g.breakBlock(x, y, z)
		}
	}
	if frame%7 == 0 {
		x, z := floorI(p.Pos.X)+rand.Intn(9)-4, floorI(p.Pos.Z)+rand.Intn(9)-4
		y := w.SurfaceY(x, z)
		if !g.blockOccupied(x, y, z) && w.InBounds(x, y, z) {
			w.Set(x, y, z, Block(1+rand.Intn(int(numBlocks)-1)))
		}
	}
	if frame%300 == 0 {
		g.ThirdPerson = !g.ThirdPerson
		p.Pos = w.RandomFreePoint(p.Pos, 5)
		p.Yaw = rand.Float32() * 6.28
		p.Pitch = rand.Float32() - 0.5
	}
	if frame%50 == 0 {
		g.spawnHostiles(2)
		q := g.World.RandomFreePoint(p.Pos, 6)
		g.Enemies = append(g.Enemies, NewEnemy(q, KindSkeleton, 3))
	}
	if frame%200 == 0 {
		for _, e := range g.Enemies {
			if e.Alive {
				g.explode(e.Pos)
				break
			}
		}
	}
	if frame%150 == 0 {
		x, z := floorI(p.Pos.X)+rand.Intn(9)-4, floorI(p.Pos.Z)+rand.Intn(9)-4
		y := w.SurfaceY(x, z)
		if w.InBounds(x, y, z) && !g.blockOccupied(x, y, z) {
			w.Set(x, y, z, TNT)
			w.Set(x+1, y, z, TNT)
			g.primeTNT(x, y, z, 0.5)
		}
		for _, a := range g.Animals {
			if a.Alive {
				if a.Hit(3) {
					g.killAnimal(a)
				}
				break
			}
		}
	}
	if frame%120 == 0 {
		x, z := floorI(p.Pos.X)+rand.Intn(9)-4, floorI(p.Pos.Z)+rand.Intn(9)-4
		y := w.SurfaceY(x, z)
		w.Set(x, y-1, z, Dirt)
		w.Set(x, y, z, Sapling)
		w.Set(x+1, y, z, Bed)
		w.Set(x+2, y, z, Ladder)
		w.Set(x+2, y+1, z, Ladder)
		g.GrowCD = 0
		g.trySleep()
	}
	if frame%650 == 0 {
		q := w.RandomFreePoint(p.Pos, 4)
		g.Enemies = append(g.Enemies, NewEnemy(q, KindGiant, 5))
		x, z := floorI(p.Pos.X)+2, floorI(p.Pos.Z)
		y := w.SurfaceY(x, z)
		w.Set(x, y, z, Crate)
		g.breakBlock(x, y, z)
		g.ShowMap = !g.ShowMap
	}
	if frame%700 == 0 {
		p.HP = 0
		g.checkDeath()
		g.respawn()
		g.Sky.Raining = !g.Sky.Raining
		g.Sky.Rain = 1
	}
	if frame%500 == 0 {
		if !g.save() || !g.load() {
			panic("save/load failed")
		}
	}
	if frame%90 == 0 {
		p.CycleHotbar(1)
		p.Inv[Block(1+rand.Intn(int(numBlocks)-1))] += 3
		for i := range recipes {
			if recipes[i].CanCraft(p) {
				recipes[i].Craft(p)
			}
		}
	}
	if frame%1000 == 0 {
		g.Reset()
	}
	if frame%3 == 0 {
		// Input-driven paths: aim at terrain, shoot, swing, mine and place.
		p.Pitch = -0.6
		switch frame % 4 {
		case 0:
			p.Held = Item{Kind: ItemRifle}
			p.Ammo = magSize
			g.fire()
		case 1:
			p.Held = Item{Kind: ItemSword}
			g.attack()
		case 2:
			p.Held = Item{Kind: ItemPickaxe}
			p.Aim = w.RayCast(p.Eye(), p.Forward(), reachDist)
			if p.Aim.Hit {
				g.breakBlock(p.Aim.X, p.Aim.Y, p.Aim.Z)
			}
		case 3:
			p.HoldBlock(Planks)
			p.Inv[Planks] += 2
			a := w.RayCast(p.Eye(), p.Forward(), reachDist)
			if a.Hit {
				x, y, z := a.X+a.NX, a.Y+a.NY, a.Z+a.NZ
				if w.InBounds(x, y, z) && w.Get(x, y, z) == Air && !g.blockOccupied(x, y, z) {
					w.Set(x, y, z, Planks)
				}
			}
		}
	}
	p.HP = maxHealth // the harness never dies
	return false
}

// trySleep skips the night when no hostile is near the bed.
func (g *Game) trySleep() {
	if !g.Sky.IsNight() && g.Sky.T < dayFrac-0.08 {
		g.say("Spawn point set. You can only sleep at night", 1.8)
		return
	}
	for _, e := range g.Enemies {
		if e.Alive && rl.Vector3Distance(e.Pos, g.Player.Pos) < 14 {
			g.say("You may not rest now, there are monsters nearby", 1.8)
			return
		}
	}
	g.SleepT = 1.6
	g.Audio.Play(g.Audio.Craft, 0.5)
}

// updateSleep fades to black, then wakes the player at sunrise with the night skipped.
func (g *Game) updateSleep(dt float32) {
	if g.SleepT <= 0 {
		return
	}
	g.SleepT -= dt
	if g.SleepT <= 0 {
		g.SleepT = 0
		if g.Sky.IsNight() || g.Sky.T >= dayFrac-0.08 {
			if g.Sky.T >= dayFrac-0.08 {
				g.Sky.Day++
			}
			g.Sky.T = 0.02
			g.WasNight = false
			g.SpawnLeft = 0
			g.say(fmt.Sprintf("Day %d  -  you slept through the night", g.Sky.Day), 3)
		}
	}
}

// growSaplings gives every sapling a chance to become a tree each tick.
func (g *Game) growSaplings(dt float32) {
	g.GrowCD -= dt
	if g.GrowCD > 0 {
		return
	}
	g.GrowCD = 3
	w := g.World
	for i, b := range w.Blocks {
		if b != Sapling {
			continue
		}
		x := i % worldW
		z := (i / worldW) % worldD
		y := i / (worldW * worldD)
		if w.sunLocal(x, y, z) < 8 || rand.Float32() > 0.05 {
			continue
		}
		wx, wz := x+originX, z+originZ
		if !g.blockOccupied(wx, y, wz) && w.GrowTree(wx, y, wz) {
			g.burst(rl.NewVector3(float32(wx)+0.5, float32(y)+1, float32(wz)+0.5), rl.Lime, 10)
		}
	}
}

// burnInLava hurts hostiles and animals standing in lava and destroys drops.
func (g *Game) burnInLava(dt float32) {
	for _, e := range g.Enemies {
		if e.Alive && g.World.LavaAt(rl.NewVector3(e.Pos.X, e.Pos.Y+0.2, e.Pos.Z)) {
			e.Burning = true
			e.BurnT += dt
			if e.BurnT >= 0.4 {
				e.BurnT = 0
				if e.Hit(2) {
					g.killEnemy(e, e.Spec.Points/4)
				}
			}
		}
	}
	for _, a := range g.Animals {
		if a.Alive && g.World.LavaAt(rl.NewVector3(a.Pos.X, a.Pos.Y+0.2, a.Pos.Z)) {
			a.WanderT = 0
			if a.Hit(1) {
				g.killAnimal(a)
			}
		}
	}
	keep := g.Drops[:0]
	for _, d := range g.Drops {
		if g.World.LavaAt(d.Pos) {
			g.burst(d.Pos, rl.Orange, 4)
			continue
		}
		keep = append(keep, d)
	}
	g.Drops = keep
}

// respawn drops everything the player carried where they died and returns
// them to the spawn point with full health. Tools and armour are kept.
func (g *Game) respawn() {
	p := g.Player
	at := rl.NewVector3(p.Pos.X, p.Pos.Y+0.5, p.Pos.Z)
	if g.isClient() {
		da := &struct {
			Pos    rl.Vector3
			Blocks [numBlocks]int
			Ammo   int
		}{Pos: at, Blocks: p.Inv, Ammo: p.Ammo + p.Reserve}
		g.sendToHost(&Msg{DropAll: da})
		p.Inv = [numBlocks]int{}
		p.Ammo, p.Reserve = 0, 0
	} else {
		for b := Block(1); b < numBlocks; b++ {
			if p.Inv[b] > 0 {
				g.Drops = append(g.Drops, Drop{Pos: at, Vel: rl.NewVector3(rand.Float32()*2-1, 3, rand.Float32()*2-1), Block: b, Count: p.Inv[b]})
				p.Inv[b] = 0
			}
		}
		if p.Ammo+p.Reserve > 0 {
			g.spawnDrop(at, 0, p.Ammo+p.Reserve)
			p.Ammo, p.Reserve = 0, 0
		}
	}
	p.Pos = g.Spawn
	p.VelY = 0
	p.HP = maxHealth
	p.SinceHurt = 0
	p.DmgFlash = 0
	p.EnsureHeld()
	g.State = StatePlaying
	g.say(fmt.Sprintf("Respawned. Your items lie at %d, %d, %d", floorI(at.X), floorI(at.Y), floorI(at.Z)), 4)
	if !g.isClient() {
		g.save()
	}
}

// giantSmash breaks the blocks in front of a giant so it can walk through anything but bedrock.
func (g *Game) giantSmash(e *Enemy) {
	fx, fz := floorI(e.Pos.X+e.Heading.X*1.2), floorI(e.Pos.Z+e.Heading.Z*1.2)
	for dy := 0; dy < 4; dy++ {
		y := floorI(e.Pos.Y) + dy
		for _, c := range [][2]int{{fx, fz}, {fx + 1, fz}, {fx - 1, fz}, {fx, fz + 1}, {fx, fz - 1}} {
			if rl.Vector3Distance(rl.NewVector3(float32(c[0])+0.5, 0, float32(c[1])+0.5), rl.NewVector3(e.Pos.X, 0, e.Pos.Z)) > 1.9 {
				continue
			}
			if b := g.World.Get(c[0], y, c[1]); b != Air && b != Bedrock && !b.Liquid() {
				g.World.Set(c[0], y, c[1], Air)
				g.burst(rl.NewVector3(float32(c[0])+0.5, float32(y)+0.5, float32(c[1])+0.5), blocks[b].Side, 6)
				g.settle(c[0], y+1, c[1])
			}
		}
	}
	if e.VoiceCD > 1 {
		e.VoiceCD = 1
	}
	g.Shake = max(g.Shake, 0.3)
	g.Audio.Play(g.Audio.Dig, 0.6)
}

// tickSpawners lets monster spawners near the player breed hostiles in the dark.
func (g *Game) tickSpawners(dt float32) {
	g.SpawnerCD -= dt
	if g.SpawnerCD > 0 || settings.Difficulty == 0 {
		return
	}
	g.SpawnerCD = 1.5
	p := g.Player
	w := g.World
	for i, b := range w.Blocks {
		if b != Spawner {
			continue
		}
		x := i%worldW + originX
		z := (i/worldW)%worldD + originZ
		y := i / (worldW * worldD)
		c := rl.NewVector3(float32(x)+0.5, float32(y), float32(z)+0.5)
		if rl.Vector3Distance(c, p.Pos) > 14 || rand.Float32() > 0.3 {
			continue
		}
		near := 0
		for _, e := range g.Enemies {
			if e.Alive && rl.Vector3Distance(e.Pos, c) < 10 {
				near++
			}
		}
		if near >= 4 {
			continue
		}
		for try := 0; try < 8; try++ {
			q := rl.NewVector3(c.X+float32(rand.Intn(5)-2), c.Y, c.Z+float32(rand.Intn(5)-2))
			if w.PointFree(q, 0.4) && w.Solid(floorI(q.X), floorI(q.Y)-1, floorI(q.Z)) {
				kind := KindZombie
				if rand.Float32() < 0.4 {
					kind = KindSkeleton
				}
				g.Enemies = append(g.Enemies, NewEnemy(q, kind, max(1, g.Night)))
				g.burst(rl.Vector3Add(q, rl.NewVector3(0, 1, 0)), rl.NewColor(120, 40, 160, 255), 8)
				break
			}
		}
	}
}

// stepKind maps the block underfoot to a footstep sound.
func stepKind(b Block) int {
	switch b {
	case Stone, Cobble, StoneBrick, CoalOre, IronOre, GoldOre, DiamondOre, Bedrock, GoldBlock, Glass:
		return 1
	case Sand, Gravel, Snow:
		return 2
	case Planks, Log, BirchLog, Bed, Ladder, TNT:
		return 3
	}
	return 0
}

func article(s string) string {
	if len(s) > 0 && (s[0] == 'I' || s[0] == 'i') {
		return "n"
	}
	return ""
}

func (g *Game) update(dt float32) {
	p := g.Player
	wasReloading := p.Reloading > 0
	wasWater := p.InWater
	p.Update(dt, g.World)
	if p.FallDmg > 0 {
		g.Audio.Play(g.Audio.Hurt, 0.8)
	}
	if p.InLava && p.LavaT == 0 {
		g.Audio.Play(g.Audio.Burn, 0.5)
	}
	if p.Stepped {
		g.Audio.Play(g.Audio.Steps[stepKind(g.World.Get(floorI(p.Pos.X), floorI(p.Pos.Y)-1, floorI(p.Pos.Z)))], 0.5)
	}
	if rl.IsKeyPressed(rl.KeyF5) {
		g.ThirdPerson = !g.ThirdPerson
	}
	if rl.IsKeyPressed(rl.KeyH) {
		g.ShowHelp = !g.ShowHelp
	}
	if rl.IsKeyPressed(rl.KeyM) {
		g.ShowMap = !g.ShowMap
	}
	g.RainCD -= dt
	if g.Sky.Rain > 0.1 && g.RainCD <= 0 && g.World.SkyExposed(p.Eye()) {
		g.RainCD = 0.85
		g.Audio.Play(g.Audio.Rain, 0.6*g.Sky.Rain)
	}
	if p.InWater && !wasWater {
		g.Audio.Play(g.Audio.Splash, 0.6)
	}
	switch p.Held.Kind {
	case ItemRifle:
		p.Mining = false
		p.Aim.Hit = false
		if p.TryFire() {
			g.fire()
		} else if attackPressed() && p.Ammo == 0 && p.Reserve == 0 {
			g.Audio.Play(g.Audio.Click, 0.6)
		}
	case ItemSword:
		p.Mining = false
		p.Aim = g.World.RayCastAny(p.Eye(), p.Forward(), reachDist)
		if p.TryAttack() {
			g.attack()
		}
	default:
		g.updateBuilder(dt)
	}
	if !wasReloading && p.Reloading > 0 {
		g.Audio.Play(g.Audio.Reload, 0.8)
	}

	if g.isClient() {
		g.clientUpdate(dt)
		return
	}
	g.assignIDs()

	// Time of day.
	g.Sky.Update(dt)
	night := g.Sky.IsNight()
	if night && !g.WasNight {
		g.nightfall()
	} else if !night && g.WasNight {
		g.dawn()
	}
	g.WasNight = night
	if night && g.SpawnLeft > 0 {
		g.SpawnCD -= dt
		if g.SpawnCD <= 0 {
			g.SpawnCD = 7
			n := min(3, g.SpawnLeft)
			g.spawnHostiles(n)
			g.SpawnLeft -= n
		}
	}

	// Enemies chase whoever is nearest.
	targets := g.targets()
	seeds := make([]rl.Vector3, len(targets))
	for i, t := range targets {
		seeds[i] = t.Pos
	}
	g.Nav.Update(seeds, g.World)
	sunny := g.Sky.Elevation() > 0.08
	for _, e := range g.Enemies {
		fuseBefore := e.Fuse
		if d := e.Update(dt, g.World, g.Nav, g.nearestTarget(e.Pos), g.Enemies); d > 0 {
			g.hurtTarget(e.TargetID, int(float32(d)*damageScale()+0.5), "was slain by a "+e.Spec.Name, true)
		}
		if fuseBefore == 0 && e.Fuse > 0 {
			g.Audio.Play(g.Audio.Fuse, 0.8)
		}
		if e.Exploded {
			e.Exploded = false
			g.explode(e.Pos)
		}
		e.VoiceCD -= dt
		if e.VoiceCD <= 0 && e.Alive {
			e.VoiceCD = 5 + rand.Float32()*8
			if vol := 0.7 * clamp(1-rl.Vector3Distance(e.Pos, p.Pos)/26, 0, 1); vol > 0.05 {
				switch e.Kind {
				case KindZombie, KindBrute:
					g.Audio.Play(g.Audio.Groan, vol)
				case KindSkeleton:
					g.Audio.Play(g.Audio.Rattle, vol)
				}
			}
		}
		if e.Smash {
			g.giantSmash(e)
		}
		if e.Shoot {
			e.Shoot = false
			tp := g.nearestTarget(e.Pos).Pos
			g.shootArrow(rl.NewVector3(e.Pos.X, e.Pos.Y+e.HeadY(), e.Pos.Z), rl.Vector3Add(tp, rl.NewVector3(0, 1.2, 0)))
		}
		// Sunlight burns the undead.
		if e.Alive && e.Spec.Burns && sunny && g.World.SkyExposed(rl.NewVector3(e.Pos.X, e.Pos.Y+e.Spec.Height, e.Pos.Z)) {
			if !e.Burning {
				g.Audio.Play(g.Audio.Burn, 0.4)
			}
			e.Burning = true
			e.BurnT += dt
			if e.BurnT >= 0.7 {
				e.BurnT = 0
				g.Sparks = append(g.Sparks, Spark{rl.NewVector3(e.Pos.X, e.Pos.Y+e.Spec.Height, e.Pos.Z), rl.NewVector3(0, 2, 0), 0.4, rl.Orange})
				if e.Hit(1) {
					g.killEnemy(e, e.Spec.Points/4)
				}
			}
		} else {
			e.Burning = false
		}
	}
	// Remove long-dead enemies.
	live := g.Enemies[:0]
	for _, e := range g.Enemies {
		if e.Alive || e.DeathT < 1 {
			live = append(live, e)
		}
	}
	g.Enemies = live

	g.updateAnimals(dt)
	g.updateArrows(dt)
	g.updatePrimed(dt)
	g.updateDrops(dt)
	g.burnInLava(dt)
	g.growSaplings(dt)
	g.tickSpawners(dt)
	g.updateSleep(dt)

	g.tickEffects(dt)
	g.checkDeath()
}

// clientUpdate is the per-frame work a joined player does locally: the host
// owns the clock, hostiles, animals and drops.
func (g *Game) clientUpdate(dt float32) {
	p := g.Player
	g.Sky.Update(dt) // smoothed between snapshots
	// Walk-over pickup: ask the host for the item.
	keep := g.Drops[:0]
	for _, d := range g.Drops {
		d.Spin += dt * 2
		flat := rl.Vector3Distance(rl.NewVector3(d.Pos.X, 0, d.Pos.Z), rl.NewVector3(p.Pos.X, 0, p.Pos.Z))
		if flat < 1.3 && math.Abs(float64(p.Pos.Y-d.Pos.Y)) < 1.8 {
			g.sendToHost(&Msg{Pickup: &struct{ ID uint32 }{d.ID}})
			continue
		}
		keep = append(keep, d)
	}
	g.Drops = keep
	for _, e := range g.Enemies {
		e.Phase += dt * e.Speed * 2
	}
	for _, a := range g.Animals {
		if a.Alive && a.Phase != 0 {
			a.Phase += dt * 2
		}
	}
	g.updateSleep(dt)
	g.tickEffects(dt)
	g.checkDeath()
}

// tickEffects ages tracers, sparks and timers.
func (g *Game) tickEffects(dt float32) {
	tr := g.Tracers[:0]
	for _, t := range g.Tracers {
		t.Life -= dt
		if t.Life > 0 {
			tr = append(tr, t)
		}
	}
	g.Tracers = tr
	sp := g.Sparks[:0]
	for _, s := range g.Sparks {
		s.Life -= dt
		s.Vel.Y -= 9 * dt
		s.Pos = rl.Vector3Add(s.Pos, rl.Vector3Scale(s.Vel, dt))
		if s.Life > 0 {
			sp = append(sp, s)
		}
	}
	g.Sparks = sp
	g.HitMark = max(0, g.HitMark-dt)
	g.Flash = max(0, g.Flash-dt)
	g.Shake = max(0, g.Shake-dt*1.5)
	g.MsgT = max(0, g.MsgT-dt)
}

// checkDeath moves to the death screen once health runs out.
func (g *Game) checkDeath() {
	if g.Player.HP > 0 || g.State == StateGameOver {
		return
	}
	g.State = StateGameOver
	g.Deaths++
	if rl.IsWindowReady() {
		rl.EnableCursor()
	}
	g.Audio.Play(g.Audio.GameOver, 0.9)
	if g.Score > g.HighScore {
		g.HighScore = g.Score
		g.NewHigh = true
		saveHighScore(g.Score)
	}
}

func (g *Game) camera() rl.Camera3D {
	cam := g.Player.Camera()
	if g.ThirdPerson {
		// Pull the camera back along the view direction, stopping short of terrain.
		fwd := rl.Vector3Normalize(rl.Vector3Subtract(cam.Target, cam.Position))
		back := rl.Vector3Scale(fwd, -1)
		d := float32(4.5)
		if h := g.World.RayCast(cam.Position, back, d); h.Hit {
			d = max(h.Dist-0.4, 0.5)
		}
		cam.Target = cam.Position
		cam.Position = rl.Vector3Add(cam.Position, rl.Vector3Scale(back, d))
	}
	if g.Shake > 0 {
		j := g.Shake * g.Shake * 0.12
		cam.Target = rl.Vector3Add(cam.Target, rl.NewVector3((rand.Float32()*2-1)*j, (rand.Float32()*2-1)*j, (rand.Float32()*2-1)*j))
	}
	return cam
}

func (g *Game) draw3D() {
	p := g.Player
	cam := g.camera()
	w := g.World
	env := g.Sky.Env(p.HeadWater)
	w.SetEnv(cam, env)
	rl.BeginMode3D(cam)
	if !p.HeadWater {
		g.Sky.DrawSky(cam)
	}
	w.Draw(cam)
	// Entities go through the world shader so they are lit and fogged like the terrain.
	w.BeginShader()
	for _, e := range g.Enemies {
		e.Lum = w.Luminance(rl.NewVector3(e.Pos.X, e.Pos.Y+0.5, e.Pos.Z), env.Light)
		e.Draw()
	}
	for _, a := range g.Animals {
		a.Lum = w.Luminance(rl.NewVector3(a.Pos.X, a.Pos.Y+0.5, a.Pos.Z), env.Light)
		a.Draw()
	}
	if g.ThirdPerson {
		g.drawHumanoid(g.localState(), p.EyeOff, p.SwordTier, p.PickTier, w.Luminance(rl.NewVector3(p.Pos.X, p.Pos.Y+1, p.Pos.Z), env.Light))
	}
	for _, r := range g.Remotes {
		off := float32(0)
		if r.Sneak {
			off = 0.3
		}
		g.drawHumanoid(r.PlayerState, off, TierIron, TierIron, w.Luminance(rl.NewVector3(r.Pos.X, r.Pos.Y+1, r.Pos.Z), env.Light))
	}
	g.drawDrops()
	g.drawArrows()
	for i := range g.Primed {
		t := &g.Primed[i]
		w.DrawBlockAt(TNT, rl.MatrixTranslate(t.Pos.X, t.Pos.Y+0.5, t.Pos.Z))
		if math.Sin(float64(t.Fuse*18)) > 0 {
			rl.DrawCube(rl.NewVector3(t.Pos.X, t.Pos.Y+0.5, t.Pos.Z), 1.02, 1.02, 1.02, rl.NewColor(255, 255, 255, 170))
		}
	}
	g.Sky.DrawClouds(cam, float32(rl.GetTime()))
	w.EndShader()
	if !p.HeadWater && w.SkyExposed(p.Eye()) {
		g.Sky.DrawRain(cam, float32(rl.GetTime()))
	}
	for _, e := range g.Enemies {
		e.DrawGlow()
	}
	// Placement preview: a ghost of the held block where right click would put it.
	if p.Aim.Hit && p.Held.Kind == ItemBlock {
		a := p.Aim
		x, y, z := a.X+a.NX, a.Y+a.NY, a.Z+a.NZ
		if w.InBounds(x, y, z) && (w.Get(x, y, z) == Air || w.Get(x, y, z).Liquid()) {
			box := unitBox
			if blocks[p.Held.Block].Tiny {
				box = p.Held.Block.TinyBox()
			}
			c := rl.NewVector3(float32(x)+(box[0][0]+box[1][0])/2, float32(y)+(box[0][1]+box[1][1])/2, float32(z)+(box[0][2]+box[1][2])/2)
			sz := rl.NewVector3(box[1][0]-box[0][0], box[1][1]-box[0][1], box[1][2]-box[0][2])
			rl.DrawCubeV(c, sz, rl.Fade(rl.White, 0.18))
			rl.DrawCubeWiresV(c, sz, rl.Fade(rl.White, 0.6))
		}
	}
	// Block selection outline and crack progress.
	if p.Aim.Hit && p.Held.Kind != ItemRifle {
		c := rl.NewVector3(float32(p.Aim.X)+0.5, float32(p.Aim.Y)+0.5, float32(p.Aim.Z)+0.5)
		rl.DrawCubeWires(c, 1.005, 1.005, 1.005, rl.NewColor(0, 0, 0, 200))
		if p.Mining {
			if need := p.MineTime(w.Get(p.Aim.X, p.Aim.Y, p.Aim.Z)); need > 0 {
				frac := clamp(p.MineT/need, 0, 1)
				rl.DrawCube(c, 1.01, 1.01, 1.01, rl.Fade(rl.Black, frac*0.55))
			}
		}
	}
	for _, tr := range g.Tracers {
		rl.DrawLine3D(tr.Start, tr.End, rl.NewColor(255, 240, 160, 255))
	}
	for _, s := range g.Sparks {
		rl.DrawCube(s.Pos, 0.1, 0.1, 0.1, rl.Fade(s.Col, s.Life*3))
	}
	w.DrawTranslucent(cam)
	rl.EndMode3D()
}

// drawHumanoid draws a blocky player: the local one in third person, or a remote one.
func (g *Game) drawHumanoid(ps PlayerState, eyeOff float32, swordTier, pickTier int, lum float32) {
	fwd := rl.NewVector3(float32(math.Sin(float64(ps.Yaw))), 0, float32(math.Cos(float64(ps.Yaw))))
	side := rl.NewVector3(-float32(math.Cos(float64(ps.Yaw))), 0, float32(math.Sin(float64(ps.Yaw))))
	x, z := ps.Pos.X, ps.Pos.Z
	y := ps.Pos.Y - eyeOff*0.5
	p := struct {
		BobPhase, BobAmount, Swing float32
		Held                       Item
		SwordTier, PickTier        int
	}{ps.BobPhase, ps.BobAmt, ps.Swing, ps.Held, swordTier, pickTier}
	skin := mul(rl.NewColor(200, 160, 120, 255), lum)
	shirt := mul(rl.NewColor(60, 170, 170, 255), lum)
	pants := mul(rl.NewColor(50, 60, 150, 255), lum)
	hair := mul(rl.NewColor(70, 45, 30, 255), lum)
	outline := rl.NewColor(20, 20, 20, 255)
	swing := float32(math.Sin(float64(p.BobPhase))) * 0.3 * p.BobAmount
	for _, sg := range []float32{-1, 1} {
		lp := rl.Vector3Add(rl.NewVector3(x, y+0.4, z), rl.Vector3Add(rl.Vector3Scale(side, sg*0.15), rl.Vector3Scale(fwd, swing*sg)))
		rl.DrawCubeV(lp, rl.NewVector3(0.26, 0.8, 0.26), pants)
	}
	torso := rl.NewVector3(x, y+1.1, z)
	rl.DrawCubeV(torso, rl.NewVector3(0.6, 0.7, 0.32), shirt)
	rl.DrawCubeWiresV(torso, rl.NewVector3(0.6, 0.7, 0.32), outline)
	for _, sg := range []float32{-1, 1} {
		armSwing := swing * -sg
		if p.Swing > 0 && sg == 1 {
			armSwing = 0.4 * p.Swing
		}
		ap := rl.Vector3Add(rl.NewVector3(x, y+1.1, z), rl.Vector3Add(rl.Vector3Scale(side, sg*0.42), rl.Vector3Scale(fwd, armSwing)))
		rl.DrawCubeV(ap, rl.NewVector3(0.22, 0.7, 0.22), skin)
	}
	head := rl.NewVector3(x, y+1.72, z)
	rl.DrawCubeV(head, rl.NewVector3(0.5, 0.5, 0.5), skin)
	rl.DrawCubeV(rl.Vector3Add(head, rl.NewVector3(0, 0.2, 0)), rl.NewVector3(0.52, 0.14, 0.52), hair)
	rl.DrawCubeWiresV(head, rl.NewVector3(0.5, 0.5, 0.5), outline)
	eyes := rl.Vector3Add(head, rl.Vector3Scale(fwd, 0.26))
	for _, sg := range []float32{-1, 1} {
		rl.DrawCubeV(rl.Vector3Add(eyes, rl.Vector3Scale(side, sg*0.11)), rl.NewVector3(0.08, 0.08, 0.02), rl.NewColor(30, 30, 60, 255))
	}
	// Held tool or block.
	hand := rl.Vector3Add(rl.NewVector3(x, y+0.85, z), rl.Vector3Add(rl.Vector3Scale(side, 0.45), rl.Vector3Scale(fwd, 0.3)))
	switch p.Held.Kind {
	case ItemBlock, ItemFood:
		g.World.DrawBlockAt(p.Held.Block, rl.MatrixMultiply(rl.MatrixScale(0.3, 0.3, 0.3), rl.MatrixTranslate(hand.X, hand.Y, hand.Z)))
	case ItemSword:
		rl.DrawCubeV(rl.Vector3Add(hand, rl.Vector3Scale(fwd, 0.3)), rl.NewVector3(0.08, 0.1, 0.8), mul(tierColors[p.SwordTier], lum))
	case ItemPickaxe:
		rl.DrawCubeV(rl.Vector3Add(hand, rl.Vector3Scale(fwd, 0.3)), rl.NewVector3(0.08, 0.08, 0.7), mul(rl.NewColor(120, 85, 45, 255), lum))
		rl.DrawCubeV(rl.Vector3Add(hand, rl.Vector3Scale(fwd, 0.6)), rl.NewVector3(0.3, 0.12, 0.12), mul(tierColors[p.PickTier], lum))
	case ItemRifle:
		rl.DrawCubeV(rl.Vector3Add(hand, rl.Vector3Scale(fwd, 0.35)), rl.NewVector3(0.1, 0.14, 0.9), mul(rl.NewColor(50, 52, 58, 255), lum))
	}
}

var tierColors = [...]rl.Color{
	rl.NewColor(200, 160, 120, 255), // hand
	rl.NewColor(150, 110, 60, 255),  // wood
	rl.NewColor(120, 120, 124, 255), // stone
	rl.NewColor(220, 220, 225, 255), // iron
	rl.NewColor(90, 230, 225, 255),  // diamond
}

func drawSwordIcon(cx, cy, size float32, tier int) {
	blade := tierColors[tier]
	rl.DrawRectanglePro(rl.NewRectangle(cx, cy, size*0.18, size*0.75), rl.NewVector2(size*0.09, size*0.65), 45, blade)
	rl.DrawRectanglePro(rl.NewRectangle(cx, cy, size*0.4, size*0.1), rl.NewVector2(size*0.2, size*0.05), 45, rl.NewColor(110, 80, 40, 255))
	rl.DrawRectanglePro(rl.NewRectangle(cx, cy, size*0.12, size*0.3), rl.NewVector2(size*0.06, -size*0.02), 45, rl.NewColor(90, 60, 30, 255))
}

func drawPickIcon(cx, cy, size float32, tier int) {
	head := tierColors[tier]
	rl.DrawRectanglePro(rl.NewRectangle(cx, cy, size*0.14, size*0.9), rl.NewVector2(size*0.07, size*0.75), 45, rl.NewColor(120, 85, 45, 255))
	rl.DrawRectanglePro(rl.NewRectangle(cx, cy, size*0.7, size*0.2), rl.NewVector2(size*0.35, size*0.85), 45, head)
}

// drawBlockIcon draws a block as its side texture with a strip of its top.
func (g *Game) drawBlockIcon(b Block, x, y, size int32) {
	tex := g.World.Atlas()
	s := float32(size)
	top := s * 0.26
	rl.DrawTexturePro(tex, TileRect(b, 1), rl.NewRectangle(float32(x), float32(y)+top, s, s-top), rl.Vector2{}, 0, rl.White)
	rl.DrawTexturePro(tex, TileRect(b, 0), rl.NewRectangle(float32(x), float32(y), s, top), rl.Vector2{}, 0, rl.NewColor(255, 255, 255, 255))
	rl.DrawRectangleLines(x, y, size, size, rl.NewColor(0, 0, 0, 120))
}

func (g *Game) drawWeapon() {
	sw, sh := float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())
	p := g.Player
	bobX := float32(math.Sin(float64(p.BobPhase))) * 6 * p.BobAmount
	bobY := float32(math.Abs(math.Sin(float64(p.BobPhase)))) * 5 * p.BobAmount
	dark := rl.NewColor(40, 42, 48, 255)
	mid := rl.NewColor(70, 74, 82, 255)
	skin := rl.NewColor(200, 160, 120, 255)
	swing := float32(math.Sin(float64(p.Swing * math.Pi)))
	switch p.Held.Kind {
	case ItemBlock, ItemFood:
		rx := sw*0.68 + bobX - swing*12
		ry := sh - 170 + bobY + swing*40
		rl.DrawRectangle(int32(rx+40), int32(ry+40), 90, 220, skin)
		rl.DrawRectangle(int32(rx+40), int32(ry+40), 90, 220, rl.Fade(rl.Black, 0.15))
		tex := g.World.Atlas()
		b := p.Held.Block
		rl.DrawTexturePro(tex, TileRect(b, 1), rl.NewRectangle(rx, ry, 130, 130), rl.Vector2{}, 0, rl.White)
		rl.DrawTexturePro(tex, TileRect(b, 0), rl.NewRectangle(rx, ry-34, 130, 34), rl.Vector2{}, 0, rl.NewColor(255, 255, 255, 255))
		rl.DrawRectangleLines(int32(rx), int32(ry-34), 130, 164, rl.NewColor(0, 0, 0, 160))
		return
	case ItemPickaxe, ItemSword:
		rx := sw*0.72 + bobX
		ry := sh - 60 + bobY
		rot := float32(-30) + swing*55
		if p.Held.Kind == ItemSword {
			rot = -55 + swing*80
		}
		// Arm.
		rl.DrawRectanglePro(rl.NewRectangle(rx+30, ry+60, 80, 200), rl.NewVector2(40, 0), rot*0.4, skin)
		handle := rl.NewColor(120, 85, 45, 255)
		if p.Held.Kind == ItemPickaxe {
			rl.DrawRectanglePro(rl.NewRectangle(rx, ry, 26, 300), rl.NewVector2(13, 290), rot, handle)
			rl.DrawRectanglePro(rl.NewRectangle(rx, ry, 190, 46), rl.NewVector2(95, 290), rot, tierColors[p.PickTier])
			rl.DrawRectanglePro(rl.NewRectangle(rx, ry, 190, 46), rl.NewVector2(95, 290), rot, rl.Fade(rl.Black, 0.12))
		} else {
			rl.DrawRectanglePro(rl.NewRectangle(rx, ry, 24, 80), rl.NewVector2(12, 60), rot, handle)
			rl.DrawRectanglePro(rl.NewRectangle(rx, ry, 90, 22), rl.NewVector2(45, 70), rot, rl.NewColor(90, 60, 30, 255))
			rl.DrawRectanglePro(rl.NewRectangle(rx, ry, 40, 280), rl.NewVector2(20, 350), rot, tierColors[p.SwordTier])
			rl.DrawRectanglePro(rl.NewRectangle(rx, ry, 14, 270), rl.NewVector2(0, 345), rot, rl.Fade(rl.Black, 0.18))
		}
		return
	}
	kick := p.Recoil * 28
	rx := sw*0.62 + bobX
	ry := sh - 150 + bobY + kick
	if p.Reloading > 0 {
		ry += 90 * float32(math.Sin(float64(p.Reloading/reloadTime*math.Pi)))
	}
	// Barrel, body, grip.
	rl.DrawRectangle(int32(rx+40), int32(ry-20), 200, 34, mid)
	rl.DrawRectangle(int32(rx+60), int32(ry-8), 190, 12, dark)
	rl.DrawRectangle(int32(rx), int32(ry), 180, 70, dark)
	rl.DrawRectangle(int32(rx+20), int32(ry+60), 60, 120, mid)
	rl.DrawRectangle(int32(rx+110), int32(ry+50), 24, 60, rl.NewColor(180, 140, 40, 255))
	if g.Flash > 0 {
		rl.DrawCircle(int32(rx+250), int32(ry-4), 28+rand.Float32()*10, rl.Fade(rl.Yellow, 0.9))
		rl.DrawCircle(int32(rx+250), int32(ry-4), 14, rl.White)
	}
}

func (g *Game) drawHotbar(sw, sh int32) {
	p := g.Player
	const slot = 50
	const gap = 4
	total := int32(hotbarSlots*(slot+gap) + gap)
	x0 := sw/2 - total/2
	y0 := sh - slot - 14
	rl.DrawRectangle(x0, y0-gap, total, slot+2*gap, rl.NewColor(0, 0, 0, 150))
	hb := p.Hotbar()
	sel := p.SelIndex()
	for i := 0; i < hotbarSlots; i++ {
		x := x0 + gap + int32(i)*(slot+gap)
		idx := p.HotScroll + i
		rl.DrawRectangle(x, y0, slot, slot, rl.NewColor(60, 60, 66, 230))
		rl.DrawRectangleLines(x, y0, slot, slot, rl.NewColor(30, 30, 34, 255))
		if idx < len(hb) {
			it := hb[idx]
			cx, cy := float32(x)+slot/2, float32(y0)+slot/2
			switch it.Kind {
			case ItemRifle:
				rl.DrawRectangle(x+8, y0+20, 34, 8, rl.NewColor(70, 74, 82, 255))
				rl.DrawRectangle(x+10, y0+26, 12, 14, rl.NewColor(40, 42, 48, 255))
			case ItemSword:
				drawSwordIcon(cx, cy, slot*0.9, p.SwordTier)
			case ItemPickaxe:
				drawPickIcon(cx, cy, slot*0.9, p.PickTier)
			case ItemBlock, ItemFood:
				g.drawBlockIcon(it.Block, x+9, y0+9, slot-18)
				cnt := fmt.Sprintf("%d", p.Inv[it.Block])
				tw := rl.MeasureText(cnt, 16)
				rl.DrawText(cnt, x+slot-tw-3, y0+slot-18, 16, rl.Black)
				rl.DrawText(cnt, x+slot-tw-4, y0+slot-19, 16, rl.White)
			}
		}
		if i+1 <= 9 {
			rl.DrawText(fmt.Sprintf("%d", i+1), x+3, y0+2, 12, rl.LightGray)
		}
		if idx == sel {
			rl.DrawRectangleLinesEx(rl.NewRectangle(float32(x)-2, float32(y0)-2, slot+4, slot+4), 3, rl.White)
		}
	}
	// Name of the held item, Minecraft style.
	name := ""
	switch p.Held.Kind {
	case ItemRifle:
		name = "Rifle"
	case ItemSword:
		name = tierNames[p.SwordTier] + " Sword"
	case ItemPickaxe:
		name = tierNames[p.PickTier] + " Pickaxe"
	case ItemBlock:
		name = blocks[p.Held.Block].Name
	case ItemFood:
		name = blocks[p.Held.Block].Name + "  (right click to eat)"
	}
	if len(hb) > hotbarSlots {
		name += "   (wheel scrolls)"
	}
	tw := rl.MeasureText(name, 20)
	rl.DrawText(name, sw/2-tw/2+1, y0-30+1, 20, rl.Black)
	rl.DrawText(name, sw/2-tw/2, y0-30, 20, rl.White)
}

func drawHeart(x, y, size float32, col rl.Color) {
	r := size * 0.28
	rl.DrawCircleV(rl.NewVector2(x+r, y+r), r, col)
	rl.DrawCircleV(rl.NewVector2(x+size-r, y+r), r, col)
	rl.DrawTriangle(rl.NewVector2(x, y+r*1.1), rl.NewVector2(x+size/2, y+size), rl.NewVector2(x+size, y+r*1.1), col)
	rl.DrawRectangleV(rl.NewVector2(x+r, y+r*0.3), rl.NewVector2(size-2*r, r), col)
}

func (g *Game) drawHUD() {
	sw, sh := int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight())
	p := g.Player
	cx, cy := sw/2, sh/2

	// Underwater and lava tints.
	if p.HeadWater {
		rl.DrawRectangle(0, 0, sw, sh, rl.NewColor(20, 60, 140, 90))
	}
	if p.InLava {
		rl.DrawRectangle(0, 0, sw, sh, rl.NewColor(255, 90, 0, 110))
	}
	if g.SleepT > 0 {
		rl.DrawRectangle(0, 0, sw, sh, rl.Fade(rl.Black, clamp(1.6-g.SleepT, 0, 1)))
	}
	// Crosshair.
	gap := int32(5 + p.BobAmount*4 + p.Recoil*10)
	l := int32(9)
	col := rl.NewColor(255, 255, 255, 220)
	if g.HitMark > 0 {
		col = rl.Red
	}
	rl.DrawRectangle(cx-gap-l, cy-1, l, 2, col)
	rl.DrawRectangle(cx+gap, cy-1, l, 2, col)
	rl.DrawRectangle(cx-1, cy-gap-l, 2, l, col)
	rl.DrawRectangle(cx-1, cy+gap, 2, l, col)
	if g.HitMark > 0 {
		for _, d := range [][2]int32{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
			rl.DrawLineEx(rl.NewVector2(float32(cx+d[0]*8), float32(cy+d[1]*8)),
				rl.NewVector2(float32(cx+d[0]*16), float32(cy+d[1]*16)), 2, rl.Red)
		}
	}
	// Damage vignette.
	if p.DmgFlash > 0 {
		rl.DrawRectangle(0, 0, sw, sh, rl.Fade(rl.Red, p.DmgFlash*0.35))
	}

	if !g.ThirdPerson {
		g.drawWeapon()
	}

	// Hearts: ten hearts, ten health each.
	hx, hy := int32(20), sh-64
	rl.DrawRectangle(hx-8, hy-8, 10*26+16, 40, rl.NewColor(0, 0, 0, 140))
	for i := 0; i < 10; i++ {
		x := float32(hx + int32(i)*26)
		drawHeart(x, float32(hy), 22, rl.NewColor(40, 20, 20, 255))
		have := p.HP - i*10
		switch {
		case have >= 10:
			drawHeart(x, float32(hy), 22, rl.NewColor(230, 40, 40, 255))
		case have >= 5:
			drawHeart(x, float32(hy), 22, rl.NewColor(230, 40, 40, 255))
			rl.DrawRectangle(int32(x)+11, hy-2, 12, 26, rl.NewColor(40, 20, 20, 255))
		}
	}
	rl.DrawText(fmt.Sprintf("%d", p.HP), hx+10*26+4, hy, 20, rl.White)
	if p.Sneak {
		rl.DrawText("SNEAKING", hx, hy-30, 16, rl.LightGray)
	}

	// Ammo.
	rl.DrawRectangle(sw-220, sh-70, 200, 50, rl.NewColor(0, 0, 0, 140))
	ammoTxt := fmt.Sprintf("%d / %d", p.Ammo, p.Reserve)
	if p.Reloading > 0 {
		ammoTxt = "RELOADING"
	}
	size := int32(20)
	if p.Held.Kind == ItemRifle {
		size = 32
	}
	rl.DrawText(ammoTxt, sw-205, sh-62, size, rl.White)
	if p.Held.Kind == ItemRifle && p.Ammo == 0 && p.Reloading == 0 && p.Reserve == 0 {
		rl.DrawText("OUT OF AMMO - craft ammo from iron ore and coal (E)", cx-250, cy+60, 20, rl.Orange)
	}

	g.drawHotbar(sw, sh)

	// Score, day and hostiles.
	rl.DrawRectangle(20, 20, 360, 100, rl.NewColor(0, 0, 0, 140))
	rl.DrawText(fmt.Sprintf("SCORE  %d", g.Score), 30, 28, 26, rl.White)
	rl.DrawText(fmt.Sprintf("DAY %d  %s  %s", g.Sky.Day, g.Sky.TimeLabel(), g.clock()), 30, 62, 18, rl.LightGray)
	hostiles := g.aliveEnemies()
	if g.isClient() {
		hostiles = g.RemoteHostiles
	}
	rl.DrawText(fmt.Sprintf("HOSTILES %d    KILLS %d    BEST %d", hostiles, g.Kills, g.HighScore), 30, 88, 18, rl.Gold)

	g.drawMinimap(sw)

	// Message.
	if g.MsgT > 0 {
		fs := int32(32)
		tw := rl.MeasureText(g.Msg, fs)
		a := min(g.MsgT*2, 1)
		rl.DrawText(g.Msg, cx-tw/2+2, 132, fs, rl.Fade(rl.Black, a))
		rl.DrawText(g.Msg, cx-tw/2, 130, fs, rl.Fade(rl.Gold, a))
	}

	// Name tags over other players.
	if len(g.Remotes) > 0 {
		cam := g.camera()
		fwd := rl.Vector3Normalize(rl.Vector3Subtract(cam.Target, cam.Position))
		for _, r := range g.Remotes {
			head := rl.Vector3Add(r.Pos, rl.NewVector3(0, playerHeight+0.4, 0))
			if rl.Vector3DotProduct(rl.Vector3Subtract(head, cam.Position), fwd) <= 0.5 {
				continue
			}
			sp := rl.GetWorldToScreen(head, cam)
			tw := rl.MeasureText(r.Name, 16)
			rl.DrawRectangle(int32(sp.X)-tw/2-4, int32(sp.Y)-10, tw+8, 20, rl.NewColor(0, 0, 0, 120))
			rl.DrawText(r.Name, int32(sp.X)-tw/2, int32(sp.Y)-8, 16, rl.White)
		}
	}
	if g.Net != nil {
		status := fmt.Sprintf("You are %s   %s   %d players", g.Net.Name, g.Net.Status, g.Net.PlayerCount())
		if g.isClient() {
			status = fmt.Sprintf("You are %s   %s", g.Net.Name, g.Net.Status)
		}
		rl.DrawText(status, 30, 124, 14, rl.SkyBlue)
	}

	// Armour next to the hearts.
	if p.ArmorTier > 0 {
		drawArmorIcon(float32(hx+10*26+58), float32(hy+10), 22, p.ArmorTier)
		rl.DrawText(fmt.Sprintf("%d%%", int(armorReduce[p.ArmorTier]*100)), hx+10*26+76, hy, 16, rl.LightGray)
	}
	if g.ShowMap {
		g.drawFullMap(sw, sh)
	}
	if g.ShowHelp {
		g.drawHelp(sw, sh)
	} else {
		rl.DrawText("H help   M map", sw-120, sh-24, 14, rl.LightGray)
	}
	rl.DrawFPS(sw-90, 20)
}

// drawFullMap shows the whole world map with everything marked.
func (g *Game) drawFullMap(sw, sh int32) {
	size := sh - 120
	mx, my := sw/2-size/2, int32(60)
	rl.DrawRectangle(0, 0, sw, sh, rl.NewColor(0, 0, 0, 120))
	rl.DrawRectangle(mx-6, my-6, size+12, size+12, rl.NewColor(30, 30, 34, 240))
	cell := float32(size) / worldW
	toMap := func(x, z float32) (float32, float32) {
		return float32(mx) + (x-originX)*cell, float32(my) + (z-originZ)*cell
	}
	if g.mapOK {
		rl.DrawTexturePro(g.mapTex, rl.NewRectangle(0, 0, worldW, worldD), rl.NewRectangle(float32(mx), float32(my), float32(size), float32(size)), rl.Vector2{}, 0, rl.White)
	}
	for _, a := range g.Animals {
		if a.Alive {
			x, y := toMap(a.Pos.X, a.Pos.Z)
			rl.DrawRectangle(int32(x)-2, int32(y)-2, 5, 5, rl.NewColor(240, 200, 220, 255))
		}
	}
	for _, e := range g.Enemies {
		if e.Alive {
			x, y := toMap(e.Pos.X, e.Pos.Z)
			rl.DrawCircle(int32(x), int32(y), 3+e.Spec.Radius*4, rl.Red)
		}
	}
	sx, sy := toMap(g.Spawn.X, g.Spawn.Z)
	rl.DrawRectangleLines(int32(sx)-5, int32(sy)-5, 10, 10, rl.SkyBlue)
	p := g.Player
	x, y := toMap(p.Pos.X, p.Pos.Z)
	f := p.FlatForward()
	rl.DrawLineEx(rl.NewVector2(x, y), rl.NewVector2(x+f.X*16, y+f.Z*16), 3, rl.White)
	rl.DrawCircle(int32(x), int32(y), 5, rl.White)
	rl.DrawText("MAP   white: you   blue: spawn   red: hostiles   pink: animals   M closes", mx, my+size+10, 16, rl.LightGray)
}

// drawHelp lists the controls over the game.
func (g *Game) drawHelp(sw, sh int32) {
	attack, use := "LEFT CLICK", "RIGHT CLICK"
	if settings.SwapButtons {
		attack, use = "RIGHT CLICK", "LEFT CLICK"
	}
	lines := []string{
		"WASD move   ARROWS look   CTRL sprint   SHIFT sneak   SPACE jump / swim / climb",
		"1-9 / wheel: hotbar   " + attack + ": shoot, swing, mine   " + use + ": place, eat, use bed   (B in pause swaps)",
		"E crafting   R reload   F5 third person   F11 fullscreen   ESC pause and settings",
		"Torches keep hostiles from rising nearby. Undead burn at sunrise. Creepers explode.",
		"Beds set your spawn point and skip the night. Dying drops your items where you fell.",
		"H closes this help",
	}
	w := int32(760)
	x0 := sw/2 - w/2
	y0 := sh/2 - 90
	rl.DrawRectangle(x0, y0, w, int32(len(lines))*26+20, rl.NewColor(0, 0, 0, 190))
	for i, l := range lines {
		rl.DrawText(l, x0+14, y0+12+int32(i)*26, 17, rl.White)
	}
}

// clock renders the in-game time as HH:MM (sunrise at 06:00).
func (g *Game) clock() string {
	t := g.Sky.T
	var hour float32
	if t < dayFrac {
		hour = 6 + t/dayFrac*14
	} else {
		hour = 20 + (t-dayFrac)/(1-dayFrac)*10
	}
	if hour >= 24 {
		hour -= 24
	}
	return fmt.Sprintf("%02d:%02d", int(hour), int((hour-float32(int(hour)))*60))
}

// refreshMinimap re-renders the top-down map texture when the world changed.
func (g *Game) refreshMinimap(dt float32) {
	g.mapCD -= dt
	w := g.World
	if g.mapOK && (g.mapVer == w.Version || g.mapCD > 0) {
		return
	}
	g.mapCD = 0.5
	g.mapVer = w.Version
	img := image.NewRGBA(image.Rect(0, 0, worldW, worldD))
	for lz := 0; lz < worldD; lz++ {
		for lx := 0; lx < worldW; lx++ {
			h := w.Height[lz*worldW+lx]
			b := w.getLocal(lx, h-1, lz)
			c := blocks[b].Top
			if b == Water {
				depth := float32(h-w.Ground[lz*worldW+lx]) / 8
				c = mix(rl.NewColor(60, 120, 220, 255), rl.NewColor(20, 40, 120, 255), depth)
			}
			s := 0.5 + 0.5*float32(h)/36
			c = mul(c, s)
			img.SetRGBA(lx, lz, color.RGBA{c.R, c.G, c.B, 255})
		}
	}
	ri := rl.NewImageFromImage(img)
	if g.mapOK {
		rl.UnloadTexture(g.mapTex)
	}
	g.mapTex = rl.LoadTextureFromImage(ri)
	rl.UnloadImage(ri)
	rl.SetTextureFilter(g.mapTex, rl.FilterPoint)
	g.mapOK = true
}

// drawMinimap draws the map texture in the top-right corner with entity markers.
func (g *Game) drawMinimap(sw int32) {
	const size = 192
	mx, my := sw-20-size, int32(50)
	cell := float32(size) / worldW
	toMap := func(x, z float32) (float32, float32) {
		return float32(mx) + (x-originX)*cell, float32(my) + (z-originZ)*cell
	}
	if g.mapOK {
		rl.DrawTexturePro(g.mapTex, rl.NewRectangle(0, 0, worldW, worldD), rl.NewRectangle(float32(mx), float32(my), size, size), rl.Vector2{}, 0, rl.NewColor(255, 255, 255, 235))
	}
	for _, e := range g.Enemies {
		if !e.Alive {
			continue
		}
		x, y := toMap(e.Pos.X, e.Pos.Z)
		rl.DrawCircle(int32(x), int32(y), 2+e.Spec.Radius*3, rl.Red)
	}
	p := g.Player
	x, y := toMap(p.Pos.X, p.Pos.Z)
	f := p.FlatForward()
	rl.DrawLineEx(rl.NewVector2(x, y), rl.NewVector2(x+f.X*10, y+f.Z*10), 2, rl.White)
	rl.DrawCircle(int32(x), int32(y), 3.5, rl.White)
	rl.DrawRectangleLines(mx, my, size, size, rl.NewColor(120, 120, 130, 255))
	// Compass heading under the map (north is -Z).
	deg := math.Atan2(float64(f.X), float64(-f.Z)) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	dirs := [...]string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	label := fmt.Sprintf("Facing %s  (%d, %d, %d)", dirs[int((deg+22.5)/45)%8], floorI(p.Pos.X), floorI(p.Pos.Y), floorI(p.Pos.Z))
	rl.DrawRectangle(mx, my+size+4, size, 22, rl.NewColor(0, 0, 0, 140))
	rl.DrawText(label, mx+6, my+size+8, 14, rl.LightGray)
}

// updateSettings handles the setting hotkeys on the pause screen.
func (g *Game) updateSettings() {
	changed := true
	switch {
	case rl.IsKeyPressed(rl.KeyLeftBracket):
		settings.Sensitivity = clamp(settings.Sensitivity-0.1, 0.2, 4)
	case rl.IsKeyPressed(rl.KeyRightBracket):
		settings.Sensitivity = clamp(settings.Sensitivity+0.1, 0.2, 4)
	case rl.IsKeyPressed(rl.KeyMinus):
		settings.Volume = clamp(settings.Volume-0.1, 0, 1)
	case rl.IsKeyPressed(rl.KeyEqual) || rl.IsKeyPressed(rl.KeyKpAdd):
		settings.Volume = clamp(settings.Volume+0.1, 0, 1)
		g.Audio.Play(g.Audio.Click, 0.8)
	case rl.IsKeyPressed(rl.KeyI):
		settings.InvertY = !settings.InvertY
	case rl.IsKeyPressed(rl.KeyB):
		settings.SwapButtons = !settings.SwapButtons
	case rl.IsKeyPressed(rl.KeyD):
		settings.Difficulty = (settings.Difficulty + 1) % 3
	case rl.IsKeyPressed(rl.KeyF11):
		toggleFullscreen()
	default:
		changed = false
	}
	if changed {
		settings.save()
	}
}

func toggleFullscreen() {
	settings.Fullscreen = !settings.Fullscreen
	rl.ToggleBorderlessWindowed()
}

func centered(text string, y, size int32, col rl.Color) {
	sw := int32(rl.GetScreenWidth())
	tw := rl.MeasureText(text, size)
	rl.DrawText(text, sw/2-tw/2, y, size, col)
}

func (g *Game) drawOverlay() {
	sw, sh := int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight())
	rl.DrawRectangle(0, 0, sw, sh, rl.NewColor(0, 0, 0, 170))
	switch g.State {
	case StateMenu:
		centered("BLOCKWORLD", sh/2-200, 72, rl.Gold)
		centered("Mine by day. Survive the night.", sh/2-120, 24, rl.LightGray)
		centered("WASD move   MOUSE or ARROW KEYS look   CTRL sprint   SHIFT sneak   SPACE jump / swim", sh/2-60, 19, rl.White)
		centered("1-9 / WHEEL pick hotbar item      LEFT CLICK mine, swing or shoot      RIGHT CLICK place", sh/2-32, 19, rl.White)
		centered("E crafting      R reload      ESC pause", sh/2-4, 19, rl.White)
		centered("Dig for coal, iron, gold and diamonds. Craft better pickaxes, swords and rifle ammo.", sh/2+40, 18, rl.LightGray)
		centered("Zombies, spiders and creepers rise at night. Undead burn at sunrise. Creepers explode.", sh/2+64, 18, rl.LightGray)
		centered("Press ENTER or CLICK to start a new world", sh/2+100, 28, rl.Lime)
		if saveExists() {
			centered("C  continue saved world", sh/2+134, 24, rl.SkyBlue)
		}
		centered(fmt.Sprintf("H  host your world for friends        J  join a friend's world        (you are %s)", playerName), sh/2+166, 22, rl.SkyBlue)
		if g.JoinErr != "" {
			centered(g.JoinErr, sh/2+196, 18, rl.Orange)
		}
		if g.HighScore > 0 {
			centered(fmt.Sprintf("High score  %d", g.HighScore), sh/2+226, 22, rl.Gold)
		}
	case StatePaused:
		centered("PAUSED", sh/2-120, 56, rl.White)
		centered("ESC resume    S save    Q save and quit to menu", sh/2-40, 22, rl.LightGray)
		centered("SETTINGS", sh/2+10, 24, rl.Gold)
		inv, fs := "off", "off"
		if settings.InvertY {
			inv = "on"
		}
		if settings.Fullscreen {
			fs = "on"
		}
		centered(fmt.Sprintf("[ / ]  mouse sensitivity  %.2f", settings.Sensitivity), sh/2+44, 20, rl.White)
		centered(fmt.Sprintf("- / +  volume  %d%%", int(settings.Volume*100+0.5)), sh/2+70, 20, rl.White)
		centered(fmt.Sprintf("I  invert mouse Y  %s", inv), sh/2+96, 20, rl.White)
		centered(fmt.Sprintf("F11  fullscreen  %s", fs), sh/2+122, 20, rl.White)
		swap := "left click mines, right click places"
		if settings.SwapButtons {
			swap = "right click mines, left click places"
		}
		centered("B  swap mouse buttons:  "+swap, sh/2+148, 20, rl.White)
		centered(fmt.Sprintf("D  difficulty  %s", difficultyNames[settings.Difficulty]), sh/2+174, 20, rl.White)
	case StateGameOver:
		centered("YOU DIED", sh/2-120, 64, rl.Red)
		if g.Player.Cause != "" {
			centered("You "+g.Player.Cause, sh/2-50, 22, rl.LightGray)
		}
		centered(fmt.Sprintf("Score %d    Nights survived %d    Kills %d    Deaths %d", g.Score, g.nightsSurvived(), g.Kills, g.Deaths), sh/2-20, 26, rl.White)
		if g.NewHigh {
			centered("NEW HIGH SCORE!", sh/2+16, 28, rl.Gold)
		} else {
			centered(fmt.Sprintf("High score  %d", g.HighScore), sh/2+16, 22, rl.Gold)
		}
		centered("ENTER respawn (items drop where you died)    N new world    Q menu", sh/2+60, 24, rl.Lime)
	}
}

// netTestStep drives the scripted multiplayer check; returns true when finished.
func (g *Game) netTestStep(role string, frame int) bool {
	p, w := g.Player, g.World
	if role == "host" {
		if frame%30 == 0 && frame > 100 {
			x, z := floorI(p.Pos.X)+rand.Intn(5)-2, floorI(p.Pos.Z)+rand.Intn(5)-2
			y := w.SurfaceY(x, z) - 1
			if y > 1 && w.Get(x, y, z) != Bedrock {
				g.breakBlock(x, y, z)
			}
		}
		if frame == 400 {
			// Creepers do not burn in daylight, so the client should still see them.
			for i := 0; i < 2; i++ {
				q := w.RandomFreePoint(p.Pos, 8)
				g.Enemies = append(g.Enemies, NewEnemy(q, KindCreeper, 1))
			}
		}
		if frame == 1500 {
			rl.TraceLog(rl.LogInfo, "NETTEST host: players=%d remotes=%d", g.Net.PlayerCount(), len(g.Remotes))
			return true
		}
		return false
	}
	// Client: mine the block under the crosshair through the host, then report.
	if frame == 200 {
		p.Pitch = -1.2
		p.Held = Item{Kind: ItemPickaxe}
		if a := w.RayCastAny(p.Eye(), p.Forward(), reachDist); a.Hit {
			g.sendToHost(&Msg{Break: &struct{ X, Y, Z int }{a.X, a.Y, a.Z}})
			g.netTestCell = [3]int{a.X, a.Y, a.Z}
			g.netTestWas = w.Get(a.X, a.Y, a.Z)
		}
		g.sendToHost(&Msg{Text: &struct{ Text string }{"hello from client"}})
	}
	if frame == 500 {
		c := g.netTestCell
		changed := w.Get(c[0], c[1], c[2]) != g.netTestWas
		ok := g.Net != nil && g.Net.Snaps > 40 && g.Net.Blocks > 0 && len(g.Remotes) == 1 && changed && len(g.Enemies) > 0
		rl.TraceLog(rl.LogInfo, "NETTEST client: ok=%v snaps=%d blocks=%d remotes=%d mined=%v enemies=%d drops=%d",
			ok, g.Net.Snaps, g.Net.Blocks, len(g.Remotes), changed, len(g.Enemies), len(g.Drops))
		return true
	}
	return false
}

// hostFromMenu hosts the saved world (or a new one) on the default port.
func (g *Game) hostFromMenu() {
	g.Net = &Net{Name: playerName}
	if saveExists() && g.load() {
		g.say("Hosting the saved world", 3)
	} else {
		g.Reset()
	}
	if err := g.StartHost(defaultPort); err != nil {
		g.State = StateMenu
		g.JoinErr = "Cannot host: " + err.Error()
		g.Net = nil
		return
	}
	g.Net.Name = playerName
	rl.DisableCursor()
}

// updateJoin edits the address and name boxes and connects on Enter.
func (g *Game) updateJoin() {
	field := &g.JoinText
	limit := 60
	if g.JoinField == 1 {
		field = &playerName
		limit = 16
	}
	for ch := rl.GetCharPressed(); ch > 0; ch = rl.GetCharPressed() {
		if ch >= 32 && ch < 127 && len(*field) < limit {
			*field += string(rune(ch))
		}
	}
	if (rl.IsKeyPressed(rl.KeyBackspace) || rl.IsKeyPressedRepeat(rl.KeyBackspace)) && len(*field) > 0 {
		*field = (*field)[:len(*field)-1]
	}
	if rl.IsKeyPressed(rl.KeyV) && (rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyLeftSuper)) {
		*field += rl.GetClipboardText()
	}
	if rl.IsKeyPressed(rl.KeyTab) {
		g.JoinField = 1 - g.JoinField
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		settings.Name = playerName
		settings.save()
		g.State = StateMenu
		return
	}
	if rl.IsKeyPressed(rl.KeyEnter) && g.JoinText != "" {
		if strings.TrimSpace(playerName) == "" {
			playerName = fmt.Sprintf("Player%d", rand.Intn(900)+100)
		}
		settings.Name = playerName
		addr := g.JoinText
		if !strings.Contains(addr, ":") {
			addr += defaultPort
		}
		g.JoinErr = "Connecting to " + addr + " ..."
		if err := g.Connect(addr, playerName); err != nil {
			g.JoinErr = "Could not join: " + err.Error()
			g.State = StateJoin
			return
		}
		settings.LastJoin = g.JoinText
		settings.save()
		g.JoinErr = ""
		rl.DisableCursor()
	}
}

func (g *Game) drawJoin() {
	sw, sh := int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight())
	rl.DrawRectangle(0, 0, sw, sh, rl.NewColor(0, 0, 0, 170))
	centered("JOIN A WORLD", sh/2-150, 48, rl.Gold)
	centered("Host address (port 7777 is assumed) and your name, then ENTER.  TAB switches boxes", sh/2-80, 20, rl.LightGray)
	w := int32(520)
	x := sw/2 - w/2
	caret := ""
	if int(rl.GetTime()*2)%2 == 0 {
		caret = "_"
	}
	box := func(y int32, label, text string, active bool) {
		rl.DrawText(label, x, y-20, 16, rl.LightGray)
		rl.DrawRectangle(x, y, w, 44, rl.NewColor(30, 30, 36, 255))
		col := rl.Gray
		if active {
			col = rl.White
			text += caret
		}
		rl.DrawRectangleLines(x, y, w, 44, col)
		rl.DrawText(text, x+12, y+10, 26, rl.White)
	}
	box(sh/2-30, "ADDRESS", g.JoinText, g.JoinField == 0)
	box(sh/2+50, "YOUR NAME", playerName, g.JoinField == 1)
	if g.JoinErr != "" {
		centered(g.JoinErr, sh/2+120, 20, rl.Orange)
	}
	centered("ESC back     CTRL+V paste", sh/2+160, 18, rl.LightGray)
}

// scriptedShots drives the screenshot session; returns true when finished.
func (g *Game) scriptedShots(frame int) bool {
	switch frame {
	case 18:
		rl.TakeScreenshot("shot_menu.png")
	case 20:
		g.State = StateJoin
		g.JoinText = "192.168.1.10:7777"
	case 26:
		rl.TakeScreenshot("shot_join.png")
	case 28:
		g.State = StateMenu
	case 30:
		g.Reset()
		g.Player.Pitch = 0.08
		g.Player.HoldBlock(Planks)
	case 100:
		g.Sky.Raining, g.Sky.Rain = true, 1
		g.ShowHelp = true
		for i, k := range []AnimalKind{AnimalPig, AnimalCow, AnimalSheep} {
			p := rl.Vector3Add(g.Player.Pos, rl.Vector3Scale(g.Player.FlatForward(), 4+float32(i)*1.5))
			p.X += float32(i)*2 - 2
			p.Y = float32(g.World.SurfaceY(floorI(p.X), floorI(p.Z)))
			g.Animals = append(g.Animals, NewAnimal(p, k))
		}
	case 120:
		rl.TakeScreenshot("shot_sky.png")
	case 130:
		g.Player.Pitch = -0.15
		g.ThirdPerson = true
		g.ShowHelp = false
		g.Sky.Raining, g.Sky.Rain = false, 0
		g.Player.ArmorTier = 2
	case 142:
		rl.TakeScreenshot("shot_third.png")
	case 145:
		g.ThirdPerson = false
	case 150:
		rl.TakeScreenshot("shot_day.png")
	case 160:
		g.Sky.T = 0.72
		g.Player.Pitch = -0.1
		g.Player.Held = Item{Kind: ItemSword}
		for i, k := range []EnemyKind{KindZombie, KindSkeleton, KindSpider, KindCreeper, KindBrute} {
			p := rl.Vector3Add(g.Player.Pos, rl.Vector3Scale(g.Player.FlatForward(), 5+float32(i)*2))
			p.X += float32(i)*1.5 - 2
			p.Y = float32(g.World.SurfaceY(floorI(p.X), floorI(p.Z)))
			g.Enemies = append(g.Enemies, NewEnemy(p, k, 1))
		}
	case 170:
		// Torches on the ground around the player.
		for _, d := range [][2]int{{2, 1}, {-2, 2}, {1, -3}, {4, 3}} {
			x, z := floorI(g.Player.Pos.X)+d[0], floorI(g.Player.Pos.Z)+d[1]
			g.World.Set(x, g.World.SurfaceY(x, z), z, Torch)
		}
	case 200:
		rl.TakeScreenshot("shot_night.png")
	case 210:
		g.State = StateCrafting
	case 230:
		rl.TakeScreenshot("shot_craft.png")
	case 240:
		// Into a cave: the nearest dark underground pocket, lit by one torch.
		g.State = StatePlaying
		p, w := g.Player, g.World
		for r := 2; r < 40; r++ {
			for dz := -r; dz <= r; dz++ {
				for dx := -r; dx <= r; dx++ {
					x, z := floorI(p.Pos.X)+dx, floorI(p.Pos.Z)+dz
					for y := 3; y < w.SurfaceY(x, z)-4; y++ {
						if w.Get(x, y, z) == Air && w.Get(x, y+1, z) == Air && w.Solid(x, y-1, z) && w.sunLocal(x-originX, y, z-originZ) == 0 {
							p.Pos = rl.NewVector3(float32(x)+0.5, float32(y), float32(z)+0.5)
							p.Pitch = -0.2
							p.Yaw = math.Pi / 2 // face +X
							w.Set(x, y, z, Torch)
							for i := 1; i < 7; i++ {
								for j := -2; j <= 2; j++ {
									for yy := y - 1; yy <= y+3; yy++ {
										w.Set(x+i, yy, z+j, Air)
									}
									if i > 1 {
										w.Set(x+i, y-1, z+j, Lava)
									}
								}
							}
							w.Set(x+1, y, z+2, Ladder)
							w.Set(x+1, y+1, z+2, Ladder)
							w.Set(x+1, y+2, z+2, Ladder)
							return false
						}
					}
				}
			}
		}
	case 300:
		rl.TakeScreenshot("shot_cave.png")
	case 310:
		return true
	}
	return false
}

func main() {
	hostAddr := flag.String("host", "", "host a world for others on this address, e.g. :7777")
	joinAddr := flag.String("join", "", "join a hosted world, e.g. 192.168.1.10:7777")
	name := flag.String("name", "", "your player name in multiplayer")
	flag.Parse()
	playerName = *name

	rl.SetConfigFlags(rl.FlagVsyncHint | rl.FlagWindowResizable)
	rl.InitWindow(1280, 720, "Blockworld")
	if !rl.IsWindowReady() {
		fmt.Fprintln(os.Stderr, "blockworld: could not open a window (is the display available and unlocked?)")
		os.Exit(1)
	}
	defer rl.CloseWindow()
	rl.SetTargetFPS(144)
	rl.SetExitKey(rl.KeyNull)
	settings = loadSettings()
	if playerName == "" {
		playerName = settings.Name
	}
	if playerName == "" {
		playerName = fmt.Sprintf("Player%d", rand.Intn(900)+100)
	}
	if settings.Fullscreen {
		rl.ToggleBorderlessWindowed()
	}

	g := NewGame()
	defer g.Audio.Close()
	g.Net = &Net{Name: *name}
	switch {
	case *hostAddr != "":
		if saveExists() && g.load() {
			g.say("Hosting the saved world", 3)
		} else {
			g.Reset()
		}
		if err := g.StartHost(*hostAddr); err != nil {
			fmt.Fprintln(os.Stderr, "blockworld: cannot host:", err)
			os.Exit(1)
		}
		g.Net.Name = *name
		rl.DisableCursor()
	case *joinAddr != "":
		if err := g.Connect(*joinAddr, *name); err != nil {
			fmt.Fprintln(os.Stderr, "blockworld: cannot join:", err)
			os.Exit(1)
		}
		rl.DisableCursor()
	default:
		g.Net = nil
	}

	// BLOCKWORLD_SHOTS=1 runs a short scripted session that saves screenshots
	// (day, night, crafting) into the working directory and exits; used for testing.
	shots := os.Getenv("BLOCKWORLD_SHOTS") != ""
	soak := os.Getenv("BLOCKWORLD_SOAK") != ""
	netTest := os.Getenv("BLOCKWORLD_NETTEST") // "host" or "client": scripted multiplayer check
	if netTest != "" {
		rl.SetTargetFPS(60)
		g.Net = &Net{Name: "Tester-" + netTest}
		if netTest == "host" {
			g.Reset()
			if err := g.StartHost("127.0.0.1:7799"); err != nil {
				fmt.Fprintln(os.Stderr, "NETTEST host failed:", err)
				os.Exit(1)
			}
		} else {
			var err error
			for i := 0; i < 40; i++ {
				if err = g.Connect("127.0.0.1:7799", "Tester-client"); err == nil {
					break
				}
				time.Sleep(500 * time.Millisecond)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, "NETTEST client failed:", err)
				os.Exit(1)
			}
		}
	}
	if soak {
		rl.SetTargetFPS(0)
	}
	frame := 0

	for !rl.WindowShouldClose() {
		dt := min(rl.GetFrameTime(), 0.05)
		if g.isHost() {
			g.HostTick()
		} else if g.isClient() {
			g.ClientTick()
		}
		if netTest != "" {
			frame++
			if g.netTestStep(netTest, frame) {
				break
			}
		}
		if shots || soak {
			frame++
			done := false
			if shots {
				done = g.scriptedShots(frame)
			} else {
				done = g.soak(frame)
			}
			if done {
				break
			}
		}

		switch g.State {
		case StateMenu:
			if rl.IsKeyPressed(rl.KeyEnter) || rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
				g.Reset()
				deleteSave()
				rl.DisableCursor()
			} else if rl.IsKeyPressed(rl.KeyC) && saveExists() {
				if g.load() {
					rl.DisableCursor()
				} else {
					g.State = StateMenu
				}
			} else if rl.IsKeyPressed(rl.KeyH) {
				g.hostFromMenu()
			} else if rl.IsKeyPressed(rl.KeyJ) {
				g.State = StateJoin
				g.JoinErr = ""
				if g.JoinText == "" {
					g.JoinText = settings.LastJoin
				}
			}
		case StateJoin:
			g.updateJoin()
		case StatePlaying:
			if rl.IsKeyPressed(rl.KeyF11) {
				toggleFullscreen()
				settings.save()
			}
			if rl.IsKeyPressed(rl.KeyEscape) {
				g.State = StatePaused
				rl.EnableCursor()
			} else if rl.IsKeyPressed(rl.KeyE) {
				g.State = StateCrafting
				g.CraftHover = -1
				rl.EnableCursor()
			} else {
				g.update(dt)
			}
		case StateCrafting:
			if rl.IsKeyPressed(rl.KeyEscape) || rl.IsKeyPressed(rl.KeyE) {
				g.State = StatePlaying
				rl.DisableCursor()
				rl.GetMouseDelta()
			} else {
				g.updateCrafting()
			}
		case StatePaused:
			g.updateSettings()
			if rl.IsKeyPressed(rl.KeyEscape) || rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
				g.State = StatePlaying
				rl.DisableCursor()
				rl.GetMouseDelta() // discard the jump from re-capturing
			} else if rl.IsKeyPressed(rl.KeyS) {
				if g.save() {
					g.say("World saved", 1.5)
				} else {
					g.say("Save failed", 1.5)
				}
				g.State = StatePlaying
				rl.DisableCursor()
				rl.GetMouseDelta()
			} else if rl.IsKeyPressed(rl.KeyQ) {
				g.leaveWorld()
				g.State = StateMenu
			}
		case StateGameOver:
			if rl.IsKeyPressed(rl.KeyEnter) {
				g.respawn()
				rl.DisableCursor()
				rl.GetMouseDelta()
			} else if rl.IsKeyPressed(rl.KeyN) {
				g.Reset()
				deleteSave()
				rl.DisableCursor()
			} else if rl.IsKeyPressed(rl.KeyQ) {
				g.save()
				g.State = StateMenu
			}
		}
		g.refreshMinimap(dt)

		rl.BeginDrawing()
		rl.ClearBackground(g.Sky.Color())
		g.draw3D()
		switch g.State {
		case StatePlaying:
			g.drawHUD()
		case StateCrafting:
			g.drawHUD()
			g.drawCrafting()
		case StateMenu:
			g.drawOverlay()
		case StateJoin:
			g.drawJoin()
		default:
			g.drawHUD()
			g.drawOverlay()
		}
		rl.EndDrawing()
	}
	if g.State != StateMenu && !shots && !soak {
		g.leaveWorld() // closing the window keeps the world
	}
}

// leaveWorld saves (host or solo) or disconnects (client).
func (g *Game) leaveWorld() {
	if g.isClient() {
		g.Net.server.conn.Close()
		g.Net = nil
		return
	}
	g.save()
	if g.isHost() {
		g.Net.listener.Close()
		g.Net.broadcast(&Msg{Leave: &struct{ ID uint32 }{0}}, 0)
		g.Net = nil
	}
}
