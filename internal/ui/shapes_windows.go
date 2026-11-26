//go:build windows

package ui

import "math"

func blendPixel(pix []byte, idx int, argb uint32, cov float64) {
	if idx < 0 || idx+3 >= len(pix) || cov <= 0 {
		return
	}
	srcA := float64(byte(argb>>24)) / 255.0 * cov
	if srcA <= 0 {
		return
	}
	if srcA > 1 {
		srcA = 1
	}
	sr := float64(byte(argb >> 16))
	sg := float64(byte(argb >> 8))
	sb := float64(byte(argb))
	inv := 1.0 - srcA

	pix[idx] = byte(math.Round(sb*srcA + float64(pix[idx])*inv))
	pix[idx+1] = byte(math.Round(sg*srcA + float64(pix[idx+1])*inv))
	pix[idx+2] = byte(math.Round(sr*srcA + float64(pix[idx+2])*inv))
	outA := srcA + (float64(pix[idx+3])/255.0)*inv
	if outA > 1 {
		outA = 1
	}
	pix[idx+3] = byte(math.Round(outA * 255.0))
}

func roundedRectSDF(px, py, x0, y0, x1, y1, r float64) float64 {
	cx := (x0 + x1) * 0.5
	cy := (y0 + y1) * 0.5
	hx := (x1-x0)*0.5 - r
	hy := (y1-y0)*0.5 - r
	dx := math.Abs(px-cx) - hx
	dy := math.Abs(py-cy) - hy
	ox := math.Max(dx, 0)
	oy := math.Max(dy, 0)
	outside := math.Hypot(ox, oy)
	inside := math.Min(math.Max(dx, dy), 0)
	return outside + inside - r
}

// FillRect desenha retangulo solido com suporte a alpha.
func (c *Canvas) FillRect(x, y, w, h int, argb uint32) {
	if w <= 0 || h <= 0 || len(c.pixels) == 0 {
		return
	}
	x0 := max(0, x)
	y0 := max(0, y)
	x1 := min(c.Width, x+w)
	y1 := min(c.Height, y+h)
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			blendPixel(c.pixels, (py*c.Width+px)*4, argb, 1.0)
		}
	}
}

// FillRoundRect desenha retangulo com cantos arredondados via SDF anti-aliased.
func (c *Canvas) FillRoundRect(x, y, w, h, radius int, argb uint32) {
	if w <= 0 || h <= 0 || len(c.pixels) == 0 {
		return
	}
	r := float64(radius)
	maxR := math.Min(float64(w), float64(h)) * 0.5
	if r > maxR {
		r = maxR
	}
	if r < 0 {
		r = 0
	}
	x0 := max(0, x-1)
	y0 := max(0, y-1)
	x1 := min(c.Width, x+w+1)
	y1 := min(c.Height, y+h+1)
	fx0, fy0, fx1, fy1 := float64(x), float64(y), float64(x+w), float64(y+h)

	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			d := roundedRectSDF(float64(px)+0.5, float64(py)+0.5, fx0, fy0, fx1, fy1, r)
			if d < 0.75 {
				cov := math.Min(1, math.Max(0, 0.75-d))
				blendPixel(c.pixels, (py*c.Width+px)*4, argb, cov)
			}
		}
	}
}

// StrokeRoundRect desenha borda de 1px+ com cantos arredondados e anti-aliasing.
func (c *Canvas) StrokeRoundRect(x, y, w, h, radius int, strokeW float64, argb uint32) {
	if w <= 0 || h <= 0 || strokeW <= 0 || len(c.pixels) == 0 {
		return
	}
	r := float64(radius)
	maxR := math.Min(float64(w), float64(h)) * 0.5
	if r > maxR {
		r = maxR
	}
	half := strokeW * 0.5
	x0 := max(0, x-2)
	y0 := max(0, y-2)
	x1 := min(c.Width, x+w+2)
	y1 := min(c.Height, y+h+2)
	fx0, fy0, fx1, fy1 := float64(x), float64(y), float64(x+w), float64(y+h)

	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			d := math.Abs(roundedRectSDF(float64(px)+0.5, float64(py)+0.5, fx0, fy0, fx1, fy1, r)) - half
			if d < 0.75 {
				cov := math.Min(1, math.Max(0, 0.75-d))
				blendPixel(c.pixels, (py*c.Width+px)*4, argb, cov)
			}
		}
	}
}

func distSegment(px, py, ax, ay, bx, by float64) float64 {
	vx, vy := bx-ax, by-ay
	wx, wy := px-ax, py-ay
	c1 := wx*vx + wy*vy
	if c1 <= 0 {
		return math.Hypot(px-ax, py-ay)
	}
	c2 := vx*vx + vy*vy
	if c2 <= c1 {
		return math.Hypot(px-bx, py-by)
	}
	b := c1 / c2
	return math.Hypot(px-(ax+b*vx), py-(ay+b*vy))
}

// DrawCheckmark desenha um icone vetorial de check dentro da caixa (x,y,size).
func (c *Canvas) DrawCheckmark(x, y, size int, argb uint32) {
	s := float64(size)
	ax, ay := float64(x)+s*0.22, float64(y)+s*0.52
	bx, by := float64(x)+s*0.43, float64(y)+s*0.74
	cx, cy := float64(x)+s*0.78, float64(y)+s*0.28
	thick := math.Max(1.4, s*0.12)
	for py := max(0, y); py < min(c.Height, y+size); py++ {
		for px := max(0, x); px < min(c.Width, x+size); px++ {
			d1 := distSegment(float64(px)+0.5, float64(py)+0.5, ax, ay, bx, by)
			d2 := distSegment(float64(px)+0.5, float64(py)+0.5, bx, by, cx, cy)
			d := math.Min(d1, d2) - thick*0.5
			if d < 0.75 {
				cov := math.Min(1, math.Max(0, 0.75-d))
				blendPixel(c.pixels, (py*c.Width+px)*4, argb, cov)
			}
		}
	}
}
