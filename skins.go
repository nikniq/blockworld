package main

import (
	"image"
	"image/color"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Character models: every creature is a set of textured boxes (head, body,
// limbs) hung on pivots, animated by rotating them, and drawn with the
// terrain shader so they are lit like the world. Skins are painted at
// startup into their own atlas, one 16x16 tile per box face.

type SkinKind int

const (
	SkinPlayer SkinKind = iota
	SkinZombie
	SkinBrute
	SkinSkeleton
	SkinCreeper
	SkinSpider
	SkinPig
	SkinCow
	SkinSheep
	numSkins
)

// Parts; not every kind uses every part.
const (
	PartHead = iota
	PartBody
	PartArmL
	PartArmR
	PartLegL
	PartLegR
	PartLegFL // quadrupeds and spiders
	PartLegFR
	PartLegBL
	PartLegBR
	PartExtra // snout, bow, wool
	numParts
)

// Face order for painting: top, bottom, left (+X), right (-X), front (+Z), back (-Z).
const skinTiles = 32 // tiles per atlas row

func skinTile(kind SkinKind, part, face int) int { return (int(kind)*numParts+part)*6 + face }

// The slot after the last skin holds the block-breaking crack stages (one per part index).
const skinCracks = numSkins
const crackStages = 10

var skinRows = ((int(numSkins)+1)*numParts*6 + skinTiles - 1) / skinTiles

func skinUV(kind SkinKind, part, face int) (u0, v0, u1, v1 float32) {
	i := skinTile(kind, part, face)
	tx, ty := float32(i%skinTiles), float32(i/skinTiles)
	aw, ah := float32(skinTiles*tileSize), float32(skinRows*tileSize)
	const inset = 0.05
	return (tx*tileSize + inset) / aw, (ty*tileSize + inset) / ah, ((tx+1)*tileSize - inset) / aw, ((ty+1)*tileSize - inset) / ah
}

// faceIndexOf maps a faceDef to the painting face order.
func faceIndexOf(f *faceDef) int {
	switch {
	case f.n[1] > 0:
		return 0
	case f.n[1] < 0:
		return 1
	case f.n[0] > 0:
		return 2
	case f.n[0] < 0:
		return 3
	case f.n[2] > 0:
		return 4
	}
	return 5
}

// ---------- painting ----------

var (
	skinTone    = col(205, 160, 120)
	zombieSkin  = col(80, 140, 70)
	boneTone    = col(220, 220, 212)
	creeperTone = col(80, 165, 65)
)

// paintSkin returns the pixel of a face tile for a character part.
func paintSkin(kind SkinKind, part, face, x, y int) rl.Color {
	n := hash2(x, y, 500+int(kind)*97+part*7+face)
	shade := func(c rl.Color) rl.Color { return mul(c, 0.9+0.2*n) }
	eyes := func(base, iris rl.Color, white bool) rl.Color {
		// Two eyes on a 16px face, mouth below.
		if y >= 8 && y <= 9 {
			if x == 4 || x == 11 {
				return iris
			}
			if white && (x == 5 || x == 10) {
				return rl.White
			}
		}
		return shade(base)
	}
	if kind == skinCracks {
		// Cracks radiating from the centre; more and longer with each stage.
		stage := part
		for k := 0; k <= stage; k++ {
			ang := float64(hash2(k, 1, 900)) * 2 * math.Pi
			length := 3 + float32(stage)*1.1 + hash2(k, 2, 900)*2
			dx, dy := float32(math.Cos(ang)), float32(math.Sin(ang))
			px, py := float32(x)-7.5, float32(y)-7.5
			along := px*dx + py*dy
			across := px*dy - py*dx
			wobble := float32(math.Sin(float64(along)*1.7+float64(k))) * 0.5
			if along > 0 && along < length && math.Abs(float64(across-wobble)) < 0.8 {
				return rl.NewColor(20, 20, 20, 230)
			}
		}
		return rl.NewColor(0, 0, 0, 0)
	}
	switch kind {
	case SkinPlayer, SkinZombie, SkinBrute:
		skin, shirt, pants := skinTone, col(60, 170, 170), col(50, 60, 150)
		hair := col(70, 45, 30)
		if kind == SkinZombie {
			skin, shirt, pants = zombieSkin, col(40, 120, 170), col(45, 50, 120)
			hair = col(50, 95, 50)
		}
		if kind == SkinBrute {
			skin, shirt, pants = zombieSkin, col(95, 60, 135), col(45, 40, 70)
			hair = col(50, 95, 50)
		}
		switch part {
		case PartHead:
			if face == 0 || (face != 1 && y < 4) {
				return shade(hair)
			}
			if face == 4 {
				if y == 12 && x >= 6 && x <= 9 {
					return col(120, 70, 60)
				}
				if kind == SkinPlayer {
					return eyes(skin, col(40, 60, 150), true)
				}
				return eyes(skin, col(15, 15, 15), false)
			}
			return shade(skin)
		case PartBody:
			if kind != SkinPlayer && hash2(x/2, y/2, 77+int(kind)) < 0.12 {
				return shade(mul(shirt, 0.55)) // torn patches
			}
			if face == 4 && y > 11 && x >= 6 && x <= 9 {
				return shade(mul(shirt, 0.8))
			}
			return shade(shirt)
		case PartArmL, PartArmR:
			if y < 4 && face != 1 {
				return shade(shirt) // sleeve
			}
			return shade(skin)
		case PartLegL, PartLegR:
			if y >= 13 && face != 0 {
				return shade(col(40, 35, 35)) // shoes
			}
			return shade(pants)
		}
	case SkinSkeleton:
		if part == PartHead {
			if face == 4 {
				if y >= 7 && y <= 9 && (x >= 3 && x <= 5 || x >= 10 && x <= 12) {
					return col(20, 20, 20)
				}
				if y >= 11 && y <= 12 && x >= 7 && x <= 8 {
					return col(60, 60, 60)
				}
				if y == 14 && x%2 == 0 {
					return col(90, 90, 90) // teeth line
				}
			}
			return shade(boneTone)
		}
		if part == PartExtra {
			return shade(col(120, 85, 45)) // bow
		}
		if part == PartBody {
			if y%4 == 2 && face != 0 && face != 1 {
				return shade(mul(boneTone, 0.6)) // ribs
			}
			return shade(mul(boneTone, 0.8))
		}
		if y == 7 || y == 8 {
			return shade(mul(boneTone, 0.65)) // joint
		}
		return shade(boneTone)
	case SkinCreeper:
		base := mix(creeperTone, col(40, 100, 40), hash2(x/2, y/2, 91))
		if part == PartHead && face == 4 {
			if (y >= 5 && y <= 7 && (x >= 3 && x <= 6 || x >= 9 && x <= 12)) ||
				(y >= 8 && y <= 11 && x >= 6 && x <= 9) || (y >= 10 && y <= 13 && (x >= 4 && x <= 5 || x >= 10 && x <= 11)) {
				return col(15, 15, 15)
			}
		}
		return shade(base)
	case SkinSpider:
		base := col(45, 35, 40)
		if hash2(x, y, 55) < 0.2 {
			base = col(70, 55, 60)
		}
		if part == PartHead && face == 4 {
			if y >= 5 && y <= 6 && (x == 3 || x == 6 || x == 9 || x == 12) {
				return col(230, 40, 40)
			}
			if y >= 8 && y <= 9 && (x == 5 || x == 10) {
				return col(200, 30, 30)
			}
		}
		return shade(base)
	case SkinPig:
		pink := col(235, 160, 170)
		if part == PartExtra {
			if face == 4 && y >= 5 && y <= 9 && (x == 5 || x == 10) {
				return col(90, 40, 50) // nostrils
			}
			return shade(col(215, 125, 140))
		}
		if part == PartHead && face == 4 {
			return eyes(pink, col(20, 20, 20), true)
		}
		return shade(pink)
	case SkinCow:
		brown := col(85, 55, 40)
		if hash2(x/3, y/3, 61+part) < 0.22 {
			brown = col(225, 220, 210) // white patches
		}
		if part == PartHead {
			if face == 4 && y >= 11 {
				return shade(col(200, 180, 170)) // muzzle
			}
			if face == 4 {
				return eyes(col(85, 55, 40), col(20, 20, 20), true)
			}
			if face == 0 && (x <= 1 || x >= 14) {
				return shade(col(200, 200, 190)) // horns
			}
		}
		return shade(brown)
	case SkinSheep:
		wool := col(232, 232, 226)
		if part == PartHead {
			if face == 4 {
				return eyes(col(75, 65, 60), col(20, 20, 20), true)
			}
			return shade(col(75, 65, 60))
		}
		if part == PartLegFL || part == PartLegFR || part == PartLegBL || part == PartLegBR {
			if y >= 10 {
				return shade(col(75, 65, 60))
			}
		}
		return mul(wool, 0.85+0.3*hash2(x/2, y/2, 71))
	}
	return shade(col(200, 200, 200))
}

func buildSkinAtlas() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, skinTiles*tileSize, skinRows*tileSize))
	for kind := SkinKind(0); kind <= numSkins; kind++ {
		for part := 0; part < numParts; part++ {
			for face := 0; face < 6; face++ {
				i := skinTile(kind, part, face)
				ox, oy := (i%skinTiles)*tileSize, (i/skinTiles)*tileSize
				for y := 0; y < tileSize; y++ {
					for x := 0; x < tileSize; x++ {
						c := paintSkin(kind, part, face, x, y)
						img.SetRGBA(ox+x, oy+y, color.RGBA{c.R, c.G, c.B, c.A})
					}
				}
			}
		}
	}
	return img
}

