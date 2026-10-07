package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	playerHalfW  = 0.3
	playerHeight = 1.8
	eyeHeight    = 1.62
	walkSpeed    = 5.0
	sprintSpeed  = 7.8
	flySpeed     = 11.0
	sneakSpeed   = 2.0
	jumpSpeed    = 6.9 // clears just over one block
	gravity      = 19.0
	mouseSens    = 0.0022
	keyLookSpeed = 2.4 // radians per second of arrow-key turning at sensitivity 1
	magSize      = 12
	fireInterval = 0.16
	reloadTime   = 1.4
	maxHealth    = 100
	startReserve = 36
	reachDist    = 5.5 // how far blocks can be mined or placed
	swordCD      = 0.45
	swordReach   = 3.2
	fallSafeV    = 11.5 // landing faster than this hurts (about four blocks)
	hotbarSlots  = 9
)

type ItemKind int

const (
	ItemRifle ItemKind = iota
	ItemSword
	ItemPickaxe
	ItemBlock
	ItemFood
)

// Item is one hotbar entry: a tool, or a stack of blocks from the inventory.
type Item struct {
	Kind  ItemKind
	Block Block
}

var swordDamage = [...]int{1, 2, 3, 4, 6}           // by tier
var armorReduce = [...]float32{0, 0.25, 0.45, 0.65} // damage absorbed by armour tier
var armorNames = [...]string{"", "Leather", "Iron", "Diamond"}
var pickSpeed = [...]float32{1, 2, 3.5, 5, 8} // mining speed multiplier for hard blocks, by tier

type Player struct {
	Pos       rl.Vector3 // feet position
	VelY      float32
	Yaw       float32
	Pitch     float32
	OnGround  bool
	InWater   bool // feet in water or lava
	InLava    bool
	OnLadder  bool
	HeadWater bool // eyes in water
	LavaT     float32
	CactusT   float32
	StepDist  float32 // distance walked since the last footstep
	Stepped   bool    // a footstep landed this frame (the game plays the sound)
	Flying    bool    // creative flight
	JumpTapT  float32 // seconds since the last jump press (double tap toggles flight)
	Sneak     bool
	Sprinting bool
	EyeOff    float32 // eye height offset while sneaking (smoothed)

	HP        int
	Ammo      int
	Reserve   int
	Reloading float32 // seconds remaining, 0 if not reloading
	FireCD    float32
	AttackCD  float32
	Recoil    float32
	BobPhase  float32
	BobAmount float32
	DmgFlash  float32
	SinceHurt float32
	RegenT    float32
	FallDmg   int // fall damage taken this frame (consumed by the game)

	Held      Item
	HotScroll int // first hotbar entry shown
	SwordTier int
	PickTier  int
	ArmorTier int
	Cause     string // what last hurt the player, for the death screen
	Inv       [numBlocks]int
	Aim       RayHit  // block under the crosshair
	Mining    bool    // currently breaking Aim
	MineT     float32 // progress in seconds on the current block
	Swing     float32 // arm swing animation timer
}

func NewPlayer(pos rl.Vector3) *Player {
	p := &Player{
		Pos:       pos,
		Yaw:       float32(math.Pi),
		HP:        maxHealth,
		Ammo:      magSize,
		Reserve:   startReserve,
		SwordTier: TierWood,
		PickTier:  TierWood,
		Held:      Item{Kind: ItemPickaxe},
	}
	p.Inv[Planks] = 16
	p.Inv[Torch] = 4
	return p
}

// Forward returns the full look direction; FlatForward ignores pitch.
func (p *Player) Forward() rl.Vector3 {
	cp := float32(math.Cos(float64(p.Pitch)))
	return rl.NewVector3(
		cp*float32(math.Sin(float64(p.Yaw))),
		float32(math.Sin(float64(p.Pitch))),
		cp*float32(math.Cos(float64(p.Yaw))))
}

