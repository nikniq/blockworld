package main

import (
	"fmt"
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type State int

const (
	StateMenu State = iota
	StatePlaying
	StatePaused
	StateGameOver
)

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

type PickupKind int

const (
	PickHealth PickupKind = iota
	PickAmmo
)

type Pickup struct {
	Pos  rl.Vector3
	Kind PickupKind
}

type Game struct {
	State    State
	World    *World
	Nav      *NavGrid
	Audio    *Audio
	Player   *Player
	Enemies  []*Enemy
	Pickups  []Pickup
	Tracers  []Tracer
	Sparks   []Spark
	Wave     int
	WaveCD   float32 // countdown before next wave
	Score    int
	Kills    int
	HitMark  float32
	PickupCD float32
	Msg      string
	MsgT     float32
	Flash    float32 // muzzle flash timer

	HighScore int
	NewHigh   bool
}

func NewGame() *Game {
	g := &Game{Audio: NewAudio(), HighScore: loadHighScore()}
	g.Reset()
	g.State = StateMenu
	return g
}

// Reset generates a fresh world and starts a run.
func (g *Game) Reset() {
	g.World = NewWorld()
	g.Nav = NewNavGrid(g.World)
	g.Player = NewPlayer(g.World.SpawnPoint())
	g.Enemies = nil
	g.Pickups = nil
	g.Tracers = nil
	g.Sparks = nil
	g.Wave = 0
	g.WaveCD = 4
	g.Score = 0
	g.Kills = 0
	g.PickupCD = 6
	g.NewHigh = false
	g.State = StatePlaying
	g.say("Get ready...  (2 = build mode)", 4)
}

func (g *Game) say(s string, t float32) { g.Msg, g.MsgT = s, t }

func (g *Game) spawnWave() {
	g.Wave++
	n := 3 + g.Wave*2
	for i := 0; i < n; i++ {
		p := g.World.RandomFreePoint(g.Player.Pos, 14)
		g.Enemies = append(g.Enemies, NewEnemy(p, g.pickKind(i, n), g.Wave))
	}
	g.say(fmt.Sprintf("WAVE %d  -  %d hostiles", g.Wave, n), 2.5)
	g.Audio.Play(g.Audio.Wave, 0.8)
}

// pickKind chooses an enemy type for slot i of n in the current wave.
// Runners appear from wave 3, brutes from wave 5 (at least one per wave).
func (g *Game) pickKind(i, n int) EnemyKind {
	if g.Wave >= 5 && i == n-1 {
		return KindBrute
	}
	r := rand.Float32()
	if g.Wave >= 5 && r < 0.12 {
		return KindBrute
	}
	if g.Wave >= 3 && r < 0.4 {
		return KindRunner
	}
	return KindGrunt
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

	wallD := g.World.RayDistance(ray)
	bestD := wallD
	var target *Enemy
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
	end := rl.Vector3Add(eye, rl.Vector3Scale(dir, min(bestD, 200)))
	// Tracer starts from a "muzzle" slightly right/below the eye.
	muzzle := rl.Vector3Add(eye, rl.Vector3Add(rl.Vector3Scale(p.Right(), 0.25), rl.NewVector3(0, -0.2, 0)))
	muzzle = rl.Vector3Add(muzzle, rl.Vector3Scale(dir, 0.6))
	g.Tracers = append(g.Tracers, Tracer{muzzle, end, 0.06})
	g.Flash = 0.05
	g.Audio.Play(g.Audio.Shoot, 0.9)

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
			g.Kills++
			pts := target.Spec.Points
			if headshot {
				pts += pts / 2
				g.say("HEADSHOT", 0.8)
			}
			g.Score += pts
			g.Audio.Play(g.Audio.Die, 0.7)
		}
		g.burst(end, col, 10)
	} else if !math.IsInf(float64(wallD), 1) {
		g.burst(end, rl.NewColor(220, 220, 200, 255), 6)
	}
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