// ---------- meshes and drawing ----------

type partMesh struct {
	mesh *meshBuf
}

// Skins owns the atlas, material and cached part meshes.
type Skins struct {
	tex   rl.Texture2D
	mat   rl.Material
	ready bool
	parts map[[2]int]*meshBuf // (kind, part) -> unit box mesh
}

var skins = &Skins{parts: map[[2]int]*meshBuf{}}

func (s *Skins) init(w *World) {
	if s.ready {
		return
	}
	img := rl.NewImageFromImage(buildSkinAtlas())
	s.tex = rl.LoadTextureFromImage(img)
	rl.UnloadImage(img)
	rl.SetTextureFilter(s.tex, rl.FilterPoint)
	s.mat = rl.LoadMaterialDefault()
	s.mat.GetMap(rl.MapDiffuse).Texture = s.tex
	if w.shaderOK {
		s.mat.Shader = w.shader
	}
	s.ready = true
}

// box returns the unit-cube mesh of a part, textured per face.
func (s *Skins) box(kind SkinKind, part int) *meshBuf {
	key := [2]int{int(kind), part}
	if m, ok := s.parts[key]; ok {
		return m
	}
	m := &meshBuf{}
	for fi := range faces {
		f := &faces[fi]
		u0, v0, u1, v1 := skinUV(kind, part, faceIndexOf(f))
		m.emitFaceUV(f, u0, v0, u1, v1, -0.5, -0.5, -0.5, unitBox, 1, [4]int{3, 3, 3, 3}, fullLight)
	}
	m.upload()
	s.parts[key] = m
	return m
}