func (p *Player) FlatForward() rl.Vector3 {
	return rl.NewVector3(float32(math.Sin(float64(p.Yaw))), 0, float32(math.Cos(float64(p.Yaw))))
}

// Right is forward x up: the direction D strafes toward.
func (p *Player) Right() rl.Vector3 {
	return rl.NewVector3(-float32(math.Cos(float64(p.Yaw))), 0, float32(math.Sin(float64(p.Yaw))))
}

func (p *Player) Eye() rl.Vector3 {
	bob := float32(math.Sin(float64(p.BobPhase))) * 0.05 * p.BobAmount
	return rl.NewVector3(p.Pos.X, p.Pos.Y+eyeHeight-p.EyeOff+bob, p.Pos.Z)
}

func (p *Player) Camera() rl.Camera3D {
	eye := p.Eye()
	// Recoil kicks the view up briefly.
	pitch := p.Pitch + p.Recoil*0.06
	cp := float32(math.Cos(float64(pitch)))
	fwd := rl.NewVector3(
		cp*float32(math.Sin(float64(p.Yaw))),
		float32(math.Sin(float64(pitch))),
		cp*float32(math.Cos(float64(p.Yaw))))
	fov := float32(75)
	if p.Sprinting {
		fov = 80
	}
	return rl.Camera3D{
		Position:   eye,
		Target:     rl.Vector3Add(eye, fwd),
		Up:         rl.NewVector3(0, 1, 0),
		Fovy:       fov,
		Projection: rl.CameraPerspective,
	}
}

// Box returns the player's collision bounds.
func (p *Player) Box() rl.BoundingBox {
	return rl.NewBoundingBox(
		rl.NewVector3(p.Pos.X-playerHalfW, p.Pos.Y, p.Pos.Z-playerHalfW),
		rl.NewVector3(p.Pos.X+playerHalfW, p.Pos.Y+playerHeight, p.Pos.Z+playerHalfW))
}

// Hotbar lists the tools followed by every block stack the player owns.
func (p *Player) Hotbar() []Item {
	items := []Item{{Kind: ItemRifle}, {Kind: ItemSword}, {Kind: ItemPickaxe}}
	for b := Block(1); b < numBlocks; b++ {
		if (p.Inv[b] > 0 || settings.Creative) && b.Placeable() {
			items = append(items, Item{ItemBlock, b})
		} else if p.Inv[b] > 0 && blocks[b].Food > 0 {
			items = append(items, Item{ItemFood, b})
		}
	}
	return items
}

// SelIndex returns the index of the held item in the hotbar (-1 if it vanished).
func (p *Player) SelIndex() int {
	for i, it := range p.Hotbar() {
		if it == p.Held {
			return i
		}
	}
	return -1
}

// EnsureHeld falls back to the pickaxe when the held block stack ran out and
// keeps the visible hotbar window around the selection.
func (p *Player) EnsureHeld() {
	i := p.SelIndex()
	if i < 0 {
		p.Held = Item{Kind: ItemPickaxe}
		i = 2
	}
	if i < p.HotScroll {
		p.HotScroll = i
	}
	if i >= p.HotScroll+hotbarSlots {
		p.HotScroll = i - hotbarSlots + 1
	}
	p.HotScroll = max(0, min(p.HotScroll, len(p.Hotbar())-hotbarSlots))
}

func (p *Player) selectIndex(i int) {
	hb := p.Hotbar()
	if len(hb) == 0 {
		return
	}
	i = ((i % len(hb)) + len(hb)) % len(hb)
	p.Held = hb[i]
	p.EnsureHeld()
}

// CycleHotbar moves the selection by step (mouse wheel).
func (p *Player) CycleHotbar(step int) { p.selectIndex(p.SelIndex() + step) }

// HoldBlock selects a block stack if the player owns it.
func (p *Player) HoldBlock(b Block) {
	if (p.Inv[b] > 0 || settings.Creative) && b.Placeable() {
		p.Held = Item{ItemBlock, b}
		p.EnsureHeld()
	}
}

