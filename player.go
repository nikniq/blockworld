package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	playerHalfW  = 0.3
	playerHeight = 1.8
	eyeHeight    = 1.62
	walkSpeed    = 5.5
	sprintSpeed  = 8.5
	jumpSpeed    = 6.9 // clears just over one block
	gravity      = 19.0
	mouseSens    = 0.0022
	magSize      = 12
	fireInterval = 0.16
	reloadTime   = 1.4
	maxHealth    = 100
	startReserve = 48
	reachDist    = 6.0 // how far blocks can be mined or placed
)

type Tool int

const (
	ToolRifle Tool = iota
	ToolBuilder
)

type Player struct {
	Pos      rl.Vector3 // feet position
	VelY     float32
	Yaw      float32
	Pitch    float32
	OnGround bool

	HP        int
	Ammo      int
	Reserve   int
	Reloading float32 // seconds remaining, 0 if not reloading
	FireCD    float32
	Recoil    float32
	BobPhase  float32
	BobAmount float32
	DmgFlash  float32

	Tool   Tool
	Inv    [numBlocks]int
	Place  Block   // block type that right click places
	Aim    RayHit  // block under the crosshair (builder tool)
	Mining bool    // currently breaking Aim
	MineT  float32 // progress in seconds on the current block
	Swing  float32 // arm swing animation timer
}

func NewPlayer(pos rl.Vector3) *Player {
	p := &Player{
		Pos:     pos,
		Yaw:     float32(math.Pi),
		HP:      maxHealth,
		Ammo:    magSize,
		Reserve: startReserve,
		Place:   Planks,
	}
	p.Inv[Planks] = 24
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

func (p *Player) Right() rl.Vector3 {
	return rl.NewVector3(float32(math.Cos(float64(p.Yaw))), 0, -float32(math.Sin(float64(p.Yaw))))
}

func (p *Player) Eye() rl.Vector3 {
	bob := float32(math.Sin(float64(p.BobPhase))) * 0.05 * p.BobAmount
	return rl.NewVector3(p.Pos.X, p.Pos.Y+eyeHeight+bob, p.Pos.Z)
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
	return rl.Camera3D{
		Position:   eye,
		Target:     rl.Vector3Add(eye, fwd),
		Up:         rl.NewVector3(0, 1, 0),
		Fovy:       75,
		Projection: rl.CameraPerspective,
	}
}

// Box returns the player's collision bounds.
func (p *Player) Box() rl.BoundingBox {
	return rl.NewBoundingBox(
		rl.NewVector3(p.Pos.X-playerHalfW, p.Pos.Y, p.Pos.Z-playerHalfW),
		rl.NewVector3(p.Pos.X+playerHalfW, p.Pos.Y+playerHeight, p.Pos.Z+playerHalfW))
}

func (p *Player) Update(dt float32, w *World) {
	// Mouse look.
	md := rl.GetMouseDelta()
	p.Yaw -= md.X * mouseSens
	p.Pitch -= md.Y * mouseSens
	p.Pitch = clamp(p.Pitch, -1.5, 1.5)

	// Tool selection.
	if rl.IsKeyPressed(rl.KeyOne) {
		p.Tool = ToolRifle
	}
	if rl.IsKeyPressed(rl.KeyTwo) {
		p.Tool = ToolBuilder
	}
	if wheel := rl.GetMouseWheelMove(); wheel != 0 || rl.IsKeyPressed(rl.KeyTab) {
		if p.Tool == ToolBuilder {
			step := 1
			if wheel < 0 {
				step = -1
			}
			p.CyclePlace(step)
		} else if wheel != 0 {
			p.Tool = ToolBuilder
		}
	}

	// Movement input.
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
	speed := float32(walkSpeed)
	if rl.IsKeyDown(rl.KeyLeftShift) {
		speed = sprintSpeed
	}
	moving := rl.Vector3Length(move) > 0
	var delta rl.Vector3
	if moving {
		m := rl.Vector3Scale(rl.Vector3Normalize(move), speed*dt)
		delta.X, delta.Z = m.X, m.Z
	}
	if p.OnGround && rl.IsKeyPressed(rl.KeySpace) {
		p.VelY = jumpSpeed
	}
	p.VelY = max(p.VelY-gravity*dt, -30)
	delta.Y = p.VelY * dt
	var res MoveResult
	p.Pos, res = w.MoveBox(p.Pos, playerHalfW, playerHeight, delta, false)
	if res.Ground || res.Ceiling {
		p.VelY = 0
	}
	p.OnGround = res.Ground

	// Head bob.
	if moving && p.OnGround {
		p.BobPhase += dt * speed * 1.6
		p.BobAmount = lerp(p.BobAmount, 1, dt*8)
	} else {
		p.BobAmount = lerp(p.BobAmount, 0, dt*8)
	}

	// Timers.
	p.FireCD = max(0, p.FireCD-dt)
	p.Recoil = lerp(p.Recoil, 0, dt*12)
	p.DmgFlash = max(0, p.DmgFlash-dt*2)
	p.Swing = max(0, p.Swing-dt*4)
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
	if rl.IsKeyPressed(rl.KeyR) && p.Tool == ToolRifle {
		p.StartReload()
	}
}

// CyclePlace moves the placement selection to the next owned block type.
func (p *Player) CyclePlace(step int) {
	for i := 1; i < int(numBlocks); i++ {
		b := Block((int(p.Place) + step*i + int(numBlocks)*i) % int(numBlocks))
		if b != Air && p.Inv[b] > 0 {
			p.Place = b
			return
		}
	}
}

// EnsurePlace picks any owned block if the current selection ran out.
func (p *Player) EnsurePlace() {
	if p.Place != Air && p.Inv[p.Place] > 0 {
		return
	}
	for b := Block(1); b < numBlocks; b++ {
		if p.Inv[b] > 0 {
			p.Place = b
			return
		}
	}
	p.Place = Air
}

func (p *Player) StartReload() {
	if p.Reloading == 0 && p.Ammo < magSize && p.Reserve > 0 {
		p.Reloading = reloadTime
	}
}

// TryFire returns true if a shot was fired this frame.
func (p *Player) TryFire() bool {
	if p.Tool != ToolRifle || !rl.IsMouseButtonDown(rl.MouseButtonLeft) || p.FireCD > 0 || p.Reloading > 0 {
		return false
	}
	if p.Ammo == 0 {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
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

func (p *Player) Damage(n int) {
	p.HP -= n
	p.DmgFlash = 1
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
