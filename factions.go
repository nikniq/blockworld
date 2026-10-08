package main

import (
	"fmt"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Factions and flags. Two rival villages, Red and Blue, send warbands across
// the map to capture flags and raid each other. Control-point flags stand on
// the open land. The player founds a village by planting their own flag;
// beds near it fill with settlers, and the village can rally its guards.

const (
	FactionNone   = 0
	FactionRed    = 1
	FactionBlue   = 2
	FactionPlayer = 3
)

var factionNames = [...]string{"Neutral", "Red", "Blue", "Your"}
var factionColors = [...]rl.Color{rl.NewColor(230, 230, 230, 255), rl.NewColor(220, 50, 50, 255), rl.NewColor(50, 90, 230, 255), rl.NewColor(240, 200, 50, 255)}

type Flag struct {
	Pos      rl.Vector3 // block position of the post
	Faction  int
	Progress float32 // capture progress in seconds
	Capturer int
	Village  int // index into g.Villages, or -1 for a control point
}

type Village struct {
	Centre  rl.Vector3
	Faction int
	Flag    int // index into g.Flags
	RaidCD  float32
	GrowCD  float32
	Name    string
}

const captureTime = 8.0

// ---------- setup ----------

// setupFactions runs on a new world: picks Red and Blue from the generated
// villages (the two farthest apart), gives every village a flag in its square,
// and scatters neutral control points.
func (g *Game) setupFactions() {
	w := g.World
	g.Flags, g.Villages = nil, nil
	for i, c := range w.VillageCentres {
		g.Villages = append(g.Villages, Village{Centre: c, Faction: FactionNone, Flag: -1, Name: fmt.Sprintf("Village %d", i+1)})
	}
	if len(g.Villages) >= 2 {
		// Red is the first village; Blue the farthest from it.
		best, bi := float32(0), 1
		for i := 1; i < len(g.Villages); i++ {
			if d := WrapDist(g.Villages[0].Centre, g.Villages[i].Centre); d > best {
				best, bi = d, i
			}
		}
		g.Villages[0].Faction, g.Villages[0].Name = FactionRed, "Redfort"
		g.Villages[bi].Faction, g.Villages[bi].Name = FactionBlue, "Bluehaven"
	}
	for i := range g.Villages {
		v := &g.Villages[i]
		fp := g.placeFlagPost(floorI(v.Centre.X)+2, floorI(v.Centre.Z), v.Faction, i)
		v.Flag = fp
	}
	// Neutral control points on dry open ground away from villages.
	want := 2 + areaScale
	for try := 0; try < want*40 && want > 0; try++ {
		p := w.RandomFreePoint(g.Spawn, 30)
		near := false
		for _, v := range g.Villages {
			if WrapDist(v.Centre, p) < 28 {
				near = true
			}
		}
		for _, f := range g.Flags {
			if WrapDist(f.Pos, p) < 40 {
				near = true
			}
		}
		if near || w.Get(floorI(p.X), floorI(p.Y)-1, floorI(p.Z)) == Mud {
			continue
		}
		g.placeFlagPost(floorI(p.X), floorI(p.Z), FactionNone, -1)
		want--
	}
	g.assignVillagers()
}

// placeFlagPost sets a flag block on a cobble pad and records it.
func (g *Game) placeFlagPost(x, z, faction, village int) int {
	w := g.World
	y := w.SurfaceY(x, z)
	for dz := -1; dz <= 1; dz++ {
		for dx := -1; dx <= 1; dx++ {
			if b := w.Get(x+dx, y-1, z+dz); !blocks[b].Solid || b == Water {
				w.Set(x+dx, y-1, z+dz, Cobble)
			}
			for yy := y; yy < y+3; yy++ {
				if b := w.Get(x+dx, yy, z+dz); b != Air {
					w.Set(x+dx, yy, z+dz, Air)
				}
			}
		}
	}
	w.Set(x, y-1, z, StoneBrick)
	w.Set(x, y, z, FlagPost)
	w.Set(x+1, y, z+1, Torch)
	g.Flags = append(g.Flags, Flag{Pos: rl.NewVector3(float32(x)+0.5, float32(y), float32(z)+0.5), Faction: faction, Village: village})
	return len(g.Flags) - 1
}

// assignVillagers gives every villager the faction of the nearest village.
func (g *Game) assignVillagers() {
	for _, a := range g.Animals {
		if a.Kind != AnimalVillager {
			continue
		}
		a.Village = g.nearestVillage(a.Home)
		if a.Village >= 0 {
			a.Faction = g.Villages[a.Village].Faction
		}
	}
}

func (g *Game) nearestVillage(p rl.Vector3) int {
	best, bi := float32(40), -1
	for i, v := range g.Villages {
		if d := WrapDist(v.Centre, p); d < best {
			best, bi = d, i
		}
	}
	return bi
}

func (g *Game) flagAt(x, y, z int) *Flag {
	for i := range g.Flags {
		f := &g.Flags[i]
		if floorI(f.Pos.X) == x && floorI(f.Pos.Y) == y && floorI(f.Pos.Z) == z {
			return f
		}
	}
	return nil
}

// ---------- founding the player's village ----------

// foundVillage plants the player's flag: a new village that fills nearby beds with settlers.
func (g *Game) foundVillage(x, y, z int) {
	idx := len(g.Villages)
	g.Villages = append(g.Villages, Village{Centre: rl.NewVector3(float32(x)+0.5, float32(y), float32(z)+0.5), Faction: FactionPlayer, Name: "Your village", GrowCD: 5})
	g.World.Set(x, y, z, FlagPost)
	g.Flags = append(g.Flags, Flag{Pos: rl.NewVector3(float32(x)+0.5, float32(y), float32(z)+0.5), Faction: FactionPlayer, Village: idx})
	g.Villages[idx].Flag = len(g.Flags) - 1
	// A first settler arrives with the flag.
	s := g.spawnSettler(idx, rl.NewVector3(float32(x)+2.5, float32(g.World.SurfaceY(x+2, z)), float32(z)+0.5))
	s.Prof = ProfFarmer
	g.announce("You founded a village! Build houses with beds nearby and settlers will move in.", 5)
	g.Audio.Play(g.Audio.Clear, 0.8)
	g.unlock(AchFound)
}

func (g *Game) spawnSettler(village int, pos rl.Vector3) *Animal {
	v := NewAnimal(pos, AnimalVillager)
	v.Home = pos
	v.Village = village
	v.Faction = g.Villages[village].Faction
	v.Prof = Profession(rand.Intn(int(numProfessions)))
	v.Name = villagerNames[rand.Intn(len(villagerNames))]
	v.Quest = rand.Intn(len(quests))
	v.Walking = true
	g.Animals = append(g.Animals, v)
	return v
}

// growVillages fills unclaimed beds near each village flag with new villagers.
func (g *Game) growVillages(dt float32) {
	w := g.World
	for i := range g.Villages {
		v := &g.Villages[i]
		v.GrowCD -= dt
		if v.GrowCD > 0 || v.Faction == FactionNone {
			continue
		}
		v.GrowCD = 45 + rand.Float32()*30
		if g.Sky.IsNight() {
			continue
		}
		cx, cy, cz := floorI(v.Centre.X), floorI(v.Centre.Y), floorI(v.Centre.Z)
		for dz := -18; dz <= 18; dz++ {
			for dx := -18; dx <= 18; dx++ {
				for dy := -4; dy <= 6; dy++ {
					x, y, z := cx+dx, cy+dy, cz+dz
					if w.Get(x, y, z) != Bed {
						continue
					}
					bed := rl.NewVector3(float32(x)+0.5, float32(y), float32(z)+0.5)
					taken := false
					for _, a := range g.Animals {
						if a.Alive && a.Kind == AnimalVillager && WrapDist(a.Home, bed) < 1 {
							taken = true
							break
						}
					}
					if taken {
						continue
					}
					s := g.spawnSettler(i, bed)
					if v.Faction == FactionPlayer {
						g.say(s.Name+" the "+professionNames[s.Prof]+" has moved into your village", 3)
					}
					return
				}
			}
		}
	}
}

// ---------- war ----------

// raids sends warbands out from Red and Blue every so often.
func (g *Game) raids(dt float32) {
	if settings.Difficulty == 0 {
		return
	}
	for i := range g.Villages {
		v := &g.Villages[i]
		if v.Faction != FactionRed && v.Faction != FactionBlue {
			continue
		}
		v.RaidCD -= dt
		if v.RaidCD > 0 || g.Sky.IsNight() {
			continue
		}
		v.RaidCD = 150 + rand.Float32()*120
		if settings.Mode == ModeBattle {
			v.RaidCD = 60 + rand.Float32()*40
		}
		soldiers := 0
		for _, a := range g.Animals {
			if a.Alive && a.Kind == AnimalVillager && a.Warband && a.Faction == v.Faction {
				soldiers++
			}
		}
		if soldiers >= 6 && settings.Mode != ModeBattle || soldiers >= 12 {
			continue
		}
		target := g.raidTarget(v.Faction)
		if settings.Mode == ModeBattle && rand.Intn(2) == 0 {
			// Battle mode: half the warbands come straight for the player's nearest flag.
			best := float32(1e9)
			for i, f := range g.Flags {
				if f.Faction != v.Faction {
					if d := WrapDist(f.Pos, g.Player.Pos); d < best {
						best, target = d, i
					}
				}
			}
		}
		if target < 0 {
			continue
		}
		for k := 0; k < 3; k++ {
			pos := rl.Vector3Add(v.Centre, rl.NewVector3(float32(k)*1.5-1.5, 0, 3))
			pos.Y = float32(g.World.SurfaceY(floorI(pos.X), floorI(pos.Z)))
			s := g.spawnSettler(i, pos)
			s.Prof, s.Warband, s.Goal = ProfGuard, true, target
			s.Name = factionNames[v.Faction] + " soldier"
		}
		tf := g.Flags[target]
		where := "a control point"
		if tf.Village >= 0 {
			where = g.Villages[tf.Village].Name
		}
		if WrapDist(v.Centre, g.Player.Pos) < 80 || (tf.Village >= 0 && g.Villages[tf.Village].Faction == FactionPlayer) {
			g.announce(fmt.Sprintf("%s sends a warband toward %s!", v.Name, where), 4)
		}
	}
}

// raidTarget picks a flag the faction does not hold: the enemy's home village half the time.
func (g *Game) raidTarget(faction int) int {
	var enemyHome, others []int
	for i, f := range g.Flags {
		if f.Faction == faction {
			continue
		}
		if f.Village >= 0 && (g.Villages[f.Village].Faction == FactionRed || g.Villages[f.Village].Faction == FactionBlue) {
			enemyHome = append(enemyHome, i)
		} else {
			others = append(others, i)
		}
	}
	if len(enemyHome) > 0 && (rand.Intn(2) == 0 || len(others) == 0) {
		return enemyHome[rand.Intn(len(enemyHome))]
	}
	if len(others) > 0 {
		return others[rand.Intn(len(others))]
	}
	return -1
}

// warTick handles soldiers: march to the goal flag, fight enemies near it, pick the next.
func (g *Game) warTick(a *Animal, dt float32) {
	if a.Goal < 0 || a.Goal >= len(g.Flags) {
		a.Goal = g.raidTarget(a.Faction)
		if a.Goal < 0 {
			a.Warband = false
			return
		}
	}
	f := &g.Flags[a.Goal]
	a.BiteCD = max(0, a.BiteCD-dt)
	// Enemy within reach: fight.
	if foe := g.nearestFoe(a, 9); foe != nil {
		to := WrapDelta(foe.Pos, a.Pos)
		to.Y = 0
		d := rl.Vector3Length(to)
		if d > 0.01 {
			a.Heading = rl.Vector3Scale(to, 1/d)
		}
		a.Walking = d > 1.8
		a.WanderT = 0.4
		if d < 2.2 && a.BiteCD == 0 {
			a.BiteCD = 1.2
			g.burst(rl.Vector3Add(foe.Pos, rl.NewVector3(0, 1, 0)), rl.NewColor(255, 90, 90, 255), 6)
			if foe.Hit(3) {
				g.announce(foe.Name+" fell in battle", 2.5)
				g.VillagersLost++
			}
		}
		return
	}
	if p := g.Player; !settings.Creative && p.HP > 0 && WrapDist(p.Pos, a.Pos) < 6 && g.playerAtWarWith(a.Faction) {
		to := WrapDelta(p.Pos, a.Pos)
		to.Y = 0
		d := rl.Vector3Length(to)
		if d > 0.01 {
			a.Heading = rl.Vector3Scale(to, 1/d)
		}
		a.Walking = d > 1.8
		a.WanderT = 0.4
		if d < 2.2 && a.BiteCD == 0 {
			a.BiteCD = 1.2
			g.hurtTargetFrom(0, 8, "was slain by a "+factionNames[a.Faction]+" soldier", true, a.Pos, 5)
		}
		return
	}
	// March to the flag.
	to := WrapDelta(f.Pos, a.Pos)
	to.Y = 0
	d := rl.Vector3Length(to)
	if d > 2.5 {
		a.Heading = rl.Vector3Scale(to, 1/d)
		a.Walking = true
		a.WanderT = 0.5
		if a.VelY == 0 && rand.Float32() < 0.03 {
			a.VelY = 5 // hop over the odd obstacle
		}
		return
	}
	// At the flag: hold position (capture happens through presence).
	a.Walking = false
	a.WanderT = 0.5
	if f.Faction == a.Faction {
		a.Goal = -1 // ours now: find the next one
	}
}

// nearestFoe finds the closest villager of another faction within range.
func (g *Game) nearestFoe(a *Animal, within float32) *Animal {
	var best *Animal
	bd := within
	for _, o := range g.Animals {
		if o == a || !o.Alive || o.Kind != AnimalVillager || o.Faction == a.Faction || o.Faction == FactionNone {
			continue
		}
		if d := WrapDist(o.Pos, a.Pos); d < bd {
			best, bd = o, d
		}
	}
	return best
}

func (g *Game) playerAtWarWith(faction int) bool {
	return faction == FactionRed && g.WarRed || faction == FactionBlue && g.WarBlue
}

// captureTick advances flag captures by presence.
func (g *Game) captureTick(dt float32) {
	p := g.Player
	for i := range g.Flags {
		f := &g.Flags[i]
		present := map[int]int{}
		if p.HP > 0 && WrapDist(p.Pos, f.Pos) < 3.5 {
			present[FactionPlayer]++
		}
		for _, r := range g.Remotes {
			if WrapDist(r.Pos, f.Pos) < 3.5 {
				present[FactionPlayer]++
			}
		}
		for _, a := range g.Animals {
			if a.Alive && a.Kind == AnimalVillager && a.Faction != FactionNone && (a.Warband || a.Prof == ProfGuard) && WrapDist(a.Pos, f.Pos) < 3.5 {
				present[a.Faction]++
			}
		}
		if len(present) != 1 {
			f.Progress = max(0, f.Progress-dt)
			continue
		}
		var who int
		for k := range present {
			who = k
		}
		if who == f.Faction {
			f.Progress = 0
			continue
		}
		if f.Capturer != who {
			f.Capturer, f.Progress = who, 0
		}
		f.Progress += dt
		if f.Progress >= captureTime {
			g.captureFlag(i, who)
		}
	}
}

func (g *Game) captureFlag(i, who int) {
	f := &g.Flags[i]
	old := f.Faction
	f.Faction, f.Progress = who, 0
	where := "a control point"
	if f.Village >= 0 {
		v := &g.Villages[f.Village]
		v.Faction = who
		where = v.Name
		if who == FactionPlayer {
			v.Name = "Your village"
		}
		for _, a := range g.Animals {
			if a.Alive && a.Kind == AnimalVillager && a.Village == f.Village && !a.Warband {
				a.Faction = who
			}
		}
	}
	switch {
	case who == FactionPlayer:
		g.Score += 250
		g.announce(fmt.Sprintf("You captured %s!  +250", where), 4)
		g.Audio.Play(g.Audio.Clear, 0.8)
		if old == FactionRed {
			g.WarRed = true
		}
		if old == FactionBlue {
			g.WarBlue = true
		}
		g.unlock(AchFlag)
		if g.ownsAllFlags() {
			g.Score += 3000
			g.announce("Every flag flies your colours. You rule the land!  +3000", 6)
			g.unlock(AchConquer)
		}
	case old == FactionPlayer:
		g.announce(fmt.Sprintf("%s took %s from you!", factionNames[who], where), 4)
	default:
		if WrapDist(f.Pos, g.Player.Pos) < 100 {
			g.announce(fmt.Sprintf("%s captured %s", factionNames[who], where), 3)
		}
	}
}

func (g *Game) ownsAllFlags() bool {
	for _, f := range g.Flags {
		if f.Faction != FactionPlayer {
			return false
		}
	}
	return len(g.Flags) > 0
}

// rally sends the player's guards marching on the nearest enemy flag.
func (g *Game) rally(from *Flag) {
	target, best := -1, float32(1e9)
	for i, f := range g.Flags {
		if f.Faction == FactionPlayer {
			continue
		}
		if d := WrapDist(f.Pos, from.Pos); d < best {
			target, best = i, d
		}
	}
	if target < 0 {
		g.say("There is nothing left to conquer", 2)
		return
	}
	n := 0
	for _, a := range g.Animals {
		if a.Alive && a.Kind == AnimalVillager && a.Faction == FactionPlayer && a.Prof == ProfGuard && !a.Warband && WrapDist(a.Pos, from.Pos) < 30 {
			a.Warband, a.Goal = true, target
			n++
		}
	}
	if n == 0 {
		g.say("No guards to send. Settlers who are guards will answer the call.", 2.5)
		return
	}
	g.announce(fmt.Sprintf("%d of your guards march on the %s flag!", n, factionNames[g.Flags[target].Faction]), 4)
	g.Audio.Play(g.Audio.Wave, 0.7)
}

// flagCounts tallies flags per faction for the HUD.
func (g *Game) flagCounts() [4]int {
	var c [4]int
	for _, f := range g.Flags {
		c[f.Faction]++
	}
	return c
}

// ---------- drawing ----------

// drawFlags draws a waving banner on every flag post in the faction's colour.
func (g *Game) drawFlags(cam rl.Camera3D) {
	t := float32(rl.GetTime())
	for i := range g.Flags {
		f := &g.Flags[i]
		pos := Near(f.Pos, cam.Position)
		if WrapDist(pos, cam.Position) > 160 {
			continue
		}
		col := mul(factionColors[f.Faction], g.World.Luminance(rl.Vector3Add(pos, rl.NewVector3(0, 1, 0)), g.Sky.Light()))
		top := rl.NewVector3(pos.X, pos.Y+2.9, pos.Z)
		for k := 0; k < 4; k++ {
			wave := float32(mathSin(float64(t*3+float32(k)*0.8+float32(i)))) * 0.08 * float32(k)
			seg := rl.NewVector3(top.X+0.15+float32(k)*0.22, top.Y-0.25+wave, top.Z)
			rl.DrawCubeV(seg, rl.NewVector3(0.22, 0.5-float32(k)*0.03, 0.04), col)
		}
		if f.Progress > 0 {
			frac := f.Progress / captureTime
			rl.DrawCubeV(rl.NewVector3(pos.X, pos.Y+3.4, pos.Z), rl.NewVector3(1.2, 0.08, 0.08), rl.NewColor(0, 0, 0, 180))
			rl.DrawCubeV(rl.NewVector3(pos.X-(1-frac)*0.6, pos.Y+3.4, pos.Z), rl.NewVector3(frac*1.2, 0.1, 0.1), factionColors[f.Capturer])
		}
	}
}
