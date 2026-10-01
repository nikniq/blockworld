package main

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Ingredient struct {
	Block Block
	N     int
}

// Recipe turns inventory blocks into blocks, ammo or a better tool.
type Recipe struct {
	Name  string
	Out   Block
	Count int
	Ammo  int
	Sword int // tier granted
	Pick  int // tier granted
	Armor int // tier granted
	In    []Ingredient
}

var recipes = []Recipe{
	{Name: "Oak Planks x4", Out: Planks, Count: 4, In: []Ingredient{{Log, 1}}},
	{Name: "Planks x4 (birch)", Out: Planks, Count: 4, In: []Ingredient{{BirchLog, 1}}},
	{Name: "Torch x4", Out: Torch, Count: 4, In: []Ingredient{{CoalOre, 1}, {Planks, 1}}},
	{Name: "TNT", Out: TNT, Count: 1, In: []Ingredient{{Sand, 4}, {CoalOre, 4}}},
	{Name: "Ladder x4", Out: Ladder, Count: 4, In: []Ingredient{{Planks, 2}}},
	{Name: "Bed", Out: Bed, Count: 1, In: []Ingredient{{Planks, 3}, {Wool, 3}}},
	{Name: "Stone Bricks x4", Out: StoneBrick, Count: 4, In: []Ingredient{{Cobble, 4}}},
	{Name: "Glass x4", Out: Glass, Count: 4, In: []Ingredient{{Sand, 4}, {CoalOre, 1}}},
	{Name: "Gold Block", Out: GoldBlock, Count: 1, In: []Ingredient{{GoldOre, 2}, {CoalOre, 1}}},
	{Name: "Rifle ammo x16", Ammo: 16, In: []Ingredient{{IronOre, 1}, {CoalOre, 1}}},
	{Name: "Rifle ammo x6", Ammo: 6, In: []Ingredient{{Gravel, 3}, {CoalOre, 1}}},
	{Name: "Stone Sword", Sword: TierStone, In: []Ingredient{{Cobble, 2}, {Planks, 1}}},
	{Name: "Iron Sword", Sword: TierIron, In: []Ingredient{{IronOre, 2}, {CoalOre, 1}, {Planks, 1}}},
	{Name: "Diamond Sword", Sword: TierDiamond, In: []Ingredient{{DiamondOre, 2}, {Planks, 1}}},
	{Name: "Stone Pickaxe", Pick: TierStone, In: []Ingredient{{Cobble, 3}, {Planks, 2}}},
	{Name: "Iron Pickaxe", Pick: TierIron, In: []Ingredient{{IronOre, 3}, {CoalOre, 1}, {Planks, 2}}},
	{Name: "Diamond Pickaxe", Pick: TierDiamond, In: []Ingredient{{DiamondOre, 3}, {Planks, 2}}},
	{Name: "Leather Armour", Armor: 1, In: []Ingredient{{Leather, 5}}},
	{Name: "Iron Armour", Armor: 2, In: []Ingredient{{IronOre, 5}, {CoalOre, 1}}},
	{Name: "Diamond Armour", Armor: 3, In: []Ingredient{{DiamondOre, 5}}},
}

// Useful reports whether the recipe would still improve anything for the player.
func (r *Recipe) Useful(p *Player) bool {
	if r.Sword > 0 && p.SwordTier >= r.Sword {
		return false
	}
	if r.Pick > 0 && p.PickTier >= r.Pick {
		return false
	}
	if r.Armor > 0 && p.ArmorTier >= r.Armor {
		return false
	}
	return true
}

func (r *Recipe) CanCraft(p *Player) bool {
	if !r.Useful(p) {
		return false
	}
	for _, in := range r.In {
		if p.Inv[in.Block] < in.N {
			return false
		}
	}
	return true
}

func (r *Recipe) Craft(p *Player) {
	for _, in := range r.In {
		p.Inv[in.Block] -= in.N
	}
	switch {
	case r.Ammo > 0:
		p.Reserve += r.Ammo
	case r.Sword > 0:
		p.SwordTier = r.Sword
	case r.Pick > 0:
		p.PickTier = r.Pick
	case r.Armor > 0:
		p.ArmorTier = r.Armor
	default:
		p.Inv[r.Out] += r.Count
	}
	p.EnsureHeld()
}

func (r *Recipe) needs() string {
	s := ""
	for i, in := range r.In {
		if i > 0 {
			s += " + "
		}
		s += fmt.Sprintf("%d %s", in.N, blocks[in.Block].Name)
	}
	return s
}

var armorColors = [...]rl.Color{rl.Gray, rl.NewColor(150, 95, 55, 255), rl.NewColor(220, 220, 225, 255), rl.NewColor(90, 230, 225, 255)}

// drawArmorIcon draws a chestplate silhouette.
func drawArmorIcon(cx, cy, size float32, tier int) {
	c := armorColors[tier]
	rl.DrawRectangleV(rl.NewVector2(cx-size*0.4, cy-size*0.4), rl.NewVector2(size*0.8, size*0.3), c)
	rl.DrawRectangleV(rl.NewVector2(cx-size*0.3, cy-size*0.1), rl.NewVector2(size*0.6, size*0.5), c)
	rl.DrawRectangleV(rl.NewVector2(cx-size*0.12, cy-size*0.4), rl.NewVector2(size*0.24, size*0.14), rl.NewColor(0, 0, 0, 90))
}

const (
	craftRowH = 44
	craftW    = 640
)

