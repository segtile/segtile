package util

import (
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	"golang.org/x/image/tiff"
)

func LoadImage(path string) image.Image {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	return src
}

func ToNRGBA(src image.Image) *image.NRGBA {
	if n, ok := src.(*image.NRGBA); ok {
		return n
	}
	b := src.Bounds()
	dst := image.NewNRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	return dst
}

func ToGray16(src image.Image) *image.Gray16 {
	if g, ok := src.(*image.Gray16); ok {
		return g
	}
	b := src.Bounds()
	dst := image.NewGray16(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	return dst
}

func SaveImage(path string, img image.Image) {
	f, err := os.Create(filepath.Clean(path))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	ext := filepath.Ext(path)
	switch ext {
	case ".png":
		err = png.Encode(f, img)
	case ".jpg", ".jpeg":
		err = jpeg.Encode(f, img, &jpeg.Options{Quality: 99})
	case ".tiff", ".tif":
		err = tiff.Encode(f, img, nil)
	default:
		panic("Unsupported file format: " + ext)
	}
	if err != nil {
		panic(err)
	}
}
