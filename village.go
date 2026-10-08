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
				if b := w.getLocal(cx+dx, h-1, cz+dz); b == Grass || b == Dirt || b == Log || b == Leaves || b == RedSand || b == EucLog || b == EucLeaves {
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
	w.VillageCentres = append(w.VillageCentres, rl.NewVector3(float32(cx+originX)+0.5, float32(base), float32(cz+originZ)+0.5))
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
	if w.getLocal(x, base-1, z) != RedSand {
		w.setLocal(x, base-1, z, Grass)
	}
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
		v.Village = -1
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
	if a.Warband {
		g.warTick(a, dt)
		return
	}
	if a.Prof == ProfGuard && a.Faction != FactionNone {
		// Home guards fight raiders of another faction that come near.
		if foe := g.nearestFoe(a, 8); foe != nil {
			a.BiteCD = max(0, a.BiteCD-dt)
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
				}
			}
			return
		}
	}
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
				g.VillagersLost++
			} else {
				a.Flee = 4 // run (guards use Flee as a short retreat too)
			}
			return
		}
	}
}

// ---------- conversation ----------

// A conversation is the villager's current line plus the choices the player
// can pick with the number keys or a click. Lines depend on profession, time
// of day, weather, what has happened lately and whether the villager likes you.

type Choice struct {
	Label string
	Do    func()
}

type Conversation struct {
	NPC     *Animal
	Line    string
	Choices []Choice
}

func pick(lines ...string) string { return lines[rand.Intn(len(lines))] }

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

// villagerMood summarises how the villager feels about things right now.
func (g *Game) villagerMood(a *Animal) string {
	switch {
	case g.VillagerGrudge > 0:
		return "angry"
	case a.HP < a.Spec.HP/2:
		return "hurt"
	case g.Sky.IsNight():
		return "nervous"
	case g.Sky.Rain > 0.5:
		return "damp"
	case a.QuestDone:
		return "grateful"
	}
	return "cheerful"
}

func (g *Game) talkTo(a *Animal) {
	g.Talking = a
	g.State = StateDialog
	rl.EnableCursor()
	a.TalkCount++
	g.converse(a, g.greeting(a))
}

// converse sets the line and the standard topic menu.
func (g *Game) converse(a *Animal, line string) {
	c := &Conversation{NPC: a, Line: line}
	if g.VillagerGrudge > 0 {
		c.Choices = []Choice{
			{"I am sorry.", func() { g.converse(a, pick("Sorry does not bring them back. Leave us be.", "Go. We will not forget.")) }},
			{"Goodbye.", g.endTalk},
		}
		g.Conv = c
		return
	}
	done, ready, _ := g.questStatus(a)
	work := "Any work for me?"
	if !done && a.QuestAccepted {
		work = "About the job..."
		if ready {
			work = "I have what you asked for."
		}
	} else if done {
		work = "Anything else I can do?"
	}
	c.Choices = []Choice{
		{"How are things in the village?", func() { g.converse(a, g.gossip(a)) }},
		{work, func() { g.questTalk(a) }},
		{"Tell me about this land.", func() { g.converse(a, g.lore(a)) }},
		{"Goodbye.", g.endTalk},
	}
	g.Conv = c
}

func (g *Game) endTalk() {
	g.State = StatePlaying
	g.Talking = nil
	g.Conv = nil
	rl.DisableCursor()
	rl.GetMouseDelta()
}

func (g *Game) greeting(a *Animal) string {
	name := playerName
	if g.Net != nil {
		name = g.Net.Name
	}
	switch g.villagerMood(a) {
	case "angry":
		return pick("You. You killed one of ours.", "Keep your distance, murderer.")
	case "hurt":
		return pick("Careful... I am not well. Something got its teeth in me.", "I need rest. The night was cruel.")
	case "nervous":
		return pick("You should be indoors! They come out at night.", "Keep your voice down. The dead have good ears.", "Is that a torch? Bless you, keep it lit.")
	case "damp":
		return pick("Wet enough for you? The crops like it, at least.", "Mind the lightning out there.")
	case "grateful":
		return pick(fmt.Sprintf("%s! Good to see you again.", name), "Our hero returns. What can I do for you?")
	}
	if a.TalkCount <= 1 {
		return pick(fmt.Sprintf("A stranger! I am %s, the %s here. You must be %s.", a.Name, professionNames[a.Prof], name),
			fmt.Sprintf("Welcome, traveller. %s is the name. I keep the village %s.", a.Name, map[Profession]string{ProfFarmer: "fed", ProfGuard: "safe", ProfLibrarian: "learned"}[a.Prof]))
	}
	return pick(fmt.Sprintf("Hello again, %s.", name), "Fine day for it.", "Back so soon?", fmt.Sprintf("%s! Pull up a chair.", name))
}

