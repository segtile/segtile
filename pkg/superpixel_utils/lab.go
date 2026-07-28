package superpixel_utils

import (
	"image"
	"runtime"
	"sync"

	"github.com/lucasb-eyer/go-colorful"
)

func BuildLab(img image.Image, labCache *[][3]float64) {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	if *labCache == nil || len(*labCache) != width*height {
		*labCache = make([][3]float64, width*height)
	}

	workers := runtime.NumCPU()
	var wg sync.WaitGroup
	wg.Add(workers)

	for w := range workers {
		rowStart := w * height / workers
		rowEnd := (w + 1) * height / workers

		go func(rowStart, rowEnd int) {
			defer wg.Done()
			for y := rowStart; y < rowEnd; y++ {
				for x := 0; x < width; x++ {
					c, _ := colorful.MakeColor(img.At(x, y))
					l, a, b := c.Lab()
					// go-colorful returns LAB values in a different range than standard CIELAB (L in [0,1] instead of the official [0,100]),
					// so we scale the lab values by 100 to match the scale the compactness parameter is calibrated for.
					l *= 100
					a *= 100
					b *= 100
					(*labCache)[y*width+x] = [3]float64{l, a, b}
				}
			}
		}(rowStart, rowEnd)
	}

	wg.Wait()
}
