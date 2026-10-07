package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	enemyAttackCD = 1.0
	enemyReachY   = 2.2 // max vertical gap for a melee hit
	creeperFuse   = 1.5
)

type EnemyKind int

const (
	KindZombie EnemyKind = iota
	KindSpider
	KindCreeper
	KindBrute
	KindSkeleton
	KindGiant
)

type kindSpec struct {
	Name     string
	HP       int
	Speed    float32
	Radius   float32
	Height   float32
	HeadR    float32
	Damage   int
	Points   int
	Body     rl.Color
	Head     rl.Color
	Eyes     rl.Color
	Burns    bool // dies in sunlight
	Explodes bool
	Ranged   bool // keeps its distance and shoots arrows
	Smashes  bool // breaks through blocks in its way
}

var kinds = [...]kindSpec{
	KindZombie:   {"Zombie", 4, 2.4, 0.35, 1.9, 0.33, 12, 100, rl.NewColor(40, 120, 170, 255), rl.NewColor(70, 150, 70, 255), rl.NewColor(20, 20, 20, 255), true, false, false, false},
	KindSpider:   {"Spider", 3, 4.6, 0.6, 0.9, 0.3, 8, 125, rl.NewColor(45, 35, 40, 255), rl.NewColor(60, 45, 50, 255), rl.NewColor(230, 40, 40, 255), true, false, false, false},
	KindCreeper:  {"Creeper", 4, 2.9, 0.3, 1.7, 0.3, 0, 200, rl.NewColor(70, 160, 60, 255), rl.NewColor(80, 175, 70, 255), rl.NewColor(10, 10, 10, 255), false, true, false, false},
	KindBrute:    {"Zombie Brute", 14, 1.7, 0.45, 2.6, 0.45, 30, 300, rl.NewColor(90, 60, 130, 255), rl.NewColor(60, 130, 60, 255), rl.NewColor(230, 60, 60, 255), true, false, false, false},
	KindSkeleton: {"Skeleton", 4, 2.6, 0.3, 1.9, 0.3, 0, 150, rl.NewColor(205, 205, 200, 255), rl.NewColor(215, 215, 210, 255), rl.NewColor(40, 40, 40, 255), true, false, true, false},
	KindGiant:    {"Giant", 60, 1.3, 0.8, 4.2, 0.7, 40, 1500, rl.NewColor(60, 110, 60, 255), rl.NewColor(80, 150, 70, 255), rl.NewColor(255, 60, 60, 255), false, false, false, true},
}

// Target is something hostiles chase: the local player or a remote one.
type Target struct {
	ID  uint32 // 0 is the local player
	Pos rl.Vector3
	Eye rl.Vector3
	Box rl.BoundingBox
}

type Enemy struct {
	ID       uint32
	TargetID uint32 // who the last Update chased
	Kind     EnemyKind
	Spec     *kindSpec
	Pos      rl.Vector3
	Heading  rl.Vector3
	VelY     float32
	HP       int
	MaxHP    int
	Speed    float32
	AttackCD float32
	HitFlash float32
	Phase    float32
	Fuse     float32 // creeper: seconds spent hissing
	Exploded bool    // creeper: went off this frame (the game handles the blast)
	Burning  bool    // standing in sunlight
	BurnT    float32
	Lum      float32 // brightness of the cell the enemy stands in (set by the game before drawing)
	ShootCD  float32
	Shoot    bool    // wants to fire an arrow this frame (the game spawns it)
	VoiceCD  float32 // time until the next idle sound
	Smash    bool    // blocked by terrain this frame (giants break through)
	Alive    bool
	DeathT   float32 // death animation timer
}

func NewEnemy(pos rl.Vector3, kind EnemyKind, night int) *Enemy {
	s := &kinds[kind]
	hp := s.HP + night/2
	if kind == KindBrute {
		hp = s.HP + night*2
	}
	jitter := float32(math.Mod(float64(pos.X*7.3+pos.Z*3.1), 1.0)) * 0.5
	speed := min(s.Speed+float32(night)*0.12, s.Speed*1.6) + jitter
	return &Enemy{
		Kind:    kind,
		Spec:    s,
		Pos:     pos,
		HP:      hp,
		MaxHP:   hp,
		Speed:   speed,
		Alive:   true,
		Lum:     1,
		VoiceCD: 2 + float32(math.Mod(float64(pos.X+pos.Z), 5)),
	}
}

func (e *Enemy) BB() rl.BoundingBox {
	r := e.Spec.Radius
	return rl.NewBoundingBox(
		rl.NewVector3(e.Pos.X-r, e.Pos.Y, e.Pos.Z-r),
		rl.NewVector3(e.Pos.X+r, e.Pos.Y+e.Spec.Height, e.Pos.Z+r))
}

func (e *Enemy) HeadY() float32 { return e.Spec.Height - e.Spec.HeadR }

func (e *Enemy) HeadCenter() rl.Vector3 {
	return rl.NewVector3(e.Pos.X, e.Pos.Y+e.HeadY(), e.Pos.Z)
}