// Pose is everything needed to draw one creature.
type Pose struct {
	Pos       rl.Vector3
	Yaw       float32 // facing, radians (0 faces +Z)
	Pitch     float32 // head tilt, radians
	Phase     float32 // walk cycle
	Amp       float32 // walk swing amplitude 0..1
	Swing     float32 // attack / mining swing 0..1 (right arm)
	Scale     float32
	Lum       float32 // light at the creature
	Alpha     float32
	Death     float32 // 0 alive .. 1 fallen over
	Flash     float32 // hit flash 0..1 (white overlay)
	ArmsOut   bool    // zombie arms held forward
	Sneak     bool
	Held      Item
	SwordTier int
	PickTier  int
}

// drawPart draws one box: size, origin offset (where the pivot sits on the box,
// as a fraction of the size: y=0.5 means pivot at the top), pivot position in
// model space, and rotations about X (swing) and Z (sideways) at the pivot.
func (s *Skins) drawPart(kind SkinKind, part int, size, pivot rl.Vector3, originY float32, rotX, rotY, rotZ float32, model rl.Matrix, color rl.Color) {
	m := rl.MatrixScale(size.X, size.Y, size.Z)
	m = rl.MatrixMultiply(m, rl.MatrixTranslate(0, -originY*size.Y, 0))
	m = rl.MatrixMultiply(m, rl.MatrixRotateZ(rotZ))
	m = rl.MatrixMultiply(m, rl.MatrixRotateX(rotX))
	m = rl.MatrixMultiply(m, rl.MatrixRotateY(rotY))
	m = rl.MatrixMultiply(m, rl.MatrixTranslate(pivot.X, pivot.Y, pivot.Z))
	m = rl.MatrixMultiply(m, model)
	mm := s.mat.GetMap(rl.MapDiffuse)
	mm.Color = color
	rl.DrawMesh(s.box(kind, part).mesh, s.mat, m)
}

