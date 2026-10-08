package main

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Trades offered by the wandering trader: give the first, get the second.
type Trade struct {
	Give  Block
	GiveN int
	Get   Block
	GetN  int
	Ammo  int // rifle ammo instead of a block
}

var trades = []Trade{
	{Give: GoldOre, GiveN: 2, Ammo: 24},
	{Give: IronOre, GiveN: 6, Get: DiamondOre, GetN: 1},
	{Give: Wool, GiveN: 4, Get: Bread, GetN: 3},
	{Give: CoalOre, GiveN: 8, Get: TNT, GetN: 1},
	{Give: Leather, GiveN: 3, Get: Apple, GetN: 4},
	{Give: GoldOre, GiveN: 1, Get: Torch, GetN: 12},
	{Give: DiamondOre, GiveN: 1, Get: ArrowItem, GetN: 24},
	{Give: WheatItem, GiveN: 6, Get: Seeds, GetN: 12},
	{Give: Amethyst, GiveN: 3, Get: DiamondOre, GetN: 1},
}

func (t *Trade) can(p *Player) bool { return p.Inv[t.Give] >= t.GiveN }

func (t *Trade) apply(p *Player) {
	p.Inv[t.Give] -= t.GiveN
	if t.Ammo > 0 {
		p.Reserve += t.Ammo
	} else {
		p.Inv[t.Get] += t.GetN
	}
	p.EnsureHeld()
}

func (t *Trade) label() string {
	if t.Ammo > 0 {
		return fmt.Sprintf("%d %s   for   %d rifle ammo", t.GiveN, blocks[t.Give].Name, t.Ammo)
	}
	return fmt.Sprintf("%d %s   for   %d %s", t.GiveN, blocks[t.Give].Name, t.GetN, blocks[t.Get].Name)
}

func (g *Game) tradeLayout() (x0, y0 int32) {
	sw, sh := int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight())
	return sw/2 - 300, sh/2 - int32(len(trades)*craftRowH)/2 - 20
}

func (g *Game) updateTrade() {
	x0, y0 := g.tradeLayout()
	m := rl.GetMousePosition()
	g.CraftHover = -1
	if int32(m.X) >= x0 && int32(m.X) < x0+600 {
		row := (int32(m.Y) - y0 - 50) / craftRowH
		if int32(m.Y) >= y0+50 && int(row) >= 0 && int(row) < len(trades) {
			g.CraftHover = int(row)
		}
	}
	if g.CraftHover >= 0 && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		t := &trades[g.CraftHover]
		if t.can(g.Player) {
			t.apply(g.Player)
			g.Audio.Play(g.Audio.Pickup, 0.8)
			g.say("Traded", 1)
			g.unlock(AchTrade)
		} else {
			g.Audio.Play(g.Audio.Click, 0.6)
		}
	}
}

func (g *Game) drawTrade() {
	sw, sh := int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight())
	rl.DrawRectangle(0, 0, sw, sh, rl.NewColor(0, 0, 0, 150))
	x0, y0 := g.tradeLayout()
	h := int32(len(trades)*craftRowH) + 70
	rl.DrawRectangle(x0-10, y0-10, 620, h+20, rl.NewColor(120, 100, 150, 255))
	rl.DrawRectangle(x0, y0, 600, h, rl.NewColor(160, 150, 175, 255))
	rl.DrawText("WANDERING TRADER", x0+16, y0+12, 28, rl.NewColor(40, 30, 60, 255))
	rl.DrawText("click a trade    ESC to close", x0+330, y0+20, 16, rl.NewColor(70, 60, 80, 255))
	p := g.Player
	for i := range trades {
		t := &trades[i]
		y := y0 + 50 + int32(i*craftRowH)
		bg := rl.NewColor(140, 130, 155, 255)
		if i == g.CraftHover {
			bg = rl.NewColor(210, 200, 220, 255)
		}
		rl.DrawRectangle(x0+8, y, 584, craftRowH-4, bg)
		c := rl.NewColor(120, 40, 40, 255)
		if t.can(p) {
			c = rl.NewColor(20, 90, 20, 255)
		}
		g.drawBlockIcon(t.Give, x0+16, y+6, 32)
		if t.Ammo > 0 {
			rl.DrawRectangle(x0+562, y+14, 20, 16, rl.NewColor(240, 190, 50, 255))
		} else {
			g.drawBlockIcon(t.Get, x0+552, y+6, 32)
		}
		rl.DrawText(t.label(), x0+60, y+12, 20, c)
		rl.DrawText(fmt.Sprintf("you have %d", p.Inv[t.Give]), x0+380, y+14, 14, rl.NewColor(70, 60, 80, 255))
	}
}