func (e *Enemy) AttackRange() float32 { return e.Spec.Radius + playerHalfW + 0.9 }

// Update steers toward the player (directly when adjacent, otherwise along the
// nav field), walks with gravity and one-block step-ups, and attacks when close.
// Returns melee damage dealt this frame.
func (e *Enemy) Update(dt float32, w *World, nav *NavGrid, p Target, others []*Enemy) int {
	e.TargetID = p.ID
	if !e.Alive {
		e.DeathT += dt
		return 0
	}
	e.HitFlash = max(0, e.HitFlash-dt*6)
	e.AttackCD = max(0, e.AttackCD-dt)
	e.Phase += dt * e.Speed * 2

	to := rl.Vector3Subtract(p.Pos, e.Pos)
	to.Y = 0
	dist := rl.Vector3Length(to)
	dy := math.Abs(float64(p.Pos.Y - e.Pos.Y))

	// Creepers stop and hiss when close, and calm down again if the player escapes.
	hissing := false
	if e.Spec.Explodes {
		if dist < 2.6 && dy < 2.5 {
			hissing = true
			e.Fuse += dt
			if e.Fuse >= creeperFuse {
				e.Exploded = true
				e.Alive = false
				e.DeathT = 10
				return 0
			}
		} else if dist > 4.5 {
			e.Fuse = max(0, e.Fuse-dt*2)
		}
	}

	// Archers hang back at range and shoot when they can see the player.
	e.Shoot = false
	e.ShootCD = max(0, e.ShootCD-dt)
	if e.Spec.Ranged {
		eye := rl.NewVector3(e.Pos.X, e.Pos.Y+e.HeadY(), e.Pos.Z)
		target := p.Eye
		clear := !w.RayCast(eye, rl.Vector3Subtract(target, eye), rl.Vector3Distance(eye, target)).Hit
		if clear && dist < 16 && e.ShootCD == 0 {
			e.ShootCD = 2.2
			e.Shoot = true
		}
	}

	inWater := w.WaterAt(rl.NewVector3(e.Pos.X, e.Pos.Y+0.3, e.Pos.Z))
	var delta rl.Vector3
	if !hissing && dist > e.AttackRange()*0.8 {
		want := rl.Vector3Scale(to, 1/dist)
		if dist > 2.5 {
			if d, ok := nav.Dir(e.Pos); ok {
				want = d
			}
		}
		if e.Spec.Ranged {
			switch {
			case dist < 6:
				want = rl.Vector3Scale(to, -1/dist) // back off
			case dist < 11:
				want = rl.NewVector3(-to.Z/dist, 0, to.X/dist) // strafe
			}
		}
		// Smooth the heading so enemies flow around corners instead of jittering.
		e.Heading = rl.Vector3Add(e.Heading, rl.Vector3Scale(rl.Vector3Subtract(want, e.Heading), min(dt*7, 1)))
		if l := rl.Vector3Length(e.Heading); l > 1e-3 {
			e.Heading = rl.Vector3Scale(e.Heading, 1/l)
		} else {
			e.Heading = want
		}
		speed := e.Speed
		if inWater {
			speed *= 0.5
		}
		delta.X = e.Heading.X * speed * dt
		delta.Z = e.Heading.Z * speed * dt
	}
	if inWater {
		e.VelY = max(e.VelY-gravity*0.3*dt, -1.5)
		if w.WaterAt(rl.NewVector3(e.Pos.X, e.Pos.Y+e.Spec.Height*0.8, e.Pos.Z)) {
			e.VelY = min(e.VelY+14*dt, 2.5) // float back up to breathe
		}
	} else {
		e.VelY = max(e.VelY-gravity*dt, -30)
	}
	delta.Y = e.VelY * dt
	var res MoveResult
	e.Pos, res = w.MoveBox(e.Pos, e.Spec.Radius, e.Spec.Height, delta, true)
	if res.Ground || res.Ceiling {
		e.VelY = 0
	}
	e.Smash = e.Spec.Smashes && res.Wall

	// Separate from other enemies so they don't stack.
	var push rl.Vector3
	for _, o := range others {
		if o == e || !o.Alive {
			continue
		}
		d := rl.Vector3Subtract(e.Pos, o.Pos)
		d.Y = 0
		l := rl.Vector3Length(d)
		minD := (e.Spec.Radius + o.Spec.Radius) * 1.1
		if l > 1e-4 && l < minD {
			push = rl.Vector3Add(push, rl.Vector3Scale(d, (minD-l)/l*0.5))
		}
	}
	if push.X != 0 || push.Z != 0 {
		e.Pos, _ = w.MoveBox(e.Pos, e.Spec.Radius, e.Spec.Height, push, false)
	}

	if e.Spec.Damage > 0 && dist <= e.AttackRange() && e.AttackCD == 0 && dy < enemyReachY {
		e.AttackCD = enemyAttackCD
		return e.Spec.Damage
	}
	return 0
}

// Hit applies damage; returns true if the enemy died from it.
func (e *Enemy) Hit(dmg int) bool {
	if !e.Alive {
		return false
	}
	e.HP -= dmg
	e.HitFlash = 1
	if e.HP <= 0 {
		e.Alive = false
		return true
	}
	return false
}

