package main

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Chests store blocks, items and ammo. Right click to open; click a row to
// move a whole stack across, hold Shift to move eight. When you die your
// belongings go into a chest at the spot (if there is room for one).

type Chest struct {
	Items [numBlocks]int
	Ammo  int
}

type ChestKey [3]int

func chestKey(x, y, z int) ChestKey {
	return ChestKey{wrapX(x-originX) + originX, y, wrapZ(z-originZ) + originZ}
}

func (g *Game) chestAt(x, y, z int) *Chest {
	if g.Chests == nil {
		g.Chests = map[ChestKey]*Chest{}
	}
	k := chestKey(x, y, z)
	c, ok := g.Chests[k]
	if !ok {
		c = &Chest{}
		g.Chests[k] = c
	}
	return c
}

// breakChest spills a chest's contents as drops and forgets it.
func (g *Game) breakChest(x, y, z int) {
	k := chestKey(x, y, z)
	c, ok := g.Chests[k]
	if !ok {
		return
	}
	at := rl.NewVector3(float32(x)+0.5, float32(y)+0.6, float32(z)+0.5)
	for b := Block(1); b < numBlocks; b++ {
		if c.Items[b] > 0 {
			g.Drops = append(g.Drops, Drop{Pos: at, Vel: rl.NewVector3(0, 2.5, 0), Block: b, Count: c.Items[b]})
		}
	}
	if c.Ammo > 0 {
		g.spawnDrop(at, 0, c.Ammo)
	}
	delete(g.Chests, k)
}

// deathChest tries to stow the player's belongings in a chest where they fell.
// Returns false if no chest could be placed (the caller scatters drops instead).
func (g *Game) deathChest(at rl.Vector3) bool {
	p := g.Player
	w := g.World
	x, y, z := floorI(at.X), floorI(at.Y), floorI(at.Z)
	placed := false
	for dy := 0; dy < 3 && !placed; dy++ {
		for _, d := range [][2]int{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			cx, cy, cz := x+d[0], y+dy, z+d[1]
			if cy < 1 || cy >= worldH-1 {
				continue
			}
			if w.Get(cx, cy, cz) == Air && blocks[w.Get(cx, cy-1, cz)].Solid {
				w.Set(cx, cy, cz, ChestBlock)
				x, y, z = cx, cy, cz
				placed = true
				break
			}
		}
	}
	if !placed {
		return false
	}
	c := g.chestAt(x, y, z)
	for b := Block(1); b < numBlocks; b++ {
		c.Items[b] += p.Inv[b]
		p.Inv[b] = 0
	}
	c.Ammo += p.Ammo + p.Reserve
	p.Ammo, p.Reserve = 0, 0
	g.say(fmt.Sprintf("Your belongings are in a chest at %d, %d, %d", x, y, z), 5)
	return true
}

// ---------- the chest screen ----------

func (g *Game) openChest(x, y, z int) {
	if g.isClient() {
		g.say("Only the host can open chests (for now)", 1.5)
		return
	}
	g.OpenChest = g.chestAt(x, y, z)
	g.OpenChestPos = [3]int{x, y, z}
	g.State = StateChest
	g.CraftHover = -1
	rl.EnableCursor()
}

type chestRow struct {
	Block Block
	Count int
	Ammo  bool
}

func chestRows(items *[numBlocks]int, ammo int) []chestRow {
	var rows []chestRow
	if ammo > 0 {
		rows = append(rows, chestRow{Ammo: true, Count: ammo})
	}
	for b := Block(1); b < numBlocks; b++ {
		if items[b] > 0 {
			rows = append(rows, chestRow{Block: b, Count: items[b]})
		}
	}
	return rows
}

const chestRowH = 26

func (g *Game) chestLayout() (x0, y0, colW int32) {
	sw, sh := int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight())
	return sw/2 - 360, sh/2 - 220, 340
}

