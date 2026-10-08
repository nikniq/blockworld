package main

import (
	"os"
	"path/filepath"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Achievements are small goals that pop up a toast when first reached. They
// persist across worlds in the user config directory.

type AchID int

const (
	AchFirstBlock AchID = iota
	AchCoal
	AchIron
	AchDiamond
	AchBuilder
	AchEat
	AchFirstNight
	AchNightFive
	AchCreeper
	AchGiant
	AchDiamondPick
	AchSleep
	AchDungeon
	AchSwim
	AchTrade
	AchQuest
	AchBeacon
	AchWolf
	AchFish
	AchMeteor
	AchCat
	AchHorse
	numAch
)

type achievement struct {
	Name string
	Desc string
}

var achievements = [numAch]achievement{
	AchFirstBlock:  {"Getting Wood", "Break your first block"},
	AchCoal:        {"Light It Up", "Pick up coal ore"},
	AchIron:        {"Acquire Hardware", "Pick up iron ore"},
	AchDiamond:     {"DIAMONDS!", "Pick up diamond ore"},
	AchBuilder:     {"Builder", "Place 100 blocks"},
	AchEat:         {"Dinner Time", "Eat something"},
	AchFirstNight:  {"Survivor", "Live through the first night"},
	AchNightFive:   {"Veteran", "Live through night five"},
	AchCreeper:     {"Sssss...", "Kill a creeper before it explodes"},
	AchGiant:       {"Giant Slayer", "Bring down a giant"},
	AchDiamondPick: {"Best Tools", "Craft a diamond pickaxe"},
	AchSleep:       {"Sweet Dreams", "Sleep through a night"},
	AchDungeon:     {"Tomb Raider", "Destroy a monster spawner"},
	AchSwim:        {"Deep End", "Go for a swim"},
	AchTrade:       {"Haggler", "Trade with the wandering trader"},
	AchQuest:       {"Good Neighbour", "Complete a villager's quest"},
	AchBeacon:      {"Conqueror", "Light the ancient beacon"},
	AchWolf:        {"Best Friend", "Tame a wolf"},
	AchFish:        {"Gone Fishing", "Catch a fish"},
	AchMeteor:      {"Sky Fall", "Witness an asteroid strike"},
	AchCat:         {"Crazy Cat Person", "Tame a cat with a fish"},
	AchHorse:       {"Saddle Up", "Tame and ride a horse"},
}

type Toast struct {
	Text string
	Sub  string
	T    float32
}

func achievementsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "blockworld", "achievements.txt")
}

func loadAchievements() [numAch]bool {
	var got [numAch]bool
	p := achievementsPath()
	if p == "" {
		return got
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return got
	}
	for _, line := range strings.Split(string(b), "\n") {
		for i := range achievements {
			if strings.TrimSpace(line) == achievements[i].Name {
				got[i] = true
			}
		}
	}
	return got
}

func saveAchievements(got [numAch]bool) {
	p := achievementsPath()
	if p == "" || os.Getenv("BLOCKWORLD_SHOTS") != "" || os.Getenv("BLOCKWORLD_SOAK") != "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	var sb strings.Builder
	for i, ok := range got {
		if ok {
			sb.WriteString(achievements[i].Name + "\n")
		}
	}
	_ = os.WriteFile(p, []byte(sb.String()), 0o644)
}

// unlock awards an achievement once, with a toast and a sound.
func (g *Game) unlock(id AchID) {
	if g.Headless || g.Got[id] {
		return
	}
	g.Got[id] = true
	g.Toasts = append(g.Toasts, Toast{Text: "Achievement: " + achievements[id].Name, Sub: achievements[id].Desc})
	g.Audio.Play(g.Audio.Clear, 0.6)
	g.Score += 100
	saveAchievements(g.Got)
}

func (g *Game) achievementCount() int {
	n := 0
	for _, ok := range g.Got {
		if ok {
			n++
		}
	}
	return n
}

// checkAchievements evaluates the goals that are easiest to test from state.
func (g *Game) checkAchievements() {
	p := g.Player
	if p.Inv[CoalOre] > 0 {
		g.unlock(AchCoal)
	}
	if p.Inv[IronOre] > 0 {
		g.unlock(AchIron)
	}
	if p.Inv[DiamondOre] > 0 {
		g.unlock(AchDiamond)
	}
	if g.PlacedCount >= 100 {
		g.unlock(AchBuilder)
	}
	if p.PickTier == TierDiamond {
		g.unlock(AchDiamondPick)
	}
	if p.HeadWater {
		g.unlock(AchSwim)
	}
}

func (g *Game) updateToasts(dt float32) {
	keep := g.Toasts[:0]
	for _, t := range g.Toasts {
		t.T += dt
		if t.T < 5 {
			keep = append(keep, t)
		}
	}
	g.Toasts = keep
}

func (g *Game) drawToasts(sw int32) {
	if len(g.Toasts) == 0 {
		return
	}
	t := g.Toasts[0]
	slide := float32(0)
	if t.T < 0.4 {
		slide = (0.4 - t.T) / 0.4
	} else if t.T > 4.5 {
		slide = (t.T - 4.5) / 0.5
	}
	w := int32(360)
	x := sw - 20 - w + int32(slide*float32(w+30))
	y := int32(260)
	rl.DrawRectangle(x, y, w, 60, rl.NewColor(30, 30, 36, 235))
	rl.DrawRectangleLines(x, y, w, 60, rl.Gold)
	rl.DrawText(t.Text, x+14, y+10, 20, rl.Gold)
	rl.DrawText(t.Sub, x+14, y+36, 16, rl.LightGray)
}

// drawAchievementList shows every goal and which are done (K key).
func (g *Game) drawAchievementList(sw, sh int32) {
	w := int32(520)
	x := sw/2 - w/2
	y := int32(120)
	rl.DrawRectangle(x, y, w, int32(numAch)*26+50, rl.NewColor(0, 0, 0, 200))
	rl.DrawText("ACHIEVEMENTS", x+14, y+10, 22, rl.Gold)
	for i := range achievements {
		c := rl.Gray
		mark := "[ ]"
		if g.Got[i] {
			c = rl.Lime
			mark = "[x]"
		}
		rl.DrawText(mark+" "+achievements[i].Name, x+14, y+44+int32(i)*26, 18, c)
		rl.DrawText(achievements[i].Desc, x+230, y+46+int32(i)*26, 15, rl.LightGray)
	}
}
