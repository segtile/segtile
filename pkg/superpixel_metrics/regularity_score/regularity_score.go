package regularity_score

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/circularity"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func init() {
	superpixel_metrics.RegisterSpMetric(
		"regularity_score",
		"Regularity Score",
		"Computes the regularity score of the superpixel segmentation.",
		1.0,
		runWithParams,
		nil,
	)
}

func runWithParams(result superpixel.SpResult, img image.Image, groundTruth *image.Gray16, params map[string]float64) (score float64, err error) {
	return Run(result, groundTruth)
}

func Run(result superpixel.SpResult, groundTruth *image.Gray16) (regularityScore float64, err error) {

	// Check that images are valid
	if result.LabelMap.Bounds() != groundTruth.Bounds() {
		return 0, fmt.Errorf("Ground truth and superpixel label map must have the same dimensions")
	}

	// Approximate gt using superpixels
	selection := result.SelectFromMask(groundTruth, 0.5)

	// Count SP used to approximate GT
	usedSuperpixelCount := 0
	for i := 0; i < len(selection); i++ {
		if selection[i] {
			usedSuperpixelCount++
		}
	}

	// Find islands in GT
	groundTruthIslands, groundTruthIslandCount := superpixel_utils.FindIslands(groundTruth)

	// Calculate circularity of GT islands
	gtCircularity, err := circularity.Run(superpixel.SpResult{LabelMap: groundTruthIslands, SpCount: int(groundTruthIslandCount)})
	if err != nil {
		return 0, fmt.Errorf("Error calculating circularity of ground truth islands: %v", err)
	}

	// Build superpixel map containing only superpixels that are in the selection and are touching the outside of the selection
	// neighbours := superpixel_utils.LabelMapNeighborhoods(result.LabelMap, result.SpCount)
	selectedSuperpixels := result.Clone()
	for y := 0; y < selectedSuperpixels.LabelMap.Bounds().Dy(); y++ {
		for x := 0; x < selectedSuperpixels.LabelMap.Bounds().Dx(); x++ {
			spId := selectedSuperpixels.LabelMap.Gray16At(x, y).Y
			keep := true

			// Not selected itself -> remove
			if !selection[spId] {
				keep = false
			} /*else {
				// Not touching the outside of the selection -> remove
				keep = false
				for _, neighbour := range neighbours[spId] {
					if !selection[neighbour] {
						keep = true
						break
					}
				}
			}*/

			if !keep {
				selectedSuperpixels.LabelMap.SetGray16(x, y, color.Gray16{Y: 0})
			}
		}
	}
	selectedSuperpixels.SetActualSpCount()

	// Calculate circularity of superpixels
	spCircularity, err := circularity.Run(selectedSuperpixels)
	if err != nil {
		return 0, fmt.Errorf("Error calculating circularity of superpixels: %v", err)
	}

	if spCircularity == 0 && gtCircularity == 0 {
		return 0, fmt.Errorf("Superpixel and ground truth circularity is zero, cannot compute regularity score")
	}

	//log.Printf("Superpixel circularity: %f, Ground truth circularity: %f", spCircularity, gtCircularity)

	regularityScore = math.Min(spCircularity, gtCircularity) / math.Max(spCircularity, gtCircularity)

	return regularityScore, nil

}
