package main

import (
	"image"
	"image/color"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Block textures are generated at startup into one atlas, three tiles per
// block (top, side, bottom), so the game ships without asset files. The style
// is painterly rather than pixel art: 64x64 tiles filled with soft tonal
// washes, warm/cool hue drift and brush strokes, with seams and shapes drawn
// as smooth gradients instead of hard pixel lines. Shapes (items, doors,
// ladders) are laid out on a coarse 16x16 design grid so they stay readable.

type texPattern int

const (
	PatNoise texPattern = iota
	PatSand
	PatGrassTop
	PatGrassSide
	PatSnowSide
	PatGravel
	PatPlanks
	PatLogSide
	PatLogTop
	PatLeaves
	PatBricks
	PatCobble
	PatOre
	PatGlass
	PatWater
	PatGold
	PatBedrock
	PatTorch
	PatTNT
	PatTNTTop
	PatMeat
	PatLava
	PatWool
	PatBedTop
	PatSapling
	PatLadder
	PatBirchSide
	PatCactus
	PatCactusTop
	PatTuft
	PatFlower
	PatSpawner
	PatCrate
	PatApple
	PatCrop0
	PatCrop1
	PatCrop2
	PatBread
	PatBow
	PatArrow
	PatDoor
	PatBeacon
	PatFish
	PatRod
	PatShroom
	PatAmethyst
	PatMeteor
	PatFruit
	PatBoat
	PatFlag
)

const (
	atlasPad   = 8 // border of repeated edge pixels around each tile, so mipmaps do not bleed
	atlasTiles = 8 // tiles per row
)

// The tile size follows the art style: 16 for the classic pixel look, 64 for
// the painted looks. applyStyleAtlas sets these before an atlas is built.
var (
	tileSize  = 64
	subT      = tileSize / 16         // design-grid cell: shapes are laid out on 16x16
	atlasCell = tileSize + 2*atlasPad // pitch of one tile in the atlas
	atlasRows = (int(numBlocks)*3 + atlasTiles - 1) / atlasTiles
)

func applyStyleAtlas() {
	tileSize = 64
	if settings.Style == StylePixel {
		tileSize = 16
	}
	subT = tileSize / 16
	atlasCell = tileSize + 2*atlasPad
	atlasRows = (int(numBlocks)*3 + atlasTiles - 1) / atlasTiles
}

func tileIndex(b Block, face int) int { return int(b)*3 + face }

// tileUV returns the atlas rectangle of a block face, inset slightly so
// filtering never bleeds into the neighbouring tile.
func tileUV(b Block, face int) (u0, v0, u1, v1 float32) {
	i := tileIndex(b, face)
	tx, ty := float32(i%atlasTiles*atlasCell+atlasPad), float32(i/atlasTiles*atlasCell+atlasPad)
	aw, ah := float32(atlasTiles*atlasCell), float32(atlasRows*atlasCell)
	const inset = 0.5
	return (tx + inset) / aw, (ty + inset) / ah, (tx + float32(tileSize) - inset) / aw, (ty + float32(tileSize) - inset) / ah
}

// TileRect returns the atlas rectangle of a block face in pixels (for the HUD).
func TileRect(b Block, face int) rl.Rectangle {
	i := tileIndex(b, face)
	return rl.NewRectangle(float32(i%atlasTiles*atlasCell+atlasPad), float32(i/atlasTiles*atlasCell+atlasPad), float32(tileSize), float32(tileSize))
}

func mul(c rl.Color, v float32) rl.Color {
	return rl.NewColor(uint8(clamp(float32(c.R)*v, 0, 255)), uint8(clamp(float32(c.G)*v, 0, 255)), uint8(clamp(float32(c.B)*v, 0, 255)), c.A)
}

func mix(a, b rl.Color, t float32) rl.Color {
	t = clamp(t, 0, 1)
	return rl.NewColor(uint8(lerp(float32(a.R), float32(b.R), t)), uint8(lerp(float32(a.G), float32(b.G), t)),
		uint8(lerp(float32(a.B), float32(b.B), t)), uint8(lerp(float32(a.A), float32(b.A), t)))
}

func buildAtlas() *image.RGBA {
	applyStyleAtlas()
	img := image.NewRGBA(image.Rect(0, 0, atlasTiles*atlasCell, atlasRows*atlasCell))
	for b := Block(1); b < numBlocks; b++ {
		info := &blocks[b]
		bases := [3]rl.Color{info.Top, info.Side, info.Bottom}
		for f := 0; f < 3; f++ {
			i := tileIndex(b, f)
			ox, oy := (i%atlasTiles)*atlasCell, (i/atlasTiles)*atlasCell
			seed := 1000 + int(b)*3 + f
			// Paint the whole cell; coordinates outside the tile clamp to its edge.
			for y := 0; y < atlasCell; y++ {
				for x := 0; x < atlasCell; x++ {
					tx := min(max(x-atlasPad, 0), tileSize-1)
					ty := min(max(y-atlasPad, 0), tileSize-1)
					var c rl.Color
					if settings.Style == StylePixel {
						c = texelPixel(info.Pat[f], bases[f], info, tx, ty, seed)
					} else {
						c = texel(info.Pat[f], bases[f], info, tx, ty, seed)
					}
					if c.A == 0 {
						// Transparent texels carry the base tint so linear filtering and
						// mipmaps blend cutout edges towards colour, not black.
						c = rl.NewColor(bases[f].R, bases[f].G, bases[f].B, 0)
					}
					img.SetRGBA(ox+x, oy+y, color.RGBA{c.R, c.G, c.B, c.A})
				}
			}
		}
	}
	return img
}

// ---------- painterly helpers ----------

// paintT is the painted tile size; the painterly patterns are written in its units.
const paintT float32 = 64

// sstep is a smooth 0..1 ramp between two edges.
func sstep(e0, e1, v float32) float32 {
	t := clamp((v-e0)/(e1-e0), 0, 1)
	return t * t * (3 - 2*t)
}

// fmod is a positive modulo for floats.
func fmod(v, m float32) float32 { return v - m*float32(math.Floor(float64(v/m))) }

// tnoise is value noise with fx by fy cells across the tile, so it wraps seamlessly.
func tnoise(x, y float32, fx, fy, seed int) float32 {
	return pnoise(x*float32(fx)/paintT, y*float32(fy)/paintT, seed, fx, fy)
}

// tfbm layers three octaves of tnoise.
func tfbm(x, y float32, f, seed int) float32 {
	return tnoise(x, y, f, f, seed)*0.55 + tnoise(x, y, f*2, f*2, seed+11)*0.3 + tnoise(x, y, f*4, f*4, seed+23)*0.15
}

// brush is stretched noise that reads as strokes along one axis.
func brush(x, y float32, vertical bool, seed int) float32 {
	if vertical {
		return tnoise(x, y, 14, 3, seed)*0.7 + tnoise(x, y, 28, 6, seed+7)*0.3
	}
	return tnoise(x, y, 3, 14, seed)*0.7 + tnoise(x, y, 6, 28, seed+7)*0.3
}

func clamp8(v float32) uint8 { return uint8(clamp(v, 0, 255)) }

// paint fills with a base colour the way a brush would: broad light and dark
// washes, a drift between warmer and cooler versions of the colour, and
// strokes along one axis. strength scales the tonal range.
func paint(base rl.Color, x, y float32, seed int, strength float32, vertical bool) rl.Color {
	tone := tfbm(x, y, 3, seed)
	str := brush(x, y, vertical, seed+3)
	hue := tnoise(x, y, 2, 2, seed+5)
	warm := rl.NewColor(clamp8(float32(base.R)*1.1), base.G, clamp8(float32(base.B)*0.86), base.A)
	cool := rl.NewColor(clamp8(float32(base.R)*0.9), base.G, clamp8(float32(base.B)*1.12), base.A)
	c := mix(cool, warm, hue)
	return mul(c, 1+strength*((tone-0.5)*1.2+(str-0.5)*0.8))
}

// seam darkens smoothly towards a joint: 1 at distance w and beyond.
func seam(d, w float32) float32 { return 0.55 + 0.45*sstep(0, w, d) }

// wrapD is the distance between two coordinates on the wrapping tile.
func wrapD(a, b float32) float32 {
	d := float32(math.Abs(float64(a - b)))
	return min(d, paintT-d)
}

// voronoi finds the nearest and second-nearest of n scattered stones.
func voronoi(x, y float32, n, seed int) (d1, d2 float32, id int) {
	d1, d2 = 99, 99
	for i := 0; i < n; i++ {
		px, py := hash2(i, 1, seed)*paintT, hash2(i, 2, seed)*paintT
		dx, dy := wrapD(x, px), wrapD(y, py)
		d := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		if d < d1 {
			d2, d1, id = d1, d, i
		} else if d < d2 {
			d2 = d
		}
	}
	return
}

// ---------- materials ----------

func texel(p texPattern, base rl.Color, info *blockInfo, px, py, seed int) rl.Color {
	x, y := float32(px)+0.5, float32(py)+0.5
	switch p {
	case PatNoise:
		if info.Hard { // stone: mottled, with faint soft cracks
			c := paint(base, x, y, seed, 0.22, false)
			cr := float32(math.Abs(float64(tnoise(x, y, 5, 5, seed+31) - 0.5)))
			v := 1 - 0.3*(1-sstep(0, 0.03, cr))
			if tfbm(x, y, 6, seed+37) > 0.66 {
				v *= 0.9
			}
			return mul(c, v)
		}
		c := paint(base, x, y, seed, 0.3, false) // soil: clumps and the odd pebble
		v := 0.88 + 0.24*tnoise(x, y, 8, 8, seed+41)
		if tnoise(x, y, 12, 12, seed+43) > 0.84 {
			v += 0.15
		}
		return mul(c, v)
	case PatSand:
		c := paint(base, x, y, seed, 0.14, false)
		r := float32(math.Sin(float64((y + 6*tnoise(x, y, 2, 2, seed+9)) * 2 * math.Pi * 4 / paintT)))
		v := 1 + 0.06*r
		if tnoise(x, y, 16, 16, seed+13) > 0.86 {
			v -= 0.08
		}
		return mul(c, v)
	case PatGrassTop:
		c := paint(base, x, y, seed, 0.32, true)
		v := float32(1)
		if d := tfbm(x, y, 4, seed+17); d < 0.34 {
			v *= 0.8 + 0.2*sstep(0.26, 0.34, d)
		} else if d > 0.68 {
			v *= 1 + 0.12*sstep(0.68, 0.78, d)
		}
		return mul(c, v)
	case PatGrassSide, PatSnowSide:
		topCol, lo, hi := blocks[Grass].Top, float32(0.12), float32(0.16)
		top := PatGrassTop
		if p == PatSnowSide {
			topCol, lo, hi, top = blocks[Snow].Top, 0.24, 0.14, PatNoise
		}
		edge := paintT * (lo + hi*tnoise(x, 0, 6, 1, seed))
		t := sstep(edge-2.5, edge+2.5, y)
		over := texel(top, topCol, info, px, py, seed+2)
		under := paint(base, x, y, seed, 0.3, false)
		c := mix(over, under, t)
		return mul(c, 1-0.18*(1-float32(math.Abs(float64(t*2-1)))))
	case PatGravel:
		d1, d2, id := voronoi(x, y, 14, seed)
		c := paint(base, x, y, seed, 0.2, false)
		v := (0.7 + 0.5*hash2(id, 0, seed)) * (0.6 + 0.4*sstep(0, 2.5, d2-d1)) * (1 + 0.1*(1-sstep(0, 8, d1)))
		return mul(c, v)
	case PatPlanks:
		row := floorI(y / 16)
		off := float32((row * 23) % 64)
		bx := fmod(x+off, 32)
		ddx, ddy := min(bx, 32-bx), min(fmod(y, 16), 16-fmod(y, 16))
		board := hash2(row, floorI((x+off)/32), seed)
		c := paint(mix(mul(base, 0.92), mul(base, 1.08), board), x, y, seed+row, 0.18, false)
		v := seam(ddy, 2.2) * seam(ddx, 1.8)
		v *= 0.9 + 0.2*tnoise(x, y, 2, 24, seed+9) // grain
		return mul(c, v)
	case PatLogSide:
		c := paint(base, x, y, seed, 0.4, true)
		r := tnoise(x, y, 16, 2, seed+3)
		return mul(c, 0.75+0.5*r)
	case PatLogTop:
		dx, dy := x-paintT/2, y-paintT/2
		d := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		bark := paint(blocks[Log].Side, x, y, seed, 0.35, false)
		if d > paintT/2-2 {
			return bark
		}
		ring := float32(math.Sin(float64(d*0.9 + 2*tnoise(x, y, 3, 3, seed))))
		c := paint(base, x, y, seed, 0.1, false)
		c = mul(c, 0.86+0.12*ring)
		return mix(c, bark, sstep(paintT/2-5, paintT/2-2, d))
	case PatLeaves, PatFruit:
		if tnoise(x, y, 16, 16, seed+9) < 0.2 {
			return rl.NewColor(0, 0, 0, 0) // small gaps between leaves
		}
		c := paint(base, x, y, seed, 0.45, false)
		c = mul(c, 0.8+0.5*tnoise(x, y, 7, 7, seed+13))
		c.A = 255
		if p == PatFruit {
			d1, _, id := voronoi(x, y, 10, seed+5)
			if hash2(id, 3, seed) < 0.45 && d1 < 4 {
				return mix(rl.NewColor(215, 40, 40, 255), rl.NewColor(255, 120, 90, 255), 1-d1/4) // an apple
			}
		}
		return c
	case PatBricks:
		row := floorI(y / 16)
		bx := fmod(x+float32((row%2)*16), 32)
		ddx, ddy := min(bx, 32-bx), min(fmod(y, 16), 16-fmod(y, 16))
		m := sstep(0, 2.5, min(ddx, ddy))
		brick := paint(mix(mul(base, 0.88), mul(base, 1.1), hash2(row, floorI((x+float32((row%2)*16))/32), seed)), x, y, seed, 0.2, false)
		mortar := paint(mix(mul(base, 0.6), rl.NewColor(150, 140, 130, 255), 0.5), x, y, seed+1, 0.15, false)
		return mix(mortar, brick, m)
	case PatCobble:
		d1, d2, id := voronoi(x, y, 8, seed)
		c := paint(mul(base, 0.8+0.4*hash2(id, 0, seed)), x, y, seed+id, 0.2, false)
		v := (0.55 + 0.45*sstep(0, 2.2, d2-d1)) * (1 + 0.12*(1-sstep(0, 12, d1)))
		return mul(c, v)
	case PatOre:
		stone := texel(PatNoise, blocks[Stone].Side, &blocks[Stone], px, py, seed)
		o := tfbm(x, y, 10, seed+3)
		k := sstep(0.64, 0.7, o)
		fleck := mul(info.Spot, 0.85+0.3*tnoise(x, y, 10, 10, seed+6))
		return mix(stone, fleck, k)
	case PatGlass:
		e := min(x, y, paintT-x, paintT-y)
		pane := rl.NewColor(200, 232, 245, 40)
		if s := float32(math.Sin(float64((x + y) * 2 * math.Pi / 32))); s > 0.75 && x < paintT*0.6 {
			pane = mix(pane, rl.NewColor(255, 255, 255, 120), sstep(0.75, 0.9, s))
		}
		return mix(rl.NewColor(225, 238, 245, 255), pane, sstep(1.5, 4, e))
	case PatWater:
		c := paint(base, x, y, seed, 0.25, false)
		w := float32(math.Sin(float64((x + 8*tnoise(x, y, 2, 2, seed+4)) * 2 * math.Pi * 2 / paintT)))
		c = mul(c, 1+0.06*w)
		c.A = 170
		return c
	case PatGold:
		cx, cy := fmod(x, 16), fmod(y, 16)
		e := min(cx, 16-cx, cy, 16-cy)
		v := 0.8 + 0.3*sstep(0, 3, e)
		if (floorI(x/16)+floorI(y/16))%2 == 0 {
			v += 0.12
		}
		return mul(paint(base, x, y, seed, 0.15, false), v)
	case PatBedrock:
		return mul(base, 0.4+1.0*tfbm(x, y, 8, seed))
	case PatLava:
		v := tfbm(x, y, 3, seed)
		c := mix(rl.NewColor(255, 220, 60, 255), base, v)
		if k := tfbm(x, y, 6, seed+3); k > 0.7 {
			c = mix(c, mul(base, 0.5), sstep(0.7, 0.78, k)) // crust
		}
		return c
	case PatWool:
		c := paint(base, x, y, seed, 0.12, false)
		return mul(c, 0.92+0.16*tnoise(x, y, 4, 24, seed+8))
	case PatBirchSide:
		c := paint(base, x, y, seed, 0.12, true)
		m := tnoise(x, y, 6, 14, seed+5)
		return mix(c, mul(rl.NewColor(40, 40, 40, 255), 0.8+0.4*tnoise(x, y, 20, 20, seed+6)), sstep(0.74, 0.8, m))
	case PatCactus:
		c := paint(base, x, y, seed, 0.2, true)
		r := float32(math.Sin(float64(x * 2 * math.Pi * 4 / paintT)))
		v := 0.8 + 0.2*r
		if tnoise(x, y, 16, 16, seed+2) > 0.9 {
			v += 0.3 // spines
		}
		return mul(c, v)
	case PatCactusTop:
		e := min(x, y, paintT-x, paintT-y)
		return mix(paint(blocks[Cactus].Side, x, y, seed, 0.2, false), paint(base, x, y, seed, 0.2, false), sstep(5, 8, e))
	case PatMeteor:
		c := paint(base, x, y, seed, 0.5, false)
		vv := float32(math.Abs(float64(tnoise(x, y, 4, 4, seed+4) - 0.5)))
		return mix(mul(rl.NewColor(255, 150, 60, 255), 0.8+0.4*tnoise(x, y, 8, 8, seed+5)), c, sstep(0, 0.06, vv))
	case PatAmethyst:
		a := fmod(x+y, 16)
		b := fmod(x-y+paintT, 32)
		da, db := min(a, 16-a), min(b, 32-b)
		facet := sstep(0, 1.5, min(da, db))
		return mix(mul(rl.NewColor(230, 200, 255, 255), 0.9+0.2*tnoise(x, y, 8, 8, seed)), paint(base, x, y, seed, 0.35, false), facet)
	}
	return texelShape(p, base, info, px, py, seed)
}

// texelShape draws items, plants and built blocks on the 16x16 design grid,
// with painterly fills instead of per-pixel noise.
func texelShape(p texPattern, base rl.Color, info *blockInfo, px, py, seed int) rl.Color {
	fx, fy := float32(px)+0.5, float32(py)+0.5
	n := tfbm(fx, fy, 4, seed)*0.6 + brush(fx, fy, false, seed+3)*0.4
	x, y := px/subT, py/subT
	switch p {
	case PatTNT:
		// Red block with a white band reading as a label.
		if y >= 6 && y <= 9 {
			if y == 7 || y == 8 {
				if x%3 == 1 && x > 2 && x < 13 {
					return rl.NewColor(40, 30, 30, 255)
				}
			}
			return mul(rl.NewColor(235, 230, 220, 255), 0.9+0.15*n)
		}
		return mul(base, 0.85+0.3*n)
	case PatTNTTop:
		if x < 2 || y < 2 || x > 13 || y > 13 {
			return mul(base, 0.85+0.3*n)
		}
		return mul(rl.NewColor(110, 90, 70, 255), 0.85+0.3*n)
	case PatMeat:
		if hash2(x/3, y/3, seed+5) < 0.3 {
			return mul(rl.NewColor(240, 200, 190, 255), 0.9+0.2*n)
		}
		return mul(base, 0.8+0.35*n)
	case PatBedTop:
		if y < 5 {
			return mul(rl.NewColor(240, 240, 235, 255), 0.9+0.15*n) // pillow
		}
		if x == 0 || x == 15 || y == 15 {
			return mul(base, 0.6)
		}
		return mul(base, 0.85+0.3*n)
	case PatSapling:
		if y > 11 && x > 6 && x < 9 {
			return mul(rl.NewColor(100, 75, 40, 255), 0.85+0.3*n) // stem
		}
		if (x-7)*(x-7)+(y-6)*(y-6) < 30 {
			return mul(base, 0.75+0.5*n)
		}
		return mul(base, 0.5)
	case PatLadder:
		if x < 3 || x > 12 || y%4 == 1 || y%4 == 2 {
			return mul(base, 0.85+0.3*n) // rails and rungs; the rest is open air
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatTuft:
		// Blades of grass on a transparent background.
		blade := (x*7+seed)%5 == 0 || (x*3+seed)%7 == 1
		if blade && y > 3+int(hash2(x, 0, seed)*8) {
			return mul(base, 0.75+0.4*n)
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatFlower:
		if x > 6 && x < 9 && y > 7 {
			return mul(rl.NewColor(70, 140, 50, 255), 0.85+0.3*n) // stem
		}
		if (x-7)*(x-7)+(y-5)*(y-5) < 9 {
			if (x-7)*(x-7)+(y-5)*(y-5) < 2 {
				return rl.NewColor(250, 220, 60, 255)
			}
			return mul(base, 0.85+0.3*n)
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatSpawner:
		if x%4 == 0 || y%4 == 0 {
			return mul(base, 0.8+0.3*n) // cage bars
		}
		return rl.NewColor(15, 15, 20, 255)
	case PatCrate:
		if x == 0 || y == 0 || x == 15 || y == 15 || y == 7 || y == 8 {
			return mul(rl.NewColor(90, 70, 40, 255), 0.85+0.3*n) // iron bands
		}
		if x == 7 || x == 8 {
			return mul(rl.NewColor(220, 190, 80, 255), 0.9+0.2*n) // clasp
		}
		return mul(base, 0.85+0.3*hash2(x/3, y, seed))
	case PatApple:
		dx, dy := float32(x)-7.5, float32(y)-8.5
		if dx*dx+dy*dy < 36 {
			return mul(base, 0.8+0.35*n)
		}
		if y < 4 && x >= 7 && x <= 8 {
			return mul(rl.NewColor(100, 70, 40, 255), 0.9+0.2*n) // stalk
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatCrop0, PatCrop1, PatCrop2:
		height := 4
		if p == PatCrop1 {
			height = 9
		} else if p == PatCrop2 {
			height = 14
		}
		stalk := (x*5+seed)%4 == 1
		if stalk && y >= 16-height {
			if p == PatCrop2 && y < 16-height+5 {
				return mul(rl.NewColor(225, 195, 90, 255), 0.85+0.3*n) // ripe heads
			}
			return mul(base, 0.75+0.4*n)
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatBread:
		dx, dy := float32(x)-7.5, float32(y)-8
		if dx*dx/36+dy*dy/14 < 1 {
			if dy < -1 {
				return mul(rl.NewColor(230, 190, 120, 255), 0.9+0.2*n)
			}
			return mul(base, 0.85+0.3*n)
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatBow:
		// A curve on the left, the string on the right.
		dx, dy := float32(x)-13, float32(y)-7.5
		d := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		if d > 8.5 && d < 10.5 && x < 11 {
			return mul(base, 0.85+0.3*n)
		}
		if x == 12 && y > 1 && y < 14 {
			return rl.NewColor(235, 235, 225, 255)
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatArrow:
		if y == 7 || y == 8 {
			if x < 3 {
				return rl.NewColor(120, 120, 125, 255) // head
			}
			if x > 12 {
				return rl.NewColor(230, 230, 230, 255) // fletching
			}
			return mul(base, 0.85+0.3*n)
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatDoor:
		if x == 0 || x == 15 || y == 0 || y == 15 || y == 7 || x == 7 {
			return mul(base, 0.6+0.1*n) // frame and panel edges
		}
		if y >= 2 && y <= 5 && x >= 9 && x <= 13 {
			return rl.NewColor(190, 225, 240, 255) // window
		}
		if y == 9 && x == 2 {
			return rl.NewColor(230, 200, 60, 255) // handle
		}
		return mul(base, 0.85+0.3*hash2(x/3, y, seed))
	case PatBeacon:
		dx, dy := x-7, y-7
		if dx*dx+dy*dy < 12 {
			return mul(rl.NewColor(230, 250, 255, 255), 0.9+0.15*n) // glowing core
		}
		if x == 0 || y == 0 || x == 15 || y == 15 || (x+y)%8 == 0 {
			return mul(rl.NewColor(40, 60, 80, 255), 0.9+0.2*n) // dark frame
		}
		return mul(base, 0.85+0.3*n)
	case PatFish:
		dx, dy := float32(x)-6, float32(y)-8
		if dx*dx/25+dy*dy/6 < 1 {
			if x == 4 && y == 7 {
				return rl.NewColor(20, 20, 20, 255) // eye
			}
			return mul(base, 0.8+0.4*n)
		}
		if x >= 11 && x <= 14 && y >= 5 && y <= 11 && abs(y-8) <= x-10 {
			return mul(base, 0.7+0.3*n) // tail
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatRod:
		if x == 15-y && y > 2 {
			return mul(base, 0.85+0.3*n) // the rod, diagonal
		}
		if x == 3 && y >= 1 && y <= 12 {
			return rl.NewColor(230, 230, 225, 255) // line
		}
		if x == 3 && y == 13 {
			return rl.NewColor(220, 60, 60, 255) // bobber
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatShroom:
		if y <= 7 && (x-7)*(x-7)+(y-6)*(y-6)*2 < 30 {
			if hash2(x, y, seed+2) < 0.2 {
				return rl.NewColor(230, 250, 255, 255) // glowing spots
			}
			return mul(base, 0.85+0.3*n)
		}
		if y > 7 && x >= 6 && x <= 9 {
			return mul(rl.NewColor(200, 210, 200, 255), 0.85+0.3*n) // stalk
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatBoat:
		// A hull seen from the side: curved bottom, flat top.
		dx := float32(x) - 7.5
		if y >= 6 && float32(y) < 10+float32(math.Sqrt(float64(36-dx*dx*0.9))) {
			if y == 6 || y == 7 {
				return mul(base, 0.6)
			}
			return mul(base, 0.85+0.3*hash2(x/3, y, seed))
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatFlag:
		if x <= 1 {
			return mul(rl.NewColor(120, 90, 50, 255), 0.85+0.3*n) // pole
		}
		if y >= 2 && y <= 9 && x <= 13 {
			return mul(base, 0.85+0.3*n) // banner
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatTorch:
		if y < 6 {
			return mul(blocks[Torch].Top, 0.85+0.3*n)
		}
		return mul(base, 0.8+0.3*n)
	default:
		return paint(base, fx, fy, seed, 0.3, false)
	}
}

// ---------- classic pixel style ----------

// wrapDist16 is the toroidal distance between two pixels of a 16x16 tile, so cobble tiles seamlessly.
func wrapDist16(ax, ay, bx, by float32) float32 {
	dx := float32(math.Abs(float64(ax - bx)))
	dy := float32(math.Abs(float64(ay - by)))
	dx = min(dx, 16-dx)
	dy = min(dy, 16-dy)
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

// texelPixel is the original look: 16x16 pixel-art tiles with per-pixel noise.
func texelPixel(p texPattern, base rl.Color, info *blockInfo, x, y, seed int) rl.Color {
	n := hash2(x, y, seed)
	switch p {
	case PatSand:
		v := 0.9 + 0.2*n
		if hash2(x/2, y/2, seed+1) < 0.15 {
			v -= 0.08
		}
		return mul(base, v)
	case PatGrassTop:
		v := 0.8 + 0.35*n
		if n > 0.93 {
			v = 0.62
		}
		return mul(base, v)
	case PatGrassSide:
		edge := 2 + int(hash2(x, 0, seed)*3.5)
		if y < edge {
			return mul(blocks[Grass].Top, 0.8+0.35*hash2(x, y, seed+2))
		}
		return mul(base, 0.85+0.3*n)
	case PatSnowSide:
		edge := 4 + int(hash2(x, 0, seed)*2.5)
		if y < edge {
			return mul(blocks[Snow].Top, 0.92+0.12*n)
		}
		return mul(base, 0.85+0.3*n)
	case PatGravel:
		c := hash2(x/2, y/2, seed)
		return mul(base, 0.65+0.6*c)
	case PatPlanks:
		row := y / 4
		off := (row * 5) % 16
		v := 0.9 + 0.2*hash2((x+off)/3, row, seed)
		if y%4 == 0 {
			v *= 0.62
		} else if (x+off)%8 == 0 {
			v *= 0.72
		}
		return mul(base, v)
	case PatLogSide:
		v := 0.72 + 0.4*hash2(x, y/5, seed)
		return mul(base, v)
	case PatLogTop:
		dx, dy := float32(x)-7.5, float32(y)-7.5
		d := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		if d > 7.4 {
			return mul(blocks[Log].Side, 0.8+0.25*n)
		}
		v := float32(0.95)
		if int(d*1.1)%2 == 1 {
			v = 0.74
		}
		return mul(base, v+0.08*n)
	case PatLeaves:
		if hash2(x, y, seed+9) < 0.16 {
			return rl.NewColor(0, 0, 0, 0) // gaps between leaves
		}
		v := 0.6 + 0.6*n
		if n < 0.08 {
			v = 0.3
		}
		return mul(base, v)
	case PatBricks:
		by := y / 4
		bx := (x + (by%2)*4) % 8
		if y%4 == 3 || bx == 7 {
			return mul(base, 0.62+0.1*n)
		}
		v := 0.85 + 0.25*hash2((x+(by%2)*4)/8, by, seed) + 0.08*n
		return mul(base, v)
	case PatCobble:
		var d1, d2 float32 = 99, 99
		id := 0
		for i := 0; i < 7; i++ {
			px, py := hash2(i, 1, seed)*16, hash2(i, 2, seed)*16
			d := wrapDist16(float32(x), float32(y), px, py)
			if d < d1 {
				d2, d1, id = d1, d, i
			} else if d < d2 {
				d2 = d
			}
		}
		v := 0.75 + 0.4*hash2(id, 0, seed)
		if d2-d1 < 1.2 {
			v *= 0.6
		}
		return mul(base, v*(0.92+0.16*n))
	case PatOre:
		if hash2(x/2, y/2, seed+3) < 0.2 {
			return mul(info.Spot, 0.8+0.4*n)
		}
		return mul(blocks[Stone].Side, 0.85+0.3*n)
	case PatGlass:
		if x == 0 || y == 0 || x == 16-1 || y == 16-1 {
			return rl.NewColor(225, 238, 245, 255)
		}
		if (x+y)%7 == 0 && x < 9 {
			return rl.NewColor(255, 255, 255, 120)
		}
		return rl.NewColor(200, 232, 245, 40)
	case PatWater:
		v := 0.8 + 0.3*hash2(x/2, y/2, seed)
		c := mul(base, v)
		c.A = 170
		return c
	case PatGold:
		v := 0.88 + 0.2*n
		if (x/4+y/4)%2 == 0 {
			v += 0.14
		}
		if x%4 == 0 || y%4 == 0 {
			v -= 0.12
		}
		return mul(base, v)
	case PatBedrock:
		return mul(base, 0.4+1.0*n)
	case PatTNT:
		// Red block with a white band reading as a label.
		if y >= 6 && y <= 9 {
			if y == 7 || y == 8 {
				if x%3 == 1 && x > 2 && x < 13 {
					return rl.NewColor(40, 30, 30, 255)
				}
			}
			return mul(rl.NewColor(235, 230, 220, 255), 0.9+0.15*n)
		}
		return mul(base, 0.85+0.3*n)
	case PatTNTTop:
		if x < 2 || y < 2 || x > 13 || y > 13 {
			return mul(base, 0.85+0.3*n)
		}
		return mul(rl.NewColor(110, 90, 70, 255), 0.85+0.3*n)
	case PatMeat:
		if hash2(x/3, y/3, seed+5) < 0.3 {
			return mul(rl.NewColor(240, 200, 190, 255), 0.9+0.2*n)
		}
		return mul(base, 0.8+0.35*n)
	case PatLava:
		v := vnoise(float32(x)/4, float32(y)/4, seed)
		c := mix(rl.NewColor(255, 220, 60, 255), base, v)
		if hash2(x/2, y/2, seed+3) < 0.12 {
			c = mul(base, 0.55)
		}
		return c
	case PatWool:
		return mul(base, 0.88+0.18*hash2(x/2, y/2, seed)+0.05*n)
	case PatBedTop:
		if y < 5 {
			return mul(rl.NewColor(240, 240, 235, 255), 0.9+0.15*n) // pillow
		}
		if x == 0 || x == 16-1 || y == 16-1 {
			return mul(base, 0.6)
		}
		return mul(base, 0.85+0.3*n)
	case PatSapling:
		if y > 11 && x > 6 && x < 9 {
			return mul(rl.NewColor(100, 75, 40, 255), 0.85+0.3*n) // stem
		}
		if (x-7)*(x-7)+(y-6)*(y-6) < 30 {
			return mul(base, 0.75+0.5*n)
		}
		return mul(base, 0.5)
	case PatLadder:
		if x < 3 || x > 12 || y%4 == 1 || y%4 == 2 {
			return mul(base, 0.85+0.3*n) // rails and rungs; the rest is open air
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatBirchSide:
		if hash2(x/2, y/3, seed) < 0.18 {
			return mul(rl.NewColor(40, 40, 40, 255), 0.8+0.4*n) // bark marks
		}
		return mul(base, 0.9+0.15*n)
	case PatCactus:
		if x%5 == 0 {
			return mul(base, 0.6+0.1*n) // ribs
		}
		return mul(base, 0.85+0.3*n)
	case PatCactusTop:
		if x < 2 || y < 2 || x > 13 || y > 13 {
			return mul(blocks[Cactus].Side, 0.8+0.2*n)
		}
		return mul(base, 0.85+0.3*n)
	case PatTuft:
		// Blades of grass on a transparent background.
		blade := (x*7+seed)%5 == 0 || (x*3+seed)%7 == 1
		if blade && y > 3+int(hash2(x, 0, seed)*8) {
			return mul(base, 0.75+0.4*n)
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatFlower:
		if x > 6 && x < 9 && y > 7 {
			return mul(rl.NewColor(70, 140, 50, 255), 0.85+0.3*n) // stem
		}
		if (x-7)*(x-7)+(y-5)*(y-5) < 9 {
			if (x-7)*(x-7)+(y-5)*(y-5) < 2 {
				return rl.NewColor(250, 220, 60, 255)
			}
			return mul(base, 0.85+0.3*n)
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatSpawner:
		if x%4 == 0 || y%4 == 0 {
			return mul(base, 0.8+0.3*n) // cage bars
		}
		return rl.NewColor(15, 15, 20, 255)
	case PatCrate:
		if x == 0 || y == 0 || x == 16-1 || y == 16-1 || y == 7 || y == 8 {
			return mul(rl.NewColor(90, 70, 40, 255), 0.85+0.3*n) // iron bands
		}
		if x == 7 || x == 8 {
			return mul(rl.NewColor(220, 190, 80, 255), 0.9+0.2*n) // clasp
		}
		return mul(base, 0.85+0.3*hash2(x/3, y, seed))
	case PatApple:
		dx, dy := float32(x)-7.5, float32(y)-8.5
		if dx*dx+dy*dy < 36 {
			return mul(base, 0.8+0.35*n)
		}
		if y < 4 && x >= 7 && x <= 8 {
			return mul(rl.NewColor(100, 70, 40, 255), 0.9+0.2*n) // stalk
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatCrop0, PatCrop1, PatCrop2:
		height := 4
		if p == PatCrop1 {
			height = 9
		} else if p == PatCrop2 {
			height = 14
		}
		stalk := (x*5+seed)%4 == 1
		if stalk && y >= 16-height {
			if p == PatCrop2 && y < 16-height+5 {
				return mul(rl.NewColor(225, 195, 90, 255), 0.85+0.3*n) // ripe heads
			}
			return mul(base, 0.75+0.4*n)
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatBread:
		dx, dy := float32(x)-7.5, float32(y)-8
		if dx*dx/36+dy*dy/14 < 1 {
			if dy < -1 {
				return mul(rl.NewColor(230, 190, 120, 255), 0.9+0.2*n)
			}
			return mul(base, 0.85+0.3*n)
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatBow:
		// A curve on the left, the string on the right.
		dx, dy := float32(x)-13, float32(y)-7.5
		d := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		if d > 8.5 && d < 10.5 && x < 11 {
			return mul(base, 0.85+0.3*n)
		}
		if x == 12 && y > 1 && y < 14 {
			return rl.NewColor(235, 235, 225, 255)
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatArrow:
		if y == 7 || y == 8 {
			if x < 3 {
				return rl.NewColor(120, 120, 125, 255) // head
			}
			if x > 12 {
				return rl.NewColor(230, 230, 230, 255) // fletching
			}
			return mul(base, 0.85+0.3*n)
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatDoor:
		if x == 0 || x == 16-1 || y == 0 || y == 16-1 || y == 7 || x == 7 {
			return mul(base, 0.6+0.1*n) // frame and panel edges
		}
		if y >= 2 && y <= 5 && x >= 9 && x <= 13 {
			return rl.NewColor(190, 225, 240, 255) // window
		}
		if y == 9 && x == 2 {
			return rl.NewColor(230, 200, 60, 255) // handle
		}
		return mul(base, 0.85+0.3*hash2(x/3, y, seed))
	case PatBeacon:
		dx, dy := x-7, y-7
		if dx*dx+dy*dy < 12 {
			return mul(rl.NewColor(230, 250, 255, 255), 0.9+0.15*n) // glowing core
		}
		if x == 0 || y == 0 || x == 16-1 || y == 16-1 || (x+y)%8 == 0 {
			return mul(rl.NewColor(40, 60, 80, 255), 0.9+0.2*n) // dark frame
		}
		return mul(base, 0.85+0.3*n)
	case PatFish:
		dx, dy := float32(x)-6, float32(y)-8
		if dx*dx/25+dy*dy/6 < 1 {
			if x == 4 && y == 7 {
				return rl.NewColor(20, 20, 20, 255) // eye
			}
			return mul(base, 0.8+0.4*n)
		}
		if x >= 11 && x <= 14 && y >= 5 && y <= 11 && abs(y-8) <= x-10 {
			return mul(base, 0.7+0.3*n) // tail
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatRod:
		if x == 15-y && y > 2 {
			return mul(base, 0.85+0.3*n) // the rod, diagonal
		}
		if x == 3 && y >= 1 && y <= 12 {
			return rl.NewColor(230, 230, 225, 255) // line
		}
		if x == 3 && y == 13 {
			return rl.NewColor(220, 60, 60, 255) // bobber
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatShroom:
		if y <= 7 && (x-7)*(x-7)+(y-6)*(y-6)*2 < 30 {
			if hash2(x, y, seed+2) < 0.2 {
				return rl.NewColor(230, 250, 255, 255) // glowing spots
			}
			return mul(base, 0.85+0.3*n)
		}
		if y > 7 && x >= 6 && x <= 9 {
			return mul(rl.NewColor(200, 210, 200, 255), 0.85+0.3*n) // stalk
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatAmethyst:
		if (x+y)%5 == 0 || (x-y+16)%7 == 0 {
			return mul(rl.NewColor(230, 200, 255, 255), 0.9+0.2*n) // crystal facets
		}
		return mul(base, 0.8+0.35*n)
	case PatMeteor:
		if hash2(x/2, y/2, seed+4) < 0.18 {
			return mul(rl.NewColor(255, 150, 60, 255), 0.8+0.4*n) // glowing veins
		}
		return mul(base, 0.6+0.6*n)
	case PatFruit:
		if hash2(x, y, seed+9) < 0.16 {
			return rl.NewColor(0, 0, 0, 0)
		}
		if hash2(x/3, y/3, seed+5) < 0.14 && x%3 == 1 && y%3 == 1 {
			return rl.NewColor(215, 40, 40, 255) // apples
		}
		v := 0.6 + 0.6*n
		if n < 0.08 {
			v = 0.3
		}
		return mul(base, v)
	case PatBoat:
		// A hull seen from the side: curved bottom, flat top.
		dx := float32(x) - 7.5
		if y >= 6 && float32(y) < 10+float32(math.Sqrt(float64(36-dx*dx*0.9))) {
			if y == 6 || y == 7 {
				return mul(base, 0.6)
			}
			return mul(base, 0.85+0.3*hash2(x/3, y, seed))
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatFlag:
		if x <= 1 {
			return mul(rl.NewColor(120, 90, 50, 255), 0.85+0.3*n) // pole
		}
		if y >= 2 && y <= 9 && x <= 13 {
			return mul(base, 0.85+0.3*n) // banner
		}
		return rl.NewColor(0, 0, 0, 0)
	case PatTorch:
		if y < 6 {
			return mul(blocks[Torch].Top, 0.85+0.3*n)
		}
		return mul(base, 0.8+0.3*n)
	default:
		return mul(base, 0.85+0.3*n)
	}
}