// updateBuilder handles mining (hold left click) and placing (right click).
func (g *Game) updateBuilder(dt float32) {
	p, w := g.Player, g.World
	p.Aim = w.RayCast(p.Eye(), p.Forward(), reachDist)
	if !p.Aim.Hit || !w.InBounds(p.Aim.X, p.Aim.Y, p.Aim.Z) {
		p.Aim.Hit = false
		p.Mining = false
		return
	}
	a := p.Aim
	if rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		b := w.Get(a.X, a.Y, a.Z)
		info := &blocks[b]
		if info.MineTime >= 0 {
			if !p.Mining {
				p.Mining = true
				p.MineT = 0
			}
			p.MineT += dt
			p.Swing = 1
			if p.MineT >= info.MineTime {
				w.Set(a.X, a.Y, a.Z, Air)
				p.Inv[info.Drops]++
				p.EnsurePlace()
				p.Mining = false
				centre := rl.NewVector3(float32(a.X)+0.5, float32(a.Y)+0.5, float32(a.Z)+0.5)
				g.burst(centre, info.Side, 12)
				g.Audio.Play(g.Audio.Dig, 0.8)
			}
		} else {
			p.Mining = false
		}
	} else {
		p.Mining = false
	}
	if rl.IsMouseButtonPressed(rl.MouseButtonRight) && p.Place != Air && p.Inv[p.Place] > 0 {
		x, y, z := a.X+a.NX, a.Y+a.NY, a.Z+a.NZ
		if w.InBounds(x, y, z) && w.Get(x, y, z) == Air && !g.blockOccupied(x, y, z) {
			w.Set(x, y, z, p.Place)
			p.Inv[p.Place]--
			p.EnsurePlace()
			p.Swing = 1
			g.Audio.Play(g.Audio.Place, 0.7)
		}
	}
}

func (g *Game) spawnPickup() {
	if len(g.Pickups) >= 4 {
		return
	}
	k := PickAmmo
	if rand.Float32() < 0.45 {
		k = PickHealth
	}
	g.Pickups = append(g.Pickups, Pickup{g.World.RandomFreePoint(g.Player.Pos, 6), k})
}