// modelMatrix builds the creature transform: scale, death tilt, facing, position.
func modelMatrix(p *Pose) rl.Matrix {
	m := rl.MatrixScale(p.Scale, p.Scale, p.Scale)
	if p.Death > 0 {
		// Topple backwards and sink a little.
		m = rl.MatrixMultiply(m, rl.MatrixRotateX(p.Death*math.Pi/2))
		m = rl.MatrixMultiply(m, rl.MatrixTranslate(0, -p.Death*0.3, 0))
	}
	m = rl.MatrixMultiply(m, rl.MatrixRotateY(p.Yaw))
	return rl.MatrixMultiply(m, rl.MatrixTranslate(p.Pos.X, p.Pos.Y, p.Pos.Z))
}

// lightColor encodes local brightness and alpha for the terrain shader's colDiffuse.
func lightColor(w *World, pos rl.Vector3, alpha float32) rl.Color {
	lx, y, lz := floorI(pos.X)-originX, floorI(pos.Y), floorI(pos.Z)-originZ
	return rl.NewColor(uint8(w.sunLocal(lx, y, lz)*17), uint8(w.blockLocal(lx, y, lz)*17), 255, uint8(255*clamp(alpha, 0, 1)))
}

func swingAt(phase, amp float32) float32 { return float32(math.Sin(float64(phase))) * 0.7 * amp }

// DrawHumanoid draws a player, zombie, brute, giant or skeleton.
func (s *Skins) DrawHumanoid(w *World, kind SkinKind, p *Pose) {
	s.init(w)
	model := modelMatrix(p)
	c := lightColor(w, rl.Vector3Add(p.Pos, rl.NewVector3(0, 1, 0)), p.Alpha)
	sw := swingAt(p.Phase, p.Amp)
	const legH, bodyH, headS float32 = 0.75, 0.75, 0.5
	bodyTilt := float32(0)
	if p.Sneak {
		bodyTilt = 0.35
	}
	// Legs pivot at the hips.
	s.drawPart(kind, PartLegL, rl.NewVector3(0.25, legH, 0.25), rl.NewVector3(0.125, legH, 0), 0.5, sw, 0, 0, model, c)
	s.drawPart(kind, PartLegR, rl.NewVector3(0.25, legH, 0.25), rl.NewVector3(-0.125, legH, 0), 0.5, -sw, 0, 0, model, c)
	// Body pivots at the hips too, so sneaking leans it forward.
	s.drawPart(kind, PartBody, rl.NewVector3(0.5, bodyH, 0.25), rl.NewVector3(0, legH, 0), -0.5, -bodyTilt, 0, 0, model, c)
	shoulderY := legH + bodyH - 0.05
	armL, armR := -sw, sw
	if p.ArmsOut {
		armL, armR = -math.Pi/2+sw*0.15, -math.Pi/2-sw*0.15
	}
	if p.Swing > 0 {
		armR = -float32(math.Sin(float64(p.Swing*math.Pi))) * 1.6
	}
	s.drawPart(kind, PartArmL, rl.NewVector3(0.25, 0.75, 0.25), rl.NewVector3(0.375, shoulderY, 0), 0.5, armL, 0, 0.08, model, c)
	s.drawPart(kind, PartArmR, rl.NewVector3(0.25, 0.75, 0.25), rl.NewVector3(-0.375, shoulderY, 0), 0.5, armR, 0, -0.08, model, c)
	headY := legH + bodyH - bodyTilt*0.3
	s.drawPart(kind, PartHead, rl.NewVector3(headS, headS, headS), rl.NewVector3(0, headY, bodyTilt*0.25), -0.5, -p.Pitch, 0, 0, model, c)
	if kind == SkinSkeleton {
		// Bow in the left hand, held across the body.
		s.drawPart(kind, PartExtra, rl.NewVector3(0.06, 0.9, 0.06), rl.NewVector3(0.42, shoulderY-0.5, 0.25), 0, 0.3, 0, 0, model, c)
	}
	// Held item in the right hand.
	hand := rl.Vector3Transform(rl.NewVector3(-0.42, shoulderY-0.65, 0.15), model)
	switch p.Held.Kind {
	case ItemBlock, ItemFood:
		m := rl.MatrixMultiply(rl.MatrixScale(0.28, 0.28, 0.28), rl.MatrixMultiply(rl.MatrixRotateY(p.Yaw), rl.MatrixTranslate(hand.X, hand.Y, hand.Z)))
		w.DrawBlockAt(p.Held.Block, m)
	case ItemSword, ItemPickaxe, ItemRifle:
		fwd := rl.NewVector3(float32(math.Sin(float64(p.Yaw))), 0, float32(math.Cos(float64(p.Yaw))))
		tip := rl.Vector3Add(hand, rl.Vector3Scale(fwd, 0.35))
		colr := mul(tierColors[p.SwordTier], p.Lum)
		size := rl.NewVector3(0.08, 0.1, 0.8)
		if p.Held.Kind == ItemPickaxe {
			colr = mul(rl.NewColor(120, 85, 45, 255), p.Lum)
			size = rl.NewVector3(0.08, 0.08, 0.7)
		} else if p.Held.Kind == ItemRifle {
			colr = mul(rl.NewColor(50, 52, 58, 255), p.Lum)
			size = rl.NewVector3(0.1, 0.14, 0.9)
		}
		rl.DrawCubeV(tip, size, colr)
		if p.Held.Kind == ItemPickaxe {
			rl.DrawCubeV(rl.Vector3Add(hand, rl.Vector3Scale(fwd, 0.7)), rl.NewVector3(0.3, 0.12, 0.12), mul(tierColors[p.PickTier], p.Lum))
		}
	}
	if p.Flash > 0 {
		rl.DrawCubeV(rl.Vector3Add(p.Pos, rl.NewVector3(0, p.Scale*1.0, 0)), rl.NewVector3(0.7*p.Scale, 2.0*p.Scale, 0.5*p.Scale), rl.Fade(rl.White, p.Flash*0.6))
	}
}