func (g *Game) gossip(a *Animal) string {
	alive, guards := 0, 0
	for _, v := range g.Animals {
		if v.Alive && v.Kind == AnimalVillager && WrapDist(v.Home, a.Home) < 40 {
			alive++
			if v.Prof == ProfGuard {
				guards++
			}
		}
	}
	lines := []string{fmt.Sprintf("There are %d of us left here, %d with a sword.", alive, guards)}
	if g.Night > 0 {
		lines = append(lines, fmt.Sprintf("We have lived through %d nights of this. Each one is longer than the last.", g.Night))
	}
	if g.VillagersLost > 0 {
		lines = append(lines, fmt.Sprintf("We buried %d since the dead started walking. Do not add to the count.", g.VillagersLost))
	}
	if a.Faction == FactionRed || a.Faction == FactionBlue {
		other := FactionBlue
		if a.Faction == FactionBlue {
			other = FactionRed
		}
		c := g.flagCounts()
		lines = append(lines, fmt.Sprintf("The %ss hold %d flags to our %d. Our warbands will see to that.", factionNames[other], c[other], c[a.Faction]))
	}
	if a.Faction == FactionPlayer {
		lines = append(lines, "Proud to live under your flag. Build us more houses and more will come.")
	}
	if g.traderAlive() {
		lines = append(lines, "That trader in the purple robe is back. His prices are robbery, but he has things we cannot make.")
	}
	if g.World.BeaconLit {
		lines = append(lines, "The beacon on the tower burns again! Grandmother said it never would.")
	} else if g.Sky.Day >= 2 {
		lines = append(lines, "Have you seen the old tower with the dead light on top? Nobody goes there anymore.")
	}
	if a.Prof == ProfFarmer {
		lines = append(lines, "Wheat is slow this year. If you find seeds, plant them in the sun.")
	}
	if g.Sky.Rain > 0.5 {
		lines = append(lines, "Rain again. The well will be full, the roads a mess.")
	}
	if settings.Difficulty == 2 {
		lines = append(lines, "They say the dead are fiercer in these parts. They say right.")
	}
	return pick(lines...)
}

func (g *Game) lore(a *Animal) string {
	switch a.Prof {
	case ProfFarmer:
		return pick("The soil here is good. Grass, dirt, a bit of sun, and seeds will take. Wheat makes bread, bread keeps you walking.",
			"Cows give leather, sheep give wool. A bed needs wool, so be kind to the sheep.",
			"Out past the forest the ground turns red and the trees go pale. Strange beasts hop about there.")
	case ProfGuard:
		return pick("Light keeps them away. Torches around the walls and nothing climbs out of the dark near your door.",
			"Under the hills there are old rooms with iron cages that breathe out zombies. Smash the cage and the breathing stops.",
			"The green ones that hiss: do not let them get close. Shoot them, or run.",
			"A closed door stops a zombie. It does not stop the thing that walks on every fifth night.")
	default:
		return pick("The tower was built by whoever was here before us. The light on top wants diamonds, three of them, so the book says.",
			"There are timbered tunnels in the rock, left by miners long gone. Crates still sit in them.",
			"The world is round, you know. Walk far enough in one direction and you will see your own back.",
			"Lightning seeks the high ground. Build low, or build in stone.")
	}
}

// questTalk handles offering, progress and hand-in for the villager's quest.
func (g *Game) questTalk(a *Animal) {
	q := &quests[a.Quest]
	done, ready, progress := g.questStatus(a)
	switch {
	case done:
		g.converse(a, pick("You have done enough for me. Ask the others; everyone here needs something.", "Nothing today, friend. Rest."))
	case !a.QuestAccepted:
		c := &Conversation{NPC: a, Line: q.Text + "  I can offer " + q.Reward + "."}
		c.Choices = []Choice{
			{"I will do it.", func() {
				a.QuestAccepted = true
				if q.Kills > 0 && a.QuestKills < 0 {
					a.QuestKills = g.Kills
				}
				g.converse(a, pick("Thank you. I will be here.", "Good. Come back when it is done."))
			}},
			{"Not now.", func() { g.converse(a, pick("Fair enough. The offer stands.", "Think it over.")) }},
		}
		g.Conv = c
	case ready:
		c := &Conversation{NPC: a, Line: pick("You have it? Hand it over then!", "Is that it? Let me see.")}
		c.Choices = []Choice{
			{"Here you go.", func() {
				if q.Need != Air {
					g.Player.Inv[q.Need] -= q.NeedN
				}
				q.Give(g.Player)
				g.Player.EnsureHeld()
				a.QuestDone = true
				g.Score += 300
				g.Audio.Play(g.Audio.Clear, 0.8)
				g.unlock(AchQuest)
				g.converse(a, pick("Wonderful! Here, as promised: "+q.Reward+".", "You are a marvel. Take this: "+q.Reward+"."))
			}},
			{"Not yet.", func() { g.converse(a, "No hurry.") }},
		}
		g.Conv = c
	default:
		g.converse(a, pick("How goes it? "+progress+".", "Still at it? "+progress+". I will wait."))
	}
}

