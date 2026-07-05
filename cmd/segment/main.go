package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/lucasb-eyer/go-colorful"
	_ "golang.org/x/image/tiff"

	"github.com/struffel/annotation-tool/internal/cli"
	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/pkg/superpixel"
	_ "github.com/struffel/annotation-tool/pkg/superpixel/slic"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func renderSegmentationBorders(img image.Image, labelMap *image.Gray16, borderColor colorful.Color) *image.NRGBA {
	segmentation := io_utils.CopyImage(img)
	for _, point := range superpixel_utils.BoundaryPoints(labelMap, true) {
		segmentation.Set(point.X, point.Y, borderColor)
	}
	return segmentation
}

func main() {

	// Show help
	if len(os.Args) < 6 {
		fmt.Println("Usage: segment <image_path> <label_map_out_path> <segmentation_out_path> <segmentation_color> <algorithm_name> <parameter1=value> ...")
		os.Exit(1)
	}

	imagePath := os.Args[1]
	labelMapOutPath := os.Args[2]
	segmentationOutPath := os.Args[3]
	segmentationColor := os.Args[4]
	algorithmName := os.Args[5]

	parameters, err := cli.LoadSuperpixelParametersFromCLI(algorithmName, os.Args[6:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing algorithm parameters: %v\n", err)
		os.Exit(1)
	}

	borderColor, err := colorful.Hex(segmentationColor)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing segmentation color: %v\n", err)
		os.Exit(1)
	}

	img := io_utils.LoadImage(imagePath)
	result, err := superpixel.Run(algorithmName, img, parameters)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running superpixel algorithm: %v\n", err)
		os.Exit(1)
	}

	io_utils.SaveImage(labelMapOutPath, result.LabelMap)
	io_utils.SaveImage(segmentationOutPath, renderSegmentationBorders(img, result.LabelMap, borderColor))

}