// DrawCreeper draws the four-legged, armless creeper.
func (s *Skins) DrawCreeper(w *World, p *Pose) {
	s.init(w)
	model := modelMatrix(p)
	c := lightColor(w, rl.Vector3Add(p.Pos, rl.NewVector3(0, 1, 0)), p.Alpha)
	sw := swingAt(p.Phase, p.Amp)
	legs := [][3]float32{{0.125, 0.2, 1}, {-0.125, 0.2, -1}, {0.125, -0.2, -1}, {-0.125, -0.2, 1}}
	for _, l := range legs {
		s.drawPart(SkinCreeper, PartLegFL, rl.NewVector3(0.25, 0.375, 0.25), rl.NewVector3(l[0], 0.375, l[1]), 0.5, sw*l[2], 0, 0, model, c)
	}
	s.drawPart(SkinCreeper, PartBody, rl.NewVector3(0.5, 0.75, 0.25), rl.NewVector3(0, 0.375, 0), -0.5, 0, 0, 0, model, c)
	s.drawPart(SkinCreeper, PartHead, rl.NewVector3(0.5, 0.5, 0.5), rl.NewVector3(0, 1.125, 0), -0.5, -p.Pitch, 0, 0, model, c)
	if p.Flash > 0 {
		rl.DrawCubeV(rl.Vector3Add(p.Pos, rl.NewVector3(0, 0.85*p.Scale, 0)), rl.NewVector3(0.6*p.Scale, 1.7*p.Scale, 0.6*p.Scale), rl.Fade(rl.White, p.Flash*0.7))
	}
}

// DrawSpider draws a low body with a head in front and eight legs.
func (s *Skins) DrawSpider(w *World, p *Pose) {
	s.init(w)
	model := modelMatrix(p)
	c := lightColor(w, rl.Vector3Add(p.Pos, rl.NewVector3(0, 0.4, 0)), p.Alpha)
	s.drawPart(SkinSpider, PartBody, rl.NewVector3(0.9, 0.45, 1.1), rl.NewVector3(0, 0.5, -0.1), 0, 0, 0, 0, model, c)
	s.drawPart(SkinSpider, PartHead, rl.NewVector3(0.5, 0.45, 0.45), rl.NewVector3(0, 0.5, 0.65), 0, -p.Pitch, 0, 0, model, c)
	for i := 0; i < 4; i++ {
		z := 0.35 - float32(i)*0.25
		lift := float32(math.Sin(float64(p.Phase+float32(i)*1.6))) * 0.25 * p.Amp
		for _, sg := range []float32{1, -1} {
			// Legs stick out sideways and angle down; pivot at the body side.
			s.drawPart(SkinSpider, PartLegFL, rl.NewVector3(0.8, 0.1, 0.1), rl.NewVector3(sg*0.4, 0.55, z), 0, lift*sg, sg*0.15*(float32(i)-1.5), sg*0.45, model, c)
		}
	}
	if p.Flash > 0 {
		rl.DrawCubeV(rl.Vector3Add(p.Pos, rl.NewVector3(0, 0.5, 0)), rl.NewVector3(1.0, 0.6, 1.2), rl.Fade(rl.White, p.Flash*0.6))
	}
}