func (p *Player) SwordDamage() int { return swordDamage[p.SwordTier] }

// MineTime returns how long the held tool needs to break a block (<0: cannot).
func (p *Player) MineTime(b Block) float32 {
	info := &blocks[b]
	if info.MineTime < 0 {
		return -1
	}
	if settings.Creative {
		return 0.05
	}
	if !info.Hard {
		return info.MineTime
	}
	tier := TierHand
	if p.Held.Kind == ItemPickaxe {
		tier = p.PickTier
	}
	if tier < info.MinTier {
		return -1
	}
	return info.MineTime / pickSpeed[tier]
}

func (p *Player) Update(dt float32, w *World) {
	p.FallDmg = 0
	// Mouse look.
	md := rl.GetMouseDelta()
	p.Yaw -= md.X * mouseSens * settings.Sensitivity
	if settings.InvertY {
		md.Y = -md.Y
	}
	p.Pitch -= md.Y * mouseSens * settings.Sensitivity
	// Arrow keys turn and tilt the view like the mouse.
	turn := keyLookSpeed * settings.Sensitivity * dt
	if rl.IsKeyDown(rl.KeyLeft) {
		p.Yaw += turn
	}
	if rl.IsKeyDown(rl.KeyRight) {
		p.Yaw -= turn
	}
	tilt := turn * 0.7
	if settings.InvertY {
		tilt = -tilt
	}
	if rl.IsKeyDown(rl.KeyUp) {
		p.Pitch += tilt
	}
	if rl.IsKeyDown(rl.KeyDown) {
		p.Pitch -= tilt
	}
	p.Pitch = clamp(p.Pitch, -1.55, 1.55)

	// Hotbar selection.
	for i, k := range []int32{rl.KeyOne, rl.KeyTwo, rl.KeyThree, rl.KeyFour, rl.KeyFive, rl.KeySix, rl.KeySeven, rl.KeyEight, rl.KeyNine} {
		if rl.IsKeyPressed(k) {
			if idx := p.HotScroll + i; idx < len(p.Hotbar()) {
				p.selectIndex(idx)
			}
		}
	}
	if wheel := rl.GetMouseWheelMove(); wheel != 0 {
		if wheel < 0 {
			p.CycleHotbar(1)
		} else {
			p.CycleHotbar(-1)
		}
	}
	if rl.IsKeyPressed(rl.KeyTab) {
		p.CycleHotbar(1)
	}
	p.EnsureHeld()

	// Movement input. Shift sneaks, Ctrl sprints (Minecraft bindings).
	var move rl.Vector3
	if rl.IsKeyDown(rl.KeyW) {
		move = rl.Vector3Add(move, p.FlatForward())
	}
	if rl.IsKeyDown(rl.KeyS) {
		move = rl.Vector3Subtract(move, p.FlatForward())
	}
	if rl.IsKeyDown(rl.KeyD) {
		move = rl.Vector3Add(move, p.Right())
	}
	if rl.IsKeyDown(rl.KeyA) {
		move = rl.Vector3Subtract(move, p.Right())
	}
	moving := rl.Vector3Length(move) > 0
	p.Sneak = rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
	p.Sprinting = !p.Sneak && moving && rl.IsKeyDown(rl.KeyW) && (rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl) || rl.IsKeyDown(rl.KeyLeftSuper))
	speed := float32(walkSpeed)
	if p.Sneak {
		speed = sneakSpeed
	} else if p.Sprinting {
		speed = sprintSpeed
	}
	// Creative flight: double-tap jump to toggle.
	p.JumpTapT += dt
	if settings.Creative && rl.IsKeyPressed(rl.KeySpace) {
		if p.JumpTapT < 0.3 {
			p.Flying = !p.Flying
			p.VelY = 0
			p.JumpTapT = 1
		} else {
			p.JumpTapT = 0
		}
	}
	if !settings.Creative {
		p.Flying = false
	}
	if p.Flying {
		speed = flySpeed
		if p.Sprinting || (rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)) {
			speed = flySpeed * 2
		}
	}

	feet := rl.NewVector3(p.Pos.X, p.Pos.Y+0.4, p.Pos.Z)
	p.InWater = w.LiquidAt(feet)
	p.InLava = w.LavaAt(feet) || w.LavaAt(p.Pos)
	p.OnLadder = w.BlockAt(feet) == Ladder || w.BlockAt(rl.NewVector3(p.Pos.X, p.Pos.Y+1.2, p.Pos.Z)) == Ladder
	atSurface := !p.InWater && w.LiquidAt(p.Pos) // bobbing at the water line
	if p.InWater {
		speed *= 0.55
		p.Sprinting = false
	}
	var delta rl.Vector3
	if moving {
		m := rl.Vector3Scale(rl.Vector3Normalize(move), speed*dt)
		delta.X, delta.Z = m.X, m.Z
	}

	// Vertical: fly, swim in water, otherwise jump and fall.
	if p.Flying {
		p.VelY = 0
		if rl.IsKeyDown(rl.KeySpace) {
			p.VelY = speed * 0.8
		}
		if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
			p.VelY = -speed * 0.8
		}
	} else if p.OnLadder && !p.InWater {
		// Climb with jump or forward, otherwise slide down slowly.
		p.VelY = -1.2
		if rl.IsKeyDown(rl.KeySpace) || rl.IsKeyDown(rl.KeyW) {
			p.VelY = 3.2
		}
		if p.Sneak {
			p.VelY = 0
		}
	} else if p.InWater {
		p.VelY = max(p.VelY-gravity*0.3*dt, -2.5)
		if rl.IsKeyDown(rl.KeySpace) {
			p.VelY = min(p.VelY+28*dt, 4)
		}
	} else {
		if (p.OnGround || atSurface) && rl.IsKeyPressed(rl.KeySpace) {
			p.VelY = jumpSpeed
		}
		p.VelY = max(p.VelY-gravity*dt, -30)
	}
	impact := p.VelY
	wasGround := p.OnGround
	var res MoveResult
	if p.Flying {
		p.Sneak = false // shift descends instead
	}
	if p.Sneak && p.OnGround && !p.InWater {
		// Sneaking never walks off an edge: apply each axis only if ground remains.
		for _, d := range [2]rl.Vector3{{X: delta.X}, {Z: delta.Z}} {
			try, _ := w.MoveBox(p.Pos, playerHalfW, playerHeight, d, false)
			if w.HasGround(try, playerHalfW) {
				p.Pos = try
			}
		}
		p.Pos, res = w.MoveBox(p.Pos, playerHalfW, playerHeight, rl.NewVector3(0, p.VelY*dt, 0), false)
	} else {
		delta.Y = p.VelY * dt
		p.Pos, res = w.MoveBox(p.Pos, playerHalfW, playerHeight, delta, false)
	}
	if res.Ground || res.Ceiling {
		p.VelY = 0
	}
	if p.Flying && res.Ground && rl.IsKeyDown(rl.KeyLeftShift) {
		p.Flying = false // landed
	}
	if (p.InWater || atSurface) && res.Wall && moving {
		p.VelY = max(p.VelY, 5.5) // swimming against a bank climbs out of the water
	}
	p.OnGround = res.Ground
	if p.OnGround && !wasGround && impact < -fallSafeV && !p.InWater && !p.OnLadder && !p.Flying {
		p.FallDmg = int((-impact - fallSafeV) * 3)
		p.Hurt(p.FallDmg, "fell from a high place", false)
	}
	p.HeadWater = w.WaterAt(p.Eye())
	// Lava burns.
	if p.InLava {
		p.LavaT += dt
		if p.LavaT >= 0.4 {
			p.LavaT = 0
			p.Hurt(6, "tried to swim in lava", false)
		}
	} else {
		p.LavaT = 0
	}

	// Cactus spines.
	if w.TouchesBlock(p.Pos, playerHalfW, playerHeight, Cactus) {
		p.CactusT += dt
		if p.CactusT >= 0.5 {
			p.CactusT = 0
			p.Hurt(2, "was pricked to death", false)
		}
	} else {
		p.CactusT = 0.4
	}

	// Head bob, footsteps and sneak camera.
	p.Stepped = false
	if moving && p.OnGround {
		p.BobPhase += dt * speed * 1.6
		p.BobAmount = lerp(p.BobAmount, 1, dt*8)
		p.StepDist += speed * dt
		if p.StepDist >= 1.9 {
			p.StepDist = 0
			p.Stepped = !p.Sneak
		}
	} else {
		p.BobAmount = lerp(p.BobAmount, 0, dt*8)
	}
	target := float32(0)
	if p.Sneak {
		target = 0.3
	}
	p.EyeOff = lerp(p.EyeOff, target, dt*12)

	// Timers.
	p.FireCD = max(0, p.FireCD-dt)
	p.AttackCD = max(0, p.AttackCD-dt)
	p.Recoil = lerp(p.Recoil, 0, dt*12)
	p.DmgFlash = max(0, p.DmgFlash-dt*2)
	p.Swing = max(0, p.Swing-dt*4)
	p.SinceHurt += dt
	// Slow natural regeneration once out of combat for a while.
	if p.SinceHurt > 6 && p.HP < maxHealth {
		p.RegenT += dt
		if p.RegenT >= 2.5 {
			p.RegenT = 0
			p.HP++
		}
	}
	if p.Reloading > 0 {
		p.Reloading -= dt
		if p.Reloading <= 0 {
			p.Reloading = 0
			need := magSize - p.Ammo
			take := min(need, p.Reserve)
			p.Ammo += take
			p.Reserve -= take
		}
	}
	if rl.IsKeyPressed(rl.KeyR) && p.Held.Kind == ItemRifle {
		p.StartReload()
	}
}

