package main

import (
	"fmt"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Villages: clusters of small houses around a well, each with a bed, a torch
// and a door. One villager lives per bed. Villagers wander near home by day,
// head indoors at night, flee hostiles (guards fight them), and offer quests.

// ---------- generation ----------

// placeVillages builds a few villages on flat grassland.
func (w *World) placeVillages(heights []int) {
	placed := 0
	for try := 0; try < 3000 && placed < 3*areaScale; try++ {
		cx, cz := rand.Intn(worldW-30)+15, rand.Intn(worldD-30)+15
		if dx, dz := cx+originX, cz+originZ; dx*dx+dz*dz < 400 {
			continue // keep the spawn clearing free
		}
		// Reasonably flat, dry grassland within a 24x24 area (houses level their own plots).
		lo, hi := worldH, 0
		grass, samples := 0, 0
		for dz := -12; dz <= 12; dz += 3 {
			for dx := -12; dx <= 12; dx += 3 {
				h := heights[wrapZ(cz+dz)*worldW+wrapX(cx+dx)]
				lo, hi = min(lo, h), max(hi, h)
				samples++
				if b := w.getLocal(cx+dx, h-1, cz+dz); b == Grass || b == Dirt || b == Log || b == Leaves {
					grass++
				}
			}
		}
		if hi-lo > 7 || lo <= seaLevel+1 || grass*10 < samples*6 {
			continue
		}
		base := (lo + hi) / 2
		w.buildVillage(cx, cz, base)
		placed++
	}
}

func (w *World) buildVillage(cx, cz, base int) {
	// Well in the middle.
	for dz := -1; dz <= 1; dz++ {
		for dx := -1; dx <= 1; dx++ {
			w.flattenColumn(cx+dx, cz+dz, base)
			if dx == 0 && dz == 0 {
				w.setLocal(cx, base-1, cz, Water)
				w.setLocal(cx, base-2, cz, Water)
				w.setLocal(cx, base-3, cz, Cobble)
			} else {
				w.setLocal(cx+dx, base, cz+dz, Cobble)
			}
		}
	}
	w.setLocal(cx-1, base+1, cz-1, Torch)
	// Houses on a ring around the well, facing inward.
	n := 4 + rand.Intn(3)
	for i := 0; i < n; i++ {
		ang := float64(i) / float64(n) * 6.2832
		hx := cx + int(11*cosf(ang))
		hz := cz + int(11*sinf(ang))
		// Face toward the well along the dominant axis.
		fx, fz := 0, 0
		if abs(cx-hx) >= abs(cz-hz) {
			fx = sign(float32(cx - hx))
		} else {
			fz = sign(float32(cz - hz))
		}
		w.buildHouse(hx, hz, base, fx, fz)
	}
	// A few torches on the paths.
	for i := 0; i < 6; i++ {
		ang := rand.Float64() * 6.2832
		tx, tz := cx+int(6*cosf(ang)), cz+int(6*sinf(ang))
		if w.getLocal(tx, base, tz) == Air && blocks[w.getLocal(tx, base-1, tz)].Solid {
			w.setLocal(tx, base, tz, Torch)
		}
	}
}

func cosf(a float64) float64 { return float64(float32(mathCos(a))) }
func sinf(a float64) float64 { return float64(float32(mathSin(a))) }

// flattenColumn makes the column solid up to base-1 and clear above, so houses sit on level ground.
func (w *World) flattenColumn(x, z, base int) {
	for y := base - 4; y < base; y++ {
		if b := w.getLocal(x, y, z); b == Air || b.Liquid() || blocks[b].Tiny || b == Log || b == Leaves {
			w.setLocal(x, y, z, Dirt)
		}
	}
	w.setLocal(x, base-1, z, Grass)
	for y := base; y < base+7; y++ {
		w.setLocal(x, y, z, Air)
	}
	w.recomputeHeight(x, z)
}

// buildHouse builds a 7x6 plank house with log corners; the door faces (fx, fz).
func (w *World) buildHouse(x, z, base, fx, fz int) {
	const hw, hd, hh = 3, 3, 3 // half width, half depth, wall height
	for dz := -hd - 1; dz <= hd+1; dz++ {
		for dx := -hw - 1; dx <= hw+1; dx++ {
			w.flattenColumn(x+dx, z+dz, base)
		}
	}
	wall := Planks
	if rand.Intn(3) == 0 {
		wall = Cobble
	}
	for dz := -hd; dz <= hd; dz++ {
		for dx := -hw; dx <= hw; dx++ {
			edge := abs(dx) == hw || abs(dz) == hd
			corner := abs(dx) == hw && abs(dz) == hd
			for y := base; y < base+hh; y++ {
				switch {
				case corner:
					w.setLocal(x+dx, y, z+dz, Log)
				case edge:
					b := wall
					if y == base+1 && (dx == 0 || dz == 0) && !(dx == fx*hw && dz == fz*hd) {
						b = Glass // a window in the middle of each wall
					}
					w.setLocal(x+dx, y, z+dz, b)
				default:
					w.setLocal(x+dx, y, z+dz, Air)
				}
			}
			// Floor and roof.
			w.setLocal(x+dx, base-1, z+dz, Planks)
			w.setLocal(x+dx, base+hh, z+dz, Planks)
			if abs(dx) < hw && abs(dz) < hd {
				w.setLocal(x+dx, base+hh+1, z+dz, Planks) // raised roof centre
			}
		}
	}
	// Door in the middle of the front wall, with a torch beside it outside.
	dx, dz := fx*hw, fz*hd
	w.setLocal(x+dx, base, z+dz, DoorClosed)
	w.setLocal(x+dx, base+1, z+dz, Air)
	w.setLocal(x+dx+fx, base, z+dz+fz, Air)
	w.setLocal(x+dx+fx+fz, base, z+dz+fz+fx, Torch)
	// Bed against the back wall, torch inside, sometimes a crate.
	w.setLocal(x-fx*(hw-1), base, z-fz*(hd-1), Bed)
	w.setLocal(x+fz*(hw-1), base+1, z+fx*(hd-1), Torch)
	if rand.Intn(2) == 0 {
		w.setLocal(x-fz*(hw-1), base, z-fx*(hd-1), Crate)
	}
	for dz := -hd - 1; dz <= hd+1; dz++ {
		for dx := -hw - 1; dx <= hw+1; dx++ {
			w.recomputeHeight(x+dx, z+dz)
		}
	}
}

// ---------- villagers ----------

type Profession int

const (
	ProfFarmer Profession = iota
	ProfGuard
	ProfLibrarian
	numProfessions
)

var professionNames = [...]string{"Farmer", "Guard", "Librarian"}
var villagerNames = []string{"Ada", "Bram", "Cora", "Dov", "Elin", "Finn", "Greta", "Hal", "Ida", "Joss", "Kai", "Lena", "Milo", "Nell", "Otto", "Pia", "Quin", "Rosa", "Sven", "Tess", "Uma", "Vik", "Wren", "Yara"}

// Quest is a villager's request and reward.
type Quest struct {
	Text   string
	Need   Block // item to bring (Air: a kill quest)
	NeedN  int
	Kills  int // hostiles to slay
	Reward string
	Give   func(p *Player)
}

var quests = []Quest{
	{Text: "The mill is idle. Bring me 8 wheat and I will pay in rifle ammo.", Need: WheatItem, NeedN: 8, Reward: "24 rifle ammo", Give: func(p *Player) { p.Reserve += 24 }},
	{Text: "We need stone for the well. 12 cobblestone, please.", Need: Cobble, NeedN: 12, Reward: "3 bread and 4 apples", Give: func(p *Player) { p.Inv[Bread] += 3; p.Inv[Apple] += 4 }},
	{Text: "My boots are worn through. Bring 4 leather.", Need: Leather, NeedN: 4, Reward: "iron armour", Give: func(p *Player) { p.ArmorTier = max(p.ArmorTier, 2) }},
	{Text: "The dead come every night. Slay 5 hostiles for us.", Kills: 5, Reward: "2 diamond ore", Give: func(p *Player) { p.Inv[DiamondOre] += 2 }},
	{Text: "The lamps are dark. 6 coal ore would light the whole village.", Need: CoalOre, NeedN: 6, Reward: "16 arrows and a bow", Give: func(p *Player) { p.Inv[ArrowItem] += 16; p.Inv[Bow] = max(p.Inv[Bow], 1) }},
	{Text: "I collect curiosities. Find me 2 gold ore.", Need: GoldOre, NeedN: 2, Reward: "2 TNT", Give: func(p *Player) { p.Inv[TNT] += 2 }},
	{Text: "A giant stalks the hills. Kill 10 hostiles and prove your mettle.", Kills: 10, Reward: "a diamond sword", Give: func(p *Player) { p.SwordTier = max(p.SwordTier, TierDiamond) }},
}

// spawnVillagers puts one villager at every bed in the world (new worlds only).
func (g *Game) spawnVillagers() {
	w := g.World
	for i, b := range w.Blocks {
		if b != Bed {
			continue
		}
		x := i%worldW + originX
		z := (i/worldW)%worldD + originZ
		y := i / (worldW * worldD)
		home := rl.NewVector3(float32(x)+0.5, float32(y), float32(z)+0.5)
		v := NewAnimal(home, AnimalVillager)
		v.Home = home
		v.Prof = Profession(rand.Intn(int(numProfessions)))
		v.Name = villagerNames[rand.Intn(len(villagerNames))]
		v.Quest = rand.Intn(len(quests))
		v.Walking = true
		g.Animals = append(g.Animals, v)
	}
}

// villagerTick runs the villager-specific behaviour: stay near home, go in at
// night, run from hostiles, and (guards) strike hostiles that come close.
func (g *Game) villagerTick(a *Animal, dt float32) {
	w := g.World
	// Nearest hostile.
	var threat *Enemy
	td := float32(99)
	for _, e := range g.Enemies {
		if e.Alive {
			if d := WrapDist(e.Pos, a.Pos); d < td {
				threat, td = e, d
			}
		}
	}
	if a.Prof == ProfGuard {
		a.BiteCD = max(0, a.BiteCD-dt)
		if threat != nil && td < 2.2 && a.BiteCD == 0 {
			a.BiteCD = 1.2
			a.Heading = rl.Vector3Normalize(WrapDelta(threat.Pos, a.Pos))
			a.Heading.Y = 0
			g.burst(rl.Vector3Add(threat.Pos, rl.NewVector3(0, 1, 0)), rl.NewColor(255, 90, 90, 255), 6)
			if threat.Hit(3) {
				g.killEnemyRaw(threat)
			}
		}
		if threat != nil && td < 8 && a.Flee == 0 {
			// Advance on the threat.
			a.Heading = rl.Vector3Normalize(WrapDelta(threat.Pos, a.Pos))
			a.Heading.Y = 0
			a.Walking = true
			a.WanderT = 0.5
			return
		}
	} else if threat != nil && td < 7 {
		// Run home, or just away.
		away := WrapDelta(a.Pos, threat.Pos)
		away.Y = 0
		if l := rl.Vector3Length(away); l > 0.01 {
			a.Heading = rl.Vector3Scale(away, 1/l)
		}
		a.Walking = true
		a.WanderT = 0.6
		return
	}
	// Night: head home and stay there. Day: wander, but not too far from home.
	toHome := WrapDelta(a.Home, a.Pos)
	toHome.Y = 0
	dist := rl.Vector3Length(toHome)
	if g.Sky.IsNight() {
		if dist > 2 {
			a.Heading = rl.Vector3Scale(toHome, 1/dist)
			a.Walking = true
			a.WanderT = 0.5
		} else {
			a.Walking = false
			a.WanderT = 1
		}
		return
	}
	if dist > 14 && a.WanderT <= 0.6 {
		a.Heading = rl.Vector3Scale(toHome, 1/dist)
		a.Walking = true
		a.WanderT = 2
	}
	// Keep villagers out of water.
	if w.WaterAt(a.Pos) {
		a.Heading = rl.Vector3Scale(toHome, 1/max(dist, 0.1))
		a.Walking = true
	}
}

// villagerTarget encodes a villager as a hostile target id.
const villagerIDBit = uint32(1) << 30

func (g *Game) villagerTargets() []Target {
	g.assignIDs() // villagers need stable ids before they can be targeted
	var ts []Target
	for _, a := range g.Animals {
		if a.Alive && a.Kind == AnimalVillager {
			ts = append(ts, Target{ID: villagerIDBit | a.ID, Pos: a.Pos, Eye: rl.Vector3Add(a.Pos, rl.NewVector3(0, 1.5, 0)), Box: a.BB()})
		}
	}
	return ts
}

func (g *Game) hurtVillager(id uint32, amount int, from rl.Vector3) {
	for _, a := range g.Animals {
		if a.Alive && a.Kind == AnimalVillager && a.ID == (id&^villagerIDBit) {
			if a.Hit(amount) {
				g.announce(a.Name+" the "+professionNames[a.Prof]+" was slain", 3)
			} else {
				a.Flee = 4 // run (guards use Flee as a short retreat too)
			}
			return
		}
	}
}

// ---------- dialogue ----------

// questStatus reports what the player can do about a villager's quest.
func (g *Game) questStatus(a *Animal) (done bool, ready bool, progress string) {
	q := &quests[a.Quest]
	if a.QuestDone {
		return true, false, ""
	}
	p := g.Player
	if q.Kills > 0 {
		have := g.Kills - a.QuestKills
		if a.QuestKills < 0 {
			have = 0
		}
		return false, have >= q.Kills, fmt.Sprintf("%d of %d slain", min(have, q.Kills), q.Kills)
	}
	return false, p.Inv[q.Need] >= q.NeedN, fmt.Sprintf("you have %d of %d %s", p.Inv[q.Need], q.NeedN, blocks[q.Need].Name)
}

func (g *Game) talkTo(a *Animal) {
	g.Talking = a
	g.State = StateDialog
	rl.EnableCursor()
	q := &quests[a.Quest]
	if q.Kills > 0 && a.QuestKills < 0 {
		a.QuestKills = g.Kills // start counting from now
	}
}

func (g *Game) updateDialog() {
	a := g.Talking
	if a == nil || !a.Alive {
		g.State = StatePlaying
		rl.DisableCursor()
		return
	}
	if rl.IsKeyPressed(rl.KeyEscape) || rl.IsKeyPressed(rl.KeyE) {
		g.State = StatePlaying
		g.Talking = nil
		rl.DisableCursor()
		rl.GetMouseDelta()
		return
	}
	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		done, ready, _ := g.questStatus(a)
		if !done && ready {
			q := &quests[a.Quest]
			if q.Need != Air {
				g.Player.Inv[q.Need] -= q.NeedN
			}
			q.Give(g.Player)
			g.Player.EnsureHeld()
			a.QuestDone = true
			g.Score += 300
			g.say("Quest complete: "+q.Reward+"  +300", 3)
			g.Audio.Play(g.Audio.Clear, 0.8)
			g.unlock(AchQuest)
		}
	}
}