// updateDialog takes a choice by number key or click.
func (g *Game) updateDialog() {
	a := g.Talking
	if a == nil || !a.Alive || g.Conv == nil {
		g.endTalk()
		return
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		g.endTalk()
		return
	}
	c := g.Conv
	for i, k := range []int32{rl.KeyOne, rl.KeyTwo, rl.KeyThree, rl.KeyFour} {
		if i < len(c.Choices) && rl.IsKeyPressed(k) {
			c.Choices[i].Do()
			return
		}
	}
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		x, y, w := g.dialogRect()
		m := rl.GetMousePosition()
		for i := range c.Choices {
			ry := y + 100 + int32(i)*26
			if int32(m.X) >= x && int32(m.X) < x+w && int32(m.Y) >= ry && int32(m.Y) < ry+24 {
				c.Choices[i].Do()
				return
			}
		}
	}
}

func (g *Game) dialogRect() (x, y, w int32) {
	sw, sh := int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight())
	w = 680
	return sw/2 - w/2, sh - 250, w
}

func (g *Game) drawDialog() {
	a := g.Talking
	c := g.Conv
	if a == nil || c == nil {
		return
	}
	x, y, w := g.dialogRect()
	h := int32(110 + len(c.Choices)*26)
	rl.DrawRectangle(x-6, y-6, w+12, h+12, rl.NewColor(120, 100, 70, 255))
	rl.DrawRectangle(x, y, w, h, rl.NewColor(40, 34, 28, 240))
	title := fmt.Sprintf("%s the %s", a.Name, professionNames[a.Prof])
	rl.DrawText(title, x+16, y+12, 24, rl.Gold)
	rl.DrawText("("+g.villagerMood(a)+")", x+20+rl.MeasureText(title, 24), y+18, 16, rl.Gray)
	drawWrapped(c.Line, x+16, y+44, w-32, 19, rl.White)
	m := rl.GetMousePosition()
	for i, ch := range c.Choices {
		ry := y + 100 + int32(i)*26
		col := rl.NewColor(200, 220, 255, 255)
		if int32(m.X) >= x && int32(m.X) < x+w && int32(m.Y) >= ry && int32(m.Y) < ry+24 {
			col = rl.Gold
		}
		rl.DrawText(fmt.Sprintf("%d  %s", i+1, ch.Label), x+24, ry+4, 18, col)
	}
}

// ambientChatter gives nearby villagers something to say over their heads now and then.
func (g *Game) ambientChatter(dt float32) {
	p := g.Player
	for _, a := range g.Animals {
		if !a.Alive || a.Kind != AnimalVillager {
			continue
		}
		a.BubbleT = max(0, a.BubbleT-dt)
		a.BubbleCD -= dt
		if a.BubbleCD > 0 || WrapDist(a.Pos, p.Pos) > 9 {
			continue
		}
		a.BubbleCD = 14 + rand.Float32()*20
		var lines []string
		switch {
		case g.VillagerGrudge > 0:
			lines = []string{"Murderer.", "Stay away from my family."}
		case g.Sky.IsNight():
			lines = []string{"Lock your door tonight.", "Did you hear that?", "Keep to the torchlight."}
		case g.Sky.Rain > 0.5:
			lines = []string{"Good for the crops.", "My roof leaks again."}
		case a.Prof == ProfFarmer:
			lines = []string{"Wheat's coming along.", "Seen my sheep?", "Lovely morning."}
		case a.Prof == ProfGuard:
			lines = []string{"All quiet.", "Stay sharp.", "I counted eight of them last night."}
		default:
			lines = []string{"Have you read about the tower?", "Three diamonds, the book says.", "Fascinating weather."}
		}
		if !a.QuestDone && !a.QuestAccepted && rand.Float32() < 0.4 {
			lines = []string{"I could use a hand.", "Got a moment?"}
		}
		a.Bubble = lines[rand.Intn(len(lines))]
		a.BubbleT = 4
	}
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