func (p *Player) StartReload() {
	if p.Reloading == 0 && p.Ammo < magSize && p.Reserve > 0 {
		p.Reloading = reloadTime
	}
}

// TryFire returns true if a rifle shot was fired this frame.
func (p *Player) TryFire() bool {
	if p.Held.Kind != ItemRifle || !attackDown() || p.FireCD > 0 || p.Reloading > 0 {
		return false
	}
	if p.Ammo == 0 {
		if attackPressed() {
			p.StartReload()
		}
		return false
	}
	p.Ammo--
	p.FireCD = fireInterval
	p.Recoil = 1
	if p.Ammo == 0 {
		p.StartReload()
	}
	return true
}

// TryAttack returns true if a sword swing starts this frame.
func (p *Player) TryAttack() bool {
	if p.Held.Kind != ItemSword || !attackDown() || p.AttackCD > 0 {
		return false
	}
	p.AttackCD = swordCD
	p.Swing = 1
	return true
}

// Hurt applies damage with a cause; armour absorbs part of it when armored is set.
func (p *Player) Hurt(n int, cause string, armored bool) {
	if n <= 0 {
		return
	}
	if armored {
		n = max(1, int(float32(n)*(1-armorReduce[p.ArmorTier])+0.5))
	}
	p.Cause = cause
	p.Damage(n)
}

func (p *Player) Damage(n int) {
	if n <= 0 || settings.Creative {
		return
	}
	p.HP -= n
	p.DmgFlash = 1
	p.SinceHurt = 0
	if p.HP < 0 {
		p.HP = 0
	}
}

func lerp(a, b, t float32) float32 {
	if t > 1 {
		t = 1
	}
	return a + (b-a)*t
}
