package main

import (
	"image"
	"image/color"
	"log"
	"math"
	"math/cmplx"
	"time"

	mandel "github.com/marben/irpc_dist_mandel"
)

const maxIter = 1000

type RendererImpl struct{}

func (imp RendererImpl) RenderTile(r mandel.Region, tile image.Rectangle, imgW, imgH int) (image.RGBA, error) {
	log.Printf("rendering tile: %s", tile)

	// Image now has global coordinates (tile.Min .. tile.Max)
	img := image.NewRGBA(tile)

	for py := tile.Min.Y; py < tile.Max.Y; py++ {
		yf := r.Ymin + (float64(py)/float64(imgH))*(r.Ymax-r.Ymin)

		for pxg := tile.Min.X; pxg < tile.Max.X; pxg++ {
			xf := r.Xmin + (float64(pxg)/float64(imgW))*(r.Xmax-r.Xmin)

			c := complex(xf, yf)

			mu, trap := MandelbrotOrbit(c, maxIter)

			var col color.RGBA
			if mu >= float64(maxIter) {
				col = color.RGBA{A: 255}
			} else {
				tnorm := math.Exp(-5 * trap)
				hue := math.Mod(mu*0.02+tnorm*0.3, 1.0)
				col = hsv(hue, 1, 1)
			}

			img.SetRGBA(pxg, py, col)
		}
	}

	time.Sleep(500 * time.Millisecond)

	return *img, nil
}

var _ mandel.Renderer = RendererImpl{}

func MandelbrotSmooth(c complex128, maxIter int) float64 {
	z := complex(0, 0)

	for i := range maxIter {
		z = z*z + c
		if real(z)*real(z)+imag(z)*imag(z) > 4 {
			// Smooth iteration count
			mu := float64(i) + 1.0 - math.Log(math.Log(cmplx.Abs(z)))/math.Log(2)
			return mu
		}
	}
	return float64(maxIter)
}

func MandelbrotOrbit(c complex128, maxIter int) (smooth float64, trap float64) {
	z := complex(0, 0)
	minTrap := math.MaxFloat64

	for i := 0; i < maxIter; i++ {
		z = z*z + c

		// Example trap: distance to imaginary axis (Re=0)
		d := math.Abs(real(z))
		if d < minTrap {
			minTrap = d
		}

		if real(z)*real(z)+imag(z)*imag(z) > 4 {
			// Smooth escape
			smooth = float64(i) + 1 - math.Log(math.Log(cmplx.Abs(z)))/math.Log(2)
			return smooth, minTrap
		}
	}

	// Inside the set
	return float64(maxIter), minTrap
}

// Mandelbrot iteration with smooth coloring + circular orbit trap
func MandelbrotCircleTrap(c complex128, maxIter int, R float64) (smooth float64, trap float64) {
	z := complex(0, 0)
	minTrap := math.MaxFloat64

	for i := 0; i < maxIter; i++ {
		z = z*z + c

		// -----------------------------------------
		// Circular orbit trap: distance to |z| = R
		// -----------------------------------------
		d := math.Abs(cmplx.Abs(z) - R)
		if d < minTrap {
			minTrap = d
		}

		// Escape check
		if real(z)*real(z)+imag(z)*imag(z) > 4 {
			smooth = float64(i) + 1 - math.Log(math.Log(cmplx.Abs(z)))/math.Log(2)
			return smooth, minTrap
		}
	}

	// Inside the set
	return float64(maxIter), minTrap
}

// Simple HSV → RGB
func hsv(h, s, v float64) color.RGBA {
	h = math.Mod(h, 1)
	i := int(h * 6)
	f := h*6 - float64(i)
	p := v * (1 - s)
	q := v * (1 - f*s)
	t := v * (1 - (1-f)*s)

	var r, g, b float64
	switch i % 6 {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	case 5:
		r, g, b = v, p, q
	}
	return color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), 255}
}
