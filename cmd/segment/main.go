package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucasb-eyer/go-colorful"
	_ "golang.org/x/image/tiff"

	"github.com/segtile/segtile/internal/cli"
	"github.com/segtile/segtile/internal/core"
	"github.com/segtile/segtile/pkg/superpixel"
	_ "github.com/segtile/segtile/pkg/superpixel/rss"
	_ "github.com/segtile/segtile/pkg/superpixel/scale_adaptive"
	_ "github.com/segtile/segtile/pkg/superpixel/slic"
	"github.com/segtile/segtile/pkg/superpixel_map"

	//_ "github.com/segtile/segtile/pkg/superpixel/snic"
	_ "github.com/segtile/segtile/pkg/superpixel/vcell"
	"github.com/segtile/segtile/pkg/superpixel_utils"
)

func renderSegmentationBorders(img image.Image, labelMap *superpixel_map.LabelMap, borderColor colorful.Color) *image.NRGBA {
	segmentation := core.CopyImage(img)
	for _, point := range superpixel_utils.BoundaryPoints(labelMap, true) {
		segmentation.Set(point.X, point.Y, borderColor)
	}
	return segmentation
}

func buildOutputPath(imagePath string, kind string, algorithmName string, parameterArgs []string) string {
	baseName := filepath.Base(imagePath)
	ext := filepath.Ext(baseName)
	originalName := strings.TrimSuffix(baseName, ext)

	parts := []string{originalName, kind, algorithmName}
	if len(parameterArgs) > 0 {
		parts = append(parts, parameterArgs...)
	}

	return filepath.Join(".", strings.Join(parts, "_")+".png")
}

func main() {

	// Show help
	if len(os.Args) < 4 {
		fmt.Println("Usage: segment <image_path> <segmentation_color> <algorithm_name> <parameter1=value> ...")
		os.Exit(1)
	}

	imagePath := os.Args[1]
	segmentationColor := os.Args[2]
	algorithmName := os.Args[3]
	parameterArgs := os.Args[4:]
	//labelMapOutPath := buildOutputPath(imagePath, "label", algorithmName, parameterArgs)
	segmentationOutPath := buildOutputPath(imagePath, "segmentation", algorithmName, parameterArgs)

	parameters, err := cli.LoadSuperpixelParametersFromCLI(algorithmName, parameterArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing algorithm parameters: %v\n", err)
		os.Exit(1)
	}

	borderColor, err := colorful.Hex(segmentationColor)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing segmentation color: %v\n", err)
		os.Exit(1)
	}

	img := core.LoadImage(imagePath)
	result, err := superpixel.Run(algorithmName, img, parameters)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running superpixel algorithm: %v\n", err)
		os.Exit(1)
	}

	core.SavePngImage(segmentationOutPath, renderSegmentationBorders(img, result.LabelMap, borderColor))

}