func (g *Game) craftLayout() (x0, y0 int32) {
	sw, sh := int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight())
	return sw/2 - craftW/2, sh/2 - int32(len(recipes)*craftRowH)/2 - 20
}

// updateCrafting handles clicks and hotkeys on the crafting screen.
func (g *Game) updateCrafting() {
	x0, y0 := g.craftLayout()
	m := rl.GetMousePosition()
	g.CraftHover = -1
	if int32(m.X) >= x0 && int32(m.X) < x0+craftW {
		row := (int32(m.Y) - y0 - 50) / craftRowH
		if int32(m.Y) >= y0+50 && int(row) >= 0 && int(row) < len(recipes) {
			g.CraftHover = int(row)
		}
	}
	if g.CraftHover >= 0 && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		g.craft(g.CraftHover)
	}
}

func (g *Game) craft(i int) {
	r := &recipes[i]
	if !r.CanCraft(g.Player) {
		g.Audio.Play(g.Audio.Click, 0.6)
		return
	}
	r.Craft(g.Player)
	g.Audio.Play(g.Audio.Craft, 0.7)
	g.say("Crafted "+r.Name, 1.2)
}

func (g *Game) drawCrafting() {
	sw, sh := int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight())
	rl.DrawRectangle(0, 0, sw, sh, rl.NewColor(0, 0, 0, 150))
	x0, y0 := g.craftLayout()
	h := int32(len(recipes)*craftRowH) + 70
	rl.DrawRectangle(x0-10, y0-10, craftW+20, h+20, rl.NewColor(198, 198, 198, 255))
	rl.DrawRectangle(x0-6, y0-6, craftW+12, h+12, rl.NewColor(120, 120, 120, 255))
	rl.DrawRectangle(x0, y0, craftW, h, rl.NewColor(160, 160, 160, 255))
	rl.DrawText("CRAFTING", x0+16, y0+12, 28, rl.NewColor(50, 50, 50, 255))
	rl.DrawText("click a recipe    E or ESC to close", x0+craftW-330, y0+20, 16, rl.NewColor(70, 70, 70, 255))
	p := g.Player
	for i := range recipes {
		r := &recipes[i]
		y := y0 + 50 + int32(i*craftRowH)
		bg := rl.NewColor(140, 140, 140, 255)
		if i == g.CraftHover {
			bg = rl.NewColor(210, 210, 190, 255)
		}
		rl.DrawRectangle(x0+8, y, craftW-16, craftRowH-4, bg)
		name := rl.NewColor(40, 40, 40, 255)
		need := rl.NewColor(80, 80, 80, 255)
		switch {
		case !r.Useful(p):
			name = rl.NewColor(110, 110, 110, 255)
			need = rl.NewColor(120, 120, 120, 255)
		case r.CanCraft(p):
			name = rl.NewColor(20, 90, 20, 255)
		default:
			name = rl.NewColor(120, 40, 40, 255)
		}
		// Result icon.
		ix, iy := x0+16, y+6
		switch {
		case r.Ammo > 0:
			rl.DrawRectangle(ix+6, iy+8, 20, 16, rl.NewColor(240, 190, 50, 255))
		case r.Sword > 0:
			drawSwordIcon(float32(ix+16), float32(iy+16), 30, r.Sword)
		case r.Pick > 0:
			drawPickIcon(float32(ix+16), float32(iy+16), 30, r.Pick)
		case r.Armor > 0:
			drawArmorIcon(float32(ix+16), float32(iy+16), 28, r.Armor)
		default:
			g.drawBlockIcon(r.Out, ix, iy, 32)
		}
		label := r.Name
		if !r.Useful(p) {
			label += "  (owned)"
		}
		rl.DrawText(label, x0+60, y+6, 20, name)
		rl.DrawText(r.needs(), x0+60, y+26, 14, need)
	}
	// Inventory summary on the right.
	invX := x0 + craftW + 30
	if invX+200 < sw {
		rl.DrawRectangle(invX-10, y0-10, 220, h+20, rl.NewColor(160, 160, 160, 255))
		rl.DrawText("INVENTORY", invX, y0+8, 20, rl.NewColor(50, 50, 50, 255))
		rl.DrawText(fmt.Sprintf("Ammo %d", p.Ammo+p.Reserve), invX, y0+34, 16, rl.NewColor(60, 60, 60, 255))
		rl.DrawText(fmt.Sprintf("%s sword", tierNames[p.SwordTier]), invX, y0+52, 16, rl.NewColor(60, 60, 60, 255))
		rl.DrawText(fmt.Sprintf("%s pickaxe", tierNames[p.PickTier]), invX, y0+70, 16, rl.NewColor(60, 60, 60, 255))
		armour := "No armour"
		if p.ArmorTier > 0 {
			armour = armorNames[p.ArmorTier] + " armour"
		}
		rl.DrawText(armour, invX, y0+88, 16, rl.NewColor(60, 60, 60, 255))
		yy := y0 + 114
		for b := Block(1); b < numBlocks; b++ {
			if p.Inv[b] == 0 {
				continue
			}
			g.drawBlockIcon(b, invX, yy, 22)
			rl.DrawText(fmt.Sprintf("%d  %s", p.Inv[b], blocks[b].Name), invX+30, yy+3, 16, rl.NewColor(40, 40, 40, 255))
			yy += 26
			if yy > y0+h-20 {
				break
			}
		}
	}
}