func (g *Game) updateChest() {
	if g.OpenChest == nil {
		g.State = StatePlaying
		rl.DisableCursor()
		return
	}
	if rl.IsKeyPressed(rl.KeyEscape) || rl.IsKeyPressed(rl.KeyE) {
		g.State = StatePlaying
		g.OpenChest = nil
		rl.DisableCursor()
		rl.GetMouseDelta()
		return
	}
	if !rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		return
	}
	p := g.Player
	c := g.OpenChest
	x0, y0, colW := g.chestLayout()
	m := rl.GetMousePosition()
	amount := func(have int) int {
		if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
			return min(8, have)
		}
		return have
	}
	row := (int32(m.Y) - y0 - 60) / chestRowH
	if int32(m.Y) < y0+60 || row < 0 {
		return
	}
	switch {
	case int32(m.X) >= x0 && int32(m.X) < x0+colW:
		// Player side: move into the chest.
		rows := chestRows(&p.Inv, p.Reserve)
		if int(row) < len(rows) {
			r := rows[row]
			if r.Ammo {
				n := amount(p.Reserve)
				p.Reserve -= n
				c.Ammo += n
			} else {
				n := amount(p.Inv[r.Block])
				p.Inv[r.Block] -= n
				c.Items[r.Block] += n
			}
			p.EnsureHeld()
			g.Audio.Play(g.Audio.Place, 0.4)
		}
	case int32(m.X) >= x0+colW+40 && int32(m.X) < x0+2*colW+40:
		rows := chestRows(&c.Items, c.Ammo)
		if int(row) < len(rows) {
			r := rows[row]
			if r.Ammo {
				n := amount(c.Ammo)
				c.Ammo -= n
				p.Reserve += n
			} else {
				n := amount(c.Items[r.Block])
				c.Items[r.Block] -= n
				p.Inv[r.Block] += n
			}
			p.EnsureHeld()
			g.Audio.Play(g.Audio.Pickup, 0.4)
		}
	}
}

func (g *Game) drawChest() {
	sw, sh := int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight())
	rl.DrawRectangle(0, 0, sw, sh, rl.NewColor(0, 0, 0, 150))
	x0, y0, colW := g.chestLayout()
	h := int32(470)
	rl.DrawRectangle(x0-10, y0-10, 2*colW+60, h+20, rl.NewColor(150, 115, 70, 255))
	rl.DrawRectangle(x0, y0, 2*colW+40, h, rl.NewColor(190, 160, 110, 255))
	rl.DrawText("CHEST", x0+16, y0+12, 26, rl.NewColor(50, 35, 20, 255))
	rl.DrawText("click a row to move the stack    SHIFT moves 8    E or ESC closes", x0+140, y0+20, 15, rl.NewColor(80, 60, 40, 255))
	m := rl.GetMousePosition()
	drawCol := func(x int32, title string, rows []chestRow) {
		rl.DrawText(title, x+8, y0+40, 16, rl.NewColor(60, 45, 30, 255))
		for i, r := range rows {
			y := y0 + 60 + int32(i)*chestRowH
			if y+chestRowH > y0+h {
				break
			}
			bg := rl.NewColor(170, 140, 95, 255)
			if int32(m.X) >= x && int32(m.X) < x+colW && int32(m.Y) >= y && int32(m.Y) < y+chestRowH {
				bg = rl.NewColor(230, 210, 170, 255)
			}
			rl.DrawRectangle(x, y, colW, chestRowH-2, bg)
			if r.Ammo {
				rl.DrawRectangle(x+8, y+6, 16, 12, rl.NewColor(240, 190, 50, 255))
				rl.DrawText(fmt.Sprintf("%d  rifle ammo", r.Count), x+32, y+5, 16, rl.NewColor(40, 30, 20, 255))
			} else {
				g.drawBlockIcon(r.Block, x+6, y+3, 20)
				rl.DrawText(fmt.Sprintf("%d  %s", r.Count, blocks[r.Block].Name), x+32, y+5, 16, rl.NewColor(40, 30, 20, 255))
			}
		}
		if len(rows) == 0 {
			rl.DrawText("(empty)", x+8, y0+64, 16, rl.NewColor(100, 85, 60, 255))
		}
	}
	drawCol(x0, "YOUR INVENTORY", chestRows(&g.Player.Inv, g.Player.Reserve))
	drawCol(x0+colW+40, "IN THE CHEST", chestRows(&g.OpenChest.Items, g.OpenChest.Ammo))
	rl.DrawText(">", x0+colW+12, y0+200, 30, rl.NewColor(60, 45, 30, 255))
	rl.DrawText("<", x0+colW+12, y0+240, 30, rl.NewColor(60, 45, 30, 255))
}
