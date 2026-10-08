package main

import (
	"image"
	"image/color"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Block textures are generated at startup into one atlas: 16x16 pixel tiles,
// three per block (top, side, bottom), so the game ships without asset files.

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
)

const (
	tileSize   = 16
	atlasPad   = 8                     // border of repeated edge pixels around each tile, so mipmaps do not bleed
	atlasCell  = tileSize + 2*atlasPad // pitch of one tile in the atlas
	atlasTiles = 8                     // tiles per row
)

var atlasRows = (int(numBlocks)*3 + atlasTiles - 1) / atlasTiles

func tileIndex(b Block, face int) int { return int(b)*3 + face }

// tileUV returns the atlas rectangle of a block face, inset slightly so point
// sampling never bleeds into the neighbouring tile.
func tileUV(b Block, face int) (u0, v0, u1, v1 float32) {
	i := tileIndex(b, face)
	tx, ty := float32(i%atlasTiles)*atlasCell+atlasPad, float32(i/atlasTiles)*atlasCell+atlasPad
	aw, ah := float32(atlasTiles*atlasCell), float32(atlasRows*atlasCell)
	const inset = 0.05
	return (tx + inset) / aw, (ty + inset) / ah, (tx + tileSize - inset) / aw, (ty + tileSize - inset) / ah
}

// TileRect returns the atlas rectangle of a block face in pixels (for the HUD).
func TileRect(b Block, face int) rl.Rectangle {
	i := tileIndex(b, face)
	return rl.NewRectangle(float32(i%atlasTiles*atlasCell+atlasPad), float32(i/atlasTiles*atlasCell+atlasPad), tileSize, tileSize)
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
					c := texel(info.Pat[f], bases[f], info, tx, ty, seed)
					img.SetRGBA(ox+x, oy+y, color.RGBA{c.R, c.G, c.B, c.A})
				}
			}
		}
	}
	return img
}

// wrapDist is the toroidal distance between two tile pixels, so cobble tiles seamlessly.
func wrapDist(ax, ay, bx, by float32) float32 {
	dx := float32(math.Abs(float64(ax - bx)))
	dy := float32(math.Abs(float64(ay - by)))
	dx = min(dx, tileSize-dx)
	dy = min(dy, tileSize-dy)
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

func texel(p texPattern, base rl.Color, info *blockInfo, x, y, seed int) rl.Color {
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
			px, py := hash2(i, 1, seed)*tileSize, hash2(i, 2, seed)*tileSize
			d := wrapDist(float32(x), float32(y), px, py)
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
		if x == 0 || y == 0 || x == tileSize-1 || y == tileSize-1 {
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
		if x == 0 || x == tileSize-1 || y == tileSize-1 {
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
		if x == 0 || y == 0 || x == tileSize-1 || y == tileSize-1 || y == 7 || y == 8 {
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
		if stalk && y >= tileSize-height {
			if p == PatCrop2 && y < tileSize-height+5 {
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
		if x == 0 || x == tileSize-1 || y == 0 || y == tileSize-1 || y == 7 || x == 7 {
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
		if x == 0 || y == 0 || x == tileSize-1 || y == tileSize-1 || (x+y)%8 == 0 {
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
	case PatTorch:
		if y < 6 {
			return mul(blocks[Torch].Top, 0.85+0.3*n)
		}
		return mul(base, 0.8+0.3*n)
	default:
		return mul(base, 0.85+0.3*n)
	}
}