func (e *Enemy) tint(c rl.Color) rl.Color {
	c = mul(c, e.Lum)
	if e.HitFlash > 0 {
		f := uint8(e.HitFlash * 255)
		return rl.NewColor(max(c.R, f), max(c.G, f), max(c.B, f), 255)
	}
	return c
}

// modelScale maps the hostile's collision height onto its skinned model.
func (e *Enemy) modelScale() float32 {
	switch e.Kind {
	case KindCreeper:
		return e.Spec.Height / 1.625
	case KindSpider:
		return e.Spec.Height / 0.95
	}
	return e.Spec.Height / 2.0
}

// Draw renders the skinned, animated model.
func (e *Enemy) Draw(w *World) {
	s := e.Spec
	pose := Pose{Pos: e.Pos, Yaw: yawOf(e.Heading), Phase: e.Phase, Amp: 1, Scale: e.modelScale(), Lum: e.Lum, Alpha: 1}
	if !e.Alive {
		pose.Death = clamp(e.DeathT*2.5, 0, 1)
		pose.Alpha = 1 - pose.Death
		pose.Amp = 0
		if pose.Death >= 1 {
			return
		}
	}
	pose.Flash = e.HitFlash
	if e.Fuse > 0 {
		pose.Flash = max(pose.Flash, (float32(math.Sin(float64(e.Fuse*22)))*0.5+0.5)*0.9)
		pose.Amp = 0
	}
	switch e.Kind {
	case KindSpider:
		skins.DrawSpider(w, &pose)
	case KindCreeper:
		skins.DrawCreeper(w, &pose)
	case KindSkeleton:
		skins.DrawHumanoid(w, SkinSkeleton, &pose)
	case KindBrute, KindGiant:
		pose.ArmsOut = true
		skins.DrawHumanoid(w, SkinBrute, &pose)
	default:
		pose.ArmsOut = true
		skins.DrawHumanoid(w, SkinZombie, &pose)
	}
	if !e.Alive {
		return
	}
	x, z := e.Pos.X, e.Pos.Z
	y := e.Pos.Y
	if e.Burning {
		// Flames licking up the body.
		for i := 0; i < 4; i++ {
			fl := float32(math.Sin(float64(e.Phase*3+float32(i)*2.1)))*0.15 + 0.2
			fp := rl.NewVector3(x+float32(math.Sin(float64(e.Phase*2+float32(i))))*0.25, y+s.Height*0.4+fl+float32(i)*0.25, z+float32(math.Cos(float64(e.Phase*2+float32(i)*1.3)))*0.25)
			rl.DrawCubeV(fp, rl.NewVector3(0.25, 0.35, 0.25), rl.NewColor(255, 140+uint8(i*20), 20, 200))
		}
	}
	// Health bar once damaged.
	if e.HP < e.MaxHP {
		top := y + s.Height + 0.2
		frac := float32(e.HP) / float32(e.MaxHP)
		rl.DrawCubeV(rl.NewVector3(x, top, z), rl.NewVector3(1.0, 0.08, 0.08), rl.NewColor(0, 0, 0, 180))
		rl.DrawCubeV(rl.NewVector3(x-(1-frac)*0.5, top, z), rl.NewVector3(frac, 0.1, 0.1), rl.Lime)
	}
}

// DrawGlow draws the eyes at full brightness so hostiles can be spotted at night.
func (e *Enemy) DrawGlow() {
	if !e.Alive {
		return
	}
	s := e.Spec
	fwd := e.Heading
	if rl.Vector3Length(fwd) < 0.01 {
		fwd = rl.NewVector3(0, 0, 1)
	}
	side := rl.NewVector3(-fwd.Z, 0, fwd.X)
	sc := e.modelScale()
	if e.Kind == KindSpider {
		hp := rl.Vector3Add(rl.NewVector3(e.Pos.X, e.Pos.Y+0.6*sc, e.Pos.Z), rl.Vector3Scale(fwd, 0.88*sc))
		for _, sg := range []float32{-1.5, -0.5, 0.5, 1.5} {
			rl.DrawCubeV(rl.Vector3Add(hp, rl.Vector3Scale(side, sg*0.1*sc)), rl.NewVector3(0.06, 0.06, 0.03), s.Eyes)
		}
		return
	}
	if e.Kind == KindCreeper {
		return
	}
	headY := e.Pos.Y + 1.72*sc
	if e.Kind == KindCreeper {
		headY = e.Pos.Y + 1.375*sc
	}
	f := rl.Vector3Scale(fwd, 0.26*sc)
	for _, sgn := range []float32{-1, 1} {
		ep := rl.Vector3Add(rl.NewVector3(e.Pos.X, headY+0.03*sc, e.Pos.Z), rl.Vector3Add(f, rl.Vector3Scale(side, sgn*0.11*sc)))
		rl.DrawCubeV(ep, rl.NewVector3(0.07*sc, 0.07*sc, 0.03), s.Eyes)
	}
}
