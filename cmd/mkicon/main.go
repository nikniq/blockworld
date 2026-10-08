// mkicon renders the Blockworld app icon: an isometric grass block with a
// pixel texture, written to icon.png (1024x1024) at the repository root.
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

const size = 1024

func hash(x, y, seed int) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + uint32(seed)*1274126177
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h&0xffff) / 65535
}

func shade(c color.RGBA, v float64) color.RGBA {
	cl := func(f float64) uint8 {
		if f < 0 {
			return 0
		}
		if f > 255 {
			return 255
		}
		return uint8(f)
	}
	return color.RGBA{cl(float64(c.R) * v), cl(float64(c.G) * v), cl(float64(c.B) * v), 255}
}

func main() {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	grass := color.RGBA{112, 176, 66, 255}
	dirt := color.RGBA{124, 92, 58, 255}
	cx, cy := float64(size)/2, float64(size)*0.5
	s := float64(size) * 0.36 // cube edge in pixels (projected)
	// Isometric axes: x goes down-right, z goes down-left, y goes up.
	ax := [2]float64{s * math.Cos(math.Pi/6), s * math.Sin(math.Pi/6)}
	az := [2]float64{-s * math.Cos(math.Pi/6), s * math.Sin(math.Pi/6)}
	ay := [2]float64{0, -s}
	// Rounded dark backdrop so the icon reads on light and dark docks.
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			dx, dy := float64(px)-cx, float64(py)-float64(size)/2
			r := float64(size) * 0.47
			if math.Abs(dx) < r && math.Abs(dy) < r {
				// squircle-ish corners
				ex, ey := math.Max(math.Abs(dx)-r*0.75, 0), math.Max(math.Abs(dy)-r*0.75, 0)
				if ex*ex+ey*ey <= (r*0.25)*(r*0.25) {
					img.SetRGBA(px, py, color.RGBA{34, 40, 52, 255})
				}
			}
		}
	}
	// Three visible faces: top (grass), left (dirt with grass edge), right.
	type face struct {
		o, u, v [2]float64
		paint   func(tx, ty int) color.RGBA
		light   float64
	}
	top := [2]float64{cx, cy - s*0.55}
	faces := []face{
		{top, ax, az, func(tx, ty int) color.RGBA {
			v := 0.8 + 0.35*hash(tx, ty, 1)
			if hash(tx, ty, 1) > 0.93 {
				v = 0.62
			}
			return shade(grass, v)
		}, 1.0},
		{[2]float64{top[0] + az[0], top[1] + az[1]}, ax, [2]float64{-ay[0], -ay[1]}, func(tx, ty int) color.RGBA {
			edge := 2 + int(hash(tx, 0, 2)*3.5)
			if ty < edge {
				return shade(grass, 0.8+0.35*hash(tx, ty, 3))
			}
			return shade(dirt, 0.85+0.3*hash(tx, ty, 2))
		}, 0.72},
		{[2]float64{top[0] + ax[0] + az[0], top[1] + ax[1] + az[1]}, [2]float64{-az[0], -az[1]}, [2]float64{-ay[0], -ay[1]}, func(tx, ty int) color.RGBA {
			edge := 2 + int(hash(tx, 7, 2)*3.5)
			if ty < edge {
				return shade(grass, 0.8+0.35*hash(tx, ty, 5))
			}
			return shade(dirt, 0.85+0.3*hash(tx, ty, 4))
		}, 0.85},
	}
	for _, f := range faces {
		const n = 16
		for ty := 0; ty < n; ty++ {
			for tx := 0; tx < n; tx++ {
				c := shade(f.paint(tx, ty), f.light)
				// Rasterise the texel as a small parallelogram by sampling sub-pixels.
				for sy := 0; sy < 40; sy++ {
					for sx := 0; sx < 40; sx++ {
						u := (float64(tx) + float64(sx)/40) / n
						v := (float64(ty) + float64(sy)/40) / n
						x := f.o[0] + f.u[0]*u + f.v[0]*v
						y := f.o[1] + f.u[1]*u + f.v[1]*v
						if x >= 0 && y >= 0 && int(x) < size && int(y) < size {
							img.SetRGBA(int(x), int(y), c)
						}
					}
				}
			}
		}
	}
	out, err := os.Create("icon.png")
	if err != nil {
		panic(err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		panic(err)
	}
}
