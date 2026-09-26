package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	enemyAttackCD = 1.0
	enemyReachY   = 2.2 // max vertical gap for a melee hit
)

type EnemyKind int

const (
	KindGrunt EnemyKind = iota
	KindRunner
	KindBrute
)

type kindSpec struct {
	Name   string
	HP     int
	Speed  float32
	Radius float32
	Height float32
	HeadR  float32
	Damage int
	Points int
	Body   rl.Color
	Head   rl.Color
}

var kinds = [...]kindSpec{
	KindGrunt:  {"Grunt", 3, 2.6, 0.35, 1.9, 0.33, 12, 100, rl.NewColor(170, 40, 40, 255), rl.NewColor(210, 60, 60, 255)},
	KindRunner: {"Runner", 2, 4.8, 0.3, 1.5, 0.26, 8, 125, rl.NewColor(210, 120, 30, 255), rl.NewColor(245, 165, 60, 255)},
	KindBrute:  {"Brute", 12, 1.7, 0.45, 2.6, 0.45, 30, 300, rl.NewColor(90, 40, 130, 255), rl.NewColor(150, 80, 190, 255)},
}

type Enemy struct {
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
	Alive    bool
	DeathT   float32 // death animation timer
}

func NewEnemy(pos rl.Vector3, kind EnemyKind, wave int) *Enemy {
	s := &kinds[kind]
	hp := s.HP + wave/3
	if kind == KindBrute {
		hp = s.HP + wave
	}
	jitter := float32(math.Mod(float64(pos.X*7.3+pos.Z*3.1), 1.0)) * 0.6
	speed := min(s.Speed+float32(wave)*0.18, s.Speed*1.7) + jitter
	return &Enemy{
		Kind:  kind,
		Spec:  s,
		Pos:   pos,
		HP:    hp,
		MaxHP: hp,
		Speed: speed,
		Alive: true,
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
// Returns damage dealt this frame.
func (e *Enemy) Update(dt float32, w *World, nav *NavGrid, p *Player, others []*Enemy) int {
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
	var delta rl.Vector3
	if dist > e.AttackRange()*0.8 {
		want := rl.Vector3Scale(to, 1/dist)
		if dist > 2.5 {
			if d, ok := nav.Dir(e.Pos); ok {
				want = d
			}
		}
		// Smooth the heading so enemies flow around corners instead of jittering.
		e.Heading = rl.Vector3Add(e.Heading, rl.Vector3Scale(rl.Vector3Subtract(want, e.Heading), min(dt*7, 1)))
		if l := rl.Vector3Length(e.Heading); l > 1e-3 {
			e.Heading = rl.Vector3Scale(e.Heading, 1/l)
		} else {
			e.Heading = want
		}
		delta.X = e.Heading.X * e.Speed * dt
		delta.Z = e.Heading.Z * e.Speed * dt
	}
	e.VelY = max(e.VelY-gravity*dt, -30)
	delta.Y = e.VelY * dt
	var res MoveResult
	e.Pos, res = w.MoveBox(e.Pos, e.Spec.Radius, e.Spec.Height, delta, true)
	if res.Ground || res.Ceiling {
		e.VelY = 0
	}

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

	if dist <= e.AttackRange() && e.AttackCD == 0 && math.Abs(float64(p.Pos.Y-e.Pos.Y)) < enemyReachY {
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

func (e *Enemy) Draw() {
	s := e.Spec
	if !e.Alive {
		// Collapse animation.
		t := clamp(1-e.DeathT*2.5, 0, 1)
		if t <= 0 {
			return
		}
		c := rl.NewColor(s.Body.R/2, s.Body.G/2, s.Body.B/2, uint8(200*t))
		w := s.Radius * 2.6
		rl.DrawCubeV(rl.NewVector3(e.Pos.X, e.Pos.Y+0.5*t, e.Pos.Z), rl.NewVector3(w, 1.0*t+0.05, w), c)
		return
	}
	body, head := s.Body, s.Head
	if e.HitFlash > 0 {
		f := uint8(e.HitFlash * 255)
		body = rl.NewColor(max(body.R, f), max(body.G, f), max(body.B, f), 255)
		head = body
	}
	kw := s.Radius / 0.35 // width scale relative to a grunt
	kh := s.Height / 1.9  // height scale relative to a grunt
	bob := float32(math.Abs(math.Sin(float64(e.Phase)))) * 0.08
	y := e.Pos.Y + bob
	dark := rl.NewColor(s.Body.R/3, s.Body.G/3, s.Body.B/3, 255)
	// Blocky legs, torso and head in the spirit of the world.
	rl.DrawCubeV(rl.NewVector3(e.Pos.X-0.17*kw, y+0.35*kh, e.Pos.Z), rl.NewVector3(0.26*kw, 0.7*kh, 0.26*kw), dark)
	rl.DrawCubeV(rl.NewVector3(e.Pos.X+0.17*kw, y+0.35*kh, e.Pos.Z), rl.NewVector3(0.26*kw, 0.7*kh, 0.26*kw), dark)
	torso := rl.NewVector3(e.Pos.X, y+1.0*kh, e.Pos.Z)
	tsz := rl.NewVector3(0.7*kw, 0.7*kh, 0.4*kw)
	rl.DrawCubeV(torso, tsz, body)
	rl.DrawCubeWiresV(torso, tsz, rl.NewColor(30, 0, 0, 255))
	hs := s.HeadR * 1.8
	rl.DrawCubeV(rl.NewVector3(e.Pos.X, y+e.HeadY(), e.Pos.Z), rl.NewVector3(hs, hs, hs), head)
	rl.DrawCubeWiresV(rl.NewVector3(e.Pos.X, y+e.HeadY(), e.Pos.Z), rl.NewVector3(hs, hs, hs), rl.NewColor(30, 0, 0, 255))
	// Eyes.
	f := rl.Vector3Scale(e.Heading, hs/2+0.01)
	side := rl.NewVector3(-e.Heading.Z, 0, e.Heading.X)
	for _, sgn := range []float32{-1, 1} {
		ep := rl.Vector3Add(rl.NewVector3(e.Pos.X, y+e.HeadY()+hs*0.1, e.Pos.Z), rl.Vector3Add(f, rl.Vector3Scale(side, sgn*hs*0.22)))
		rl.DrawCubeV(ep, rl.NewVector3(hs*0.18, hs*0.18, hs*0.18), rl.NewColor(255, 230, 90, 255))
	}
	// Health bar once damaged.
	if e.HP < e.MaxHP {
		top := y + s.Height + 0.2
		frac := float32(e.HP) / float32(e.MaxHP)
		rl.DrawCubeV(rl.NewVector3(e.Pos.X, top, e.Pos.Z), rl.NewVector3(1.0, 0.08, 0.08), rl.NewColor(0, 0, 0, 180))
		rl.DrawCubeV(rl.NewVector3(e.Pos.X-(1-frac)*0.5, top, e.Pos.Z), rl.NewVector3(frac, 0.1, 0.1), rl.Lime)
	}
}