func (g *Game) drawDialog() {
	a := g.Talking
	if a == nil {
		return
	}
	sw, sh := int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight())
	w := int32(640)
	x := sw/2 - w/2
	y := sh - 230
	rl.DrawRectangle(x-6, y-6, w+12, 192, rl.NewColor(120, 100, 70, 255))
	rl.DrawRectangle(x, y, w, 180, rl.NewColor(40, 34, 28, 240))
	title := fmt.Sprintf("%s the %s", a.Name, professionNames[a.Prof])
	rl.DrawText(title, x+16, y+12, 24, rl.Gold)
	q := &quests[a.Quest]
	done, ready, progress := g.questStatus(a)
	var line, hint string
	switch {
	case done:
		line = "Thank you again, friend. The village is in your debt."
		hint = "ESC  leave"
	case ready:
		line = q.Text
		hint = "ENTER  hand it over for " + q.Reward + "        ESC  not now"
	default:
		line = q.Text
		hint = progress + "        reward: " + q.Reward + "        ESC  leave"
	}
	drawWrapped(line, x+16, y+48, w-32, 20, rl.White)
	rl.DrawText(hint, x+16, y+150, 16, rl.LightGray)
}

// drawWrapped draws text broken into lines that fit the width.
func drawWrapped(text string, x, y, width, size int32, col rl.Color) {
	words := splitWords(text)
	line := ""
	for _, wd := range words {
		try := wd
		if line != "" {
			try = line + " " + wd
		}
		if rl.MeasureText(try, size) > width && line != "" {
			rl.DrawText(line, x, y, size, col)
			y += size + 6
			line = wd
		} else {
			line = try
		}
	}
	if line != "" {
		rl.DrawText(line, x, y, size, col)
	}
}

func splitWords(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ' ' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