// DrawQuadruped draws pigs, cows and sheep.
func (s *Skins) DrawQuadruped(w *World, kind SkinKind, p *Pose) {
	s.init(w)
	model := modelMatrix(p)
	c := lightColor(w, rl.Vector3Add(p.Pos, rl.NewVector3(0, 0.5, 0)), p.Alpha)
	sw := swingAt(p.Phase, p.Amp)
	var body, head rl.Vector3
	var legH, legW float32
	switch kind {
	case SkinPig:
		body, head, legH, legW = rl.NewVector3(0.6, 0.5, 1.0), rl.NewVector3(0.5, 0.5, 0.5), 0.375, 0.25
	case SkinCow:
		body, head, legH, legW = rl.NewVector3(0.75, 0.65, 1.15), rl.NewVector3(0.5, 0.5, 0.4), 0.75, 0.25
	default:
		body, head, legH, legW = rl.NewVector3(0.75, 0.6, 1.0), rl.NewVector3(0.4, 0.4, 0.45), 0.55, 0.22
	}
	legs := [][3]float32{{body.X/2 - legW/2, body.Z/2 - legW/2, 1}, {-(body.X/2 - legW/2), body.Z/2 - legW/2, -1},
		{body.X/2 - legW/2, -(body.Z/2 - legW/2), -1}, {-(body.X/2 - legW/2), -(body.Z/2 - legW/2), 1}}
	parts := []int{PartLegFL, PartLegFR, PartLegBL, PartLegBR}
	for i, l := range legs {
		s.drawPart(kind, parts[i], rl.NewVector3(legW, legH, legW), rl.NewVector3(l[0], legH, l[1]), 0.5, sw*l[2], 0, 0, model, c)
	}
	s.drawPart(kind, PartBody, body, rl.NewVector3(0, legH, 0), -0.5, 0, 0, 0, model, c)
	headPos := rl.NewVector3(0, legH+body.Y-head.Y*0.4, body.Z/2+head.Z*0.3)
	s.drawPart(kind, PartHead, head, headPos, 0, -p.Pitch, 0, 0, model, c)
	if kind == SkinPig {
		s.drawPart(kind, PartExtra, rl.NewVector3(0.25, 0.18, 0.1), rl.Vector3Add(headPos, rl.NewVector3(0, -0.08, head.Z/2+0.03)), 0, 0, 0, 0, model, c)
	}
	if p.Flash > 0 {
		rl.DrawCubeV(rl.Vector3Add(p.Pos, rl.NewVector3(0, 0.5, 0)), rl.NewVector3(body.X+0.1, legH+body.Y+0.1, body.Z+0.1), rl.Fade(rl.White, p.Flash*0.6))
	}
}

// DrawCrack overlays the block-breaking cracks on a block (stage 0..9).
func (s *Skins) DrawCrack(w *World, x, y, z int, stage int) {
	s.init(w)
	stage = max(0, min(crackStages-1, stage))
	m := rl.MatrixMultiply(rl.MatrixScale(1.02, 1.02, 1.02), rl.MatrixTranslate(float32(x)+0.5, float32(y)+0.5, float32(z)+0.5))
	mm := s.mat.GetMap(rl.MapDiffuse)
	mm.Color = lightColor(w, rl.NewVector3(float32(x)+0.5, float32(y)+1.5, float32(z)+0.5), 1)
	rl.DrawMesh(s.box(skinCracks, stage).mesh, s.mat, m)
}

func yawOf(heading rl.Vector3) float32 {
	if heading.X == 0 && heading.Z == 0 {
		return 0
	}
	return float32(math.Atan2(float64(heading.X), float64(heading.Z)))
}

func (s *Skins) Unload() {
	if !s.ready {
		return
	}
	for _, m := range s.parts {
		m.free()
	}
	s.parts = map[[2]int]*meshBuf{}
	rl.UnloadTexture(s.tex)
	s.ready = false
}