func (g *Game) update(dt float32) {
	p := g.Player
	wasReloading := p.Reloading > 0
	p.Update(dt, g.World)
	if p.Tool == ToolRifle {
		p.Mining = false
		if p.TryFire() {
			g.fire()
		} else if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && p.Ammo == 0 && p.Reserve == 0 {
			g.Audio.Play(g.Audio.Click, 0.6)
		}
	} else {
		g.updateBuilder(dt)
	}
	if !wasReloading && p.Reloading > 0 {
		g.Audio.Play(g.Audio.Reload, 0.8)
	}

	// Enemies.
	g.Nav.Update(p.Pos, g.World)
	for _, e := range g.Enemies {
		if d := e.Update(dt, g.World, g.Nav, p, g.Enemies); d > 0 {
			p.Damage(d)
			g.Audio.Play(g.Audio.Hurt, 0.9)
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

	// Waves.
	if g.aliveEnemies() == 0 {
		if g.WaveCD > 0 {
			g.WaveCD -= dt
			if g.WaveCD <= 0 {
				g.spawnWave()
			}
		} else {
			g.WaveCD = 5
			g.say(fmt.Sprintf("Wave %d cleared!", g.Wave), 3)
			g.Score += 250 * g.Wave
			g.Audio.Play(g.Audio.Clear, 0.7)
		}
	}

	// Pickups.
	g.PickupCD -= dt
	if g.PickupCD <= 0 {
		g.PickupCD = 7
		g.spawnPickup()
	}
	keep := g.Pickups[:0]
	for _, pk := range g.Pickups {
		flat := rl.Vector3Distance(rl.NewVector3(pk.Pos.X, 0, pk.Pos.Z), rl.NewVector3(p.Pos.X, 0, p.Pos.Z))
		if flat < 1.1 && math.Abs(float64(p.Pos.Y-pk.Pos.Y)) < 1.5 {
			switch pk.Kind {
			case PickHealth:
				p.HP = min(maxHealth, p.HP+35)
				g.say("+35 HEALTH", 1)
			case PickAmmo:
				p.Reserve += 24
				g.say("+24 AMMO", 1)
			}
			g.Audio.Play(g.Audio.Pickup, 0.7)
			continue
		}
		keep = append(keep, pk)
	}
	g.Pickups = keep

	// Effects.
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
	g.MsgT = max(0, g.MsgT-dt)

	if p.HP <= 0 {
		g.State = StateGameOver
		rl.EnableCursor()
		g.Audio.Play(g.Audio.GameOver, 0.9)
		if g.Score > g.HighScore {
			g.HighScore = g.Score
			g.NewHigh = true
			saveHighScore(g.Score)
		}
	}
}

func (g *Game) draw3D() {
	cam := g.Player.Camera()
	rl.BeginMode3D(cam)
	g.World.Draw(cam)
	for _, e := range g.Enemies {
		e.Draw()
	}
	t := float32(rl.GetTime())
	for _, pk := range g.Pickups {
		y := 0.6 + float32(math.Sin(float64(t*3)))*0.1
		pos := rl.NewVector3(pk.Pos.X, pk.Pos.Y+y, pk.Pos.Z)
		col := rl.NewColor(60, 200, 90, 255)
		if pk.Kind == PickAmmo {
			col = rl.NewColor(240, 180, 40, 255)
		}
		rl.DrawCube(pos, 0.6, 0.6, 0.6, col)
		rl.DrawCubeWires(pos, 0.6, 0.6, 0.6, rl.White)
		// Beacon so pickups are visible across the map.
		rl.DrawCylinder(rl.NewVector3(pk.Pos.X, pk.Pos.Y, pk.Pos.Z), 0.05, 0.05, 12, 6, rl.Fade(col, 0.35))
	}
	// Block selection outline and crack progress.
	p := g.Player
	if p.Tool == ToolBuilder && p.Aim.Hit {
		c := rl.NewVector3(float32(p.Aim.X)+0.5, float32(p.Aim.Y)+0.5, float32(p.Aim.Z)+0.5)
		rl.DrawCubeWires(c, 1.01, 1.01, 1.01, rl.NewColor(0, 0, 0, 220))
		if p.Mining {
			info := &blocks[g.World.Get(p.Aim.X, p.Aim.Y, p.Aim.Z)]
			if info.MineTime > 0 {
				frac := clamp(p.MineT/info.MineTime, 0, 1)
				rl.DrawCube(c, 1.02, 1.02, 1.02, rl.Fade(rl.Black, frac*0.6))
			}
		}
	}
	for _, tr := range g.Tracers {
		rl.DrawLine3D(tr.Start, tr.End, rl.NewColor(255, 240, 160, 255))
	}
	for _, s := range g.Sparks {
		rl.DrawCube(s.Pos, 0.1, 0.1, 0.1, rl.Fade(s.Col, s.Life*3))
	}
	rl.EndMode3D()
}

func (g *Game) drawWeapon() {
	sw, sh := float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())
	p := g.Player
	bobX := float32(math.Sin(float64(p.BobPhase))) * 6 * p.BobAmount
	bobY := float32(math.Abs(math.Sin(float64(p.BobPhase)))) * 5 * p.BobAmount
	dark := rl.NewColor(40, 42, 48, 255)
	mid := rl.NewColor(70, 74, 82, 255)
	if p.Tool == ToolBuilder {
		// A blocky arm holding the selected block.
		swing := float32(math.Sin(float64(p.Swing*math.Pi))) * 40
		rx := sw*0.68 + bobX - swing*0.3
		ry := sh - 170 + bobY + swing
		skin := rl.NewColor(200, 160, 120, 255)
		rl.DrawRectangle(int32(rx+40), int32(ry+40), 90, 220, skin)
		rl.DrawRectangle(int32(rx+40), int32(ry+40), 90, 220, rl.Fade(rl.Black, 0.15))
		if p.Place != Air {
			info := &blocks[p.Place]
			rl.DrawRectangle(int32(rx), int32(ry), 120, 120, info.Side)
			rl.DrawRectangle(int32(rx), int32(ry-30), 120, 30, info.Top)
			rl.DrawRectangleLines(int32(rx), int32(ry-30), 120, 150, rl.NewColor(0, 0, 0, 160))
		} else {
			rl.DrawRectangle(int32(rx+20), int32(ry-10), 50, 60, skin)
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
	panel := rl.NewColor(0, 0, 0, 140)
	const slot = 54
	x0 := sw/2 - 150
	y0 := sh - 76
	rl.DrawRectangle(x0-6, y0-6, 312, slot+12, panel)
	drawSlot := func(i int32, label string, sel bool) {
		x := x0 + i*(slot+6)
		bg := rl.NewColor(60, 60, 70, 220)
		if sel {
			bg = rl.NewColor(230, 200, 80, 255)
		}
		rl.DrawRectangle(x, y0, slot, slot, bg)
		rl.DrawRectangle(x+3, y0+3, slot-6, slot-6, rl.NewColor(30, 30, 36, 255))
		rl.DrawText(label, x+6, y0+18, 16, rl.White)
		rl.DrawText(fmt.Sprintf("%d", i+1), x+4, y0+2, 12, rl.LightGray)
	}
	drawSlot(0, "RIFLE", p.Tool == ToolRifle)
	drawSlot(1, "BUILD", p.Tool == ToolBuilder)
	// Selected block preview and inventory.
	x := x0 + 2*(slot+6) + 10
	if p.Place != Air {
		info := &blocks[p.Place]
		rl.DrawRectangle(x, y0+8, 38, 38, info.Side)
		rl.DrawRectangle(x, y0+8, 38, 10, info.Top)
		rl.DrawRectangleLines(x, y0+8, 38, 38, rl.Black)
		rl.DrawText(fmt.Sprintf("%s x%d", info.Name, p.Inv[p.Place]), x+46, y0+10, 18, rl.White)
		rl.DrawText("wheel/TAB: next block", x+46, y0+32, 12, rl.LightGray)
	} else {
		rl.DrawText("no blocks - mine some", x, y0+18, 16, rl.LightGray)
	}
}

func (g *Game) drawHUD() {
	sw, sh := int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight())
	p := g.Player
	cx, cy := sw/2, sh/2

	// Crosshair.
	gap := int32(6 + p.BobAmount*6 + p.Recoil*10)
	l := int32(10)
	col := rl.White
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

	g.drawWeapon()

	// Health bar.
	panel := rl.NewColor(0, 0, 0, 140)
	rl.DrawRectangle(20, sh-70, 260, 50, panel)
	rl.DrawText("HP", 30, sh-62, 20, rl.LightGray)
	w := int32(float32(200) * float32(p.HP) / maxHealth)
	hc := rl.Lime
	if p.HP < 35 {
		hc = rl.Red
	} else if p.HP < 65 {
		hc = rl.Orange
	}
	rl.DrawRectangle(70, sh-58, 200, 14, rl.DarkGray)
	rl.DrawRectangle(70, sh-58, w, 14, hc)
	rl.DrawText(fmt.Sprintf("%d", p.HP), 70, sh-42, 18, rl.White)

	// Ammo.
	rl.DrawRectangle(sw-220, sh-70, 200, 50, panel)
	ammoTxt := fmt.Sprintf("%d / %d", p.Ammo, p.Reserve)
	if p.Reloading > 0 {
		ammoTxt = "RELOADING"
	}
	rl.DrawText(ammoTxt, sw-205, sh-62, 32, rl.White)
	if p.Ammo == 0 && p.Reloading == 0 && p.Reserve == 0 {
		rl.DrawText("OUT OF AMMO - find yellow crates", cx-170, cy+60, 20, rl.Orange)
	}

	g.drawHotbar(sw, sh)

	// Score / wave.
	rl.DrawRectangle(20, 20, 340, 96, panel)
	rl.DrawText(fmt.Sprintf("SCORE  %d", g.Score), 30, 28, 26, rl.White)
	rl.DrawText(fmt.Sprintf("WAVE %d    HOSTILES %d    KILLS %d", g.Wave, g.aliveEnemies(), g.Kills), 30, 62, 18, rl.LightGray)
	rl.DrawText(fmt.Sprintf("BEST  %d", g.HighScore), 30, 86, 18, rl.Gold)

	g.drawMinimap(sw)

	// Message.
	if g.MsgT > 0 {
		fs := int32(36)
		tw := rl.MeasureText(g.Msg, fs)
		a := min(g.MsgT*2, 1)
		rl.DrawText(g.Msg, cx-tw/2+2, 122, fs, rl.Fade(rl.Black, a))
		rl.DrawText(g.Msg, cx-tw/2, 120, fs, rl.Fade(rl.Gold, a))
	}

	rl.DrawFPS(sw-90, 20)
}

// drawMinimap draws a top-down height-shaded map of the terrain in the top-right corner.
func (g *Game) drawMinimap(sw int32) {
	const size = 192
	mx, my := sw-20-size, int32(50)
	w := g.World
	cell := float32(size) / worldW
	toMap := func(x, z float32) (float32, float32) {
		return float32(mx) + (x-originX)*cell, float32(my) + (z-originZ)*cell
	}
	for lz := 0; lz < worldD; lz++ {
		for lx := 0; lx < worldW; lx++ {
			h := w.Height[lz*worldW+lx]
			b := w.getLocal(lx, h-1, lz)
			c := blocks[b].Top
			s := 0.55 + 0.45*float32(h)/22
			c = rl.NewColor(uint8(min(float32(c.R)*s, 255)), uint8(min(float32(c.G)*s, 255)), uint8(min(float32(c.B)*s, 255)), 230)
			rl.DrawRectangle(mx+int32(float32(lx)*cell), my+int32(float32(lz)*cell), int32(cell)+1, int32(cell)+1, c)
		}
	}
	for _, pk := range g.Pickups {
		x, y := toMap(pk.Pos.X, pk.Pos.Z)
		c := rl.NewColor(60, 200, 90, 255)
		if pk.Kind == PickAmmo {
			c = rl.NewColor(240, 180, 40, 255)
		}
		rl.DrawRectangle(int32(x)-2, int32(y)-2, 5, 5, c)
	}
	for _, e := range g.Enemies {
		if !e.Alive {
			continue
		}
		x, y := toMap(e.Pos.X, e.Pos.Z)
		rl.DrawCircle(int32(x), int32(y), 2+e.Spec.Radius*3, e.Spec.Head)
	}
	p := g.Player
	x, y := toMap(p.Pos.X, p.Pos.Z)
	f := p.FlatForward()
	rl.DrawLineEx(rl.NewVector2(x, y), rl.NewVector2(x+f.X*10, y+f.Z*10), 2, rl.White)
	rl.DrawCircle(int32(x), int32(y), 3.5, rl.White)
	rl.DrawRectangleLines(mx, my, size, size, rl.NewColor(120, 120, 130, 255))
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
		centered("ARENA STRIKE", sh/2-180, 64, rl.Gold)
		centered("A wave-survival shooter in a world of blocks", sh/2-110, 22, rl.LightGray)
		centered("WASD move   SHIFT sprint   SPACE jump   MOUSE look", sh/2-50, 20, rl.White)
		centered("1 rifle: LEFT CLICK fire, R reload", sh/2-20, 20, rl.White)
		centered("2 build: hold LEFT CLICK to mine, RIGHT CLICK to place, WHEEL picks block", sh/2+10, 20, rl.White)
		centered("Dig in, wall off, build high. Enemies climb one block at a time.", sh/2+46, 18, rl.LightGray)
		centered("Green cubes heal, yellow cubes give ammo. Headshots do 3x damage.", sh/2+70, 18, rl.LightGray)
		centered("Press ENTER or CLICK to start", sh/2+120, 28, rl.Lime)
		if g.HighScore > 0 {
			centered(fmt.Sprintf("High score  %d", g.HighScore), sh/2+165, 22, rl.Gold)
		}
	case StatePaused:
		centered("PAUSED", sh/2-60, 56, rl.White)
		centered("ESC resume    Q quit to menu", sh/2+20, 22, rl.LightGray)
	case StateGameOver:
		centered("YOU DIED", sh/2-100, 64, rl.Red)
		centered(fmt.Sprintf("Score %d    Waves survived %d    Kills %d", g.Score, max(g.Wave-1, 0), g.Kills), sh/2-20, 26, rl.White)
		if g.NewHigh {
			centered("NEW HIGH SCORE!", sh/2+16, 28, rl.Gold)
		} else {
			centered(fmt.Sprintf("High score  %d", g.HighScore), sh/2+16, 22, rl.Gold)
		}
		centered("ENTER restart    Q menu", sh/2+60, 24, rl.Lime)
	}
}

func main() {
	rl.SetConfigFlags(rl.FlagMsaa4xHint | rl.FlagVsyncHint | rl.FlagWindowResizable)
	rl.InitWindow(1280, 720, "Arena Strike")
	defer rl.CloseWindow()
	rl.SetTargetFPS(144)
	rl.SetExitKey(rl.KeyNull)

	g := NewGame()
	defer g.Audio.Close()

	for !rl.WindowShouldClose() {
		dt := min(rl.GetFrameTime(), 0.05)

		switch g.State {
		case StateMenu:
			if rl.IsKeyPressed(rl.KeyEnter) || rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
				g.Reset()
				rl.DisableCursor()
			}
		case StatePlaying:
			if rl.IsKeyPressed(rl.KeyEscape) {
				g.State = StatePaused
				rl.EnableCursor()
			} else {
				g.update(dt)
			}
		case StatePaused:
			if rl.IsKeyPressed(rl.KeyEscape) || rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
				g.State = StatePlaying
				rl.DisableCursor()
				rl.GetMouseDelta() // discard the jump from re-capturing
			} else if rl.IsKeyPressed(rl.KeyQ) {
				g.State = StateMenu
			}
		case StateGameOver:
			if rl.IsKeyPressed(rl.KeyEnter) {
				g.Reset()
				rl.DisableCursor()
			} else if rl.IsKeyPressed(rl.KeyQ) {
				g.State = StateMenu
			}
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.NewColor(120, 180, 235, 255))
		g.draw3D()
		if g.State == StatePlaying {
			g.drawHUD()
		} else {
			if g.State != StateMenu {
				g.drawHUD()
			}
			g.drawOverlay()
		}
		rl.EndDrawing()
	}
}
