package main

import (
	"fmt"

	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/dice_coefficient"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/distance_weighted_dice_coefficient"
)

func main() {

	groundTruth := "S:/Google-Drive/Studium/Master/Masterarbeit/F1 Demo/raw/new_gt.png"
	gtImage := io_utils.ToGray16(io_utils.LoadImage(groundTruth))
	approx := "S:/Google-Drive/Studium/Master/Masterarbeit/F1 Demo/raw/124_slic_150_0.png"
	apxImage := io_utils.ToGray16(io_utils.LoadImage(approx))

	result := superpixel.SpResult{}
	result.LabelMap = apxImage
	result.SetActualSpCount()
	result.EnforceContiguousIndexing()

	f1_score, _ := dice_coefficient.Run(result, gtImage)
	weighted_f1, _ := distance_weighted_dice_coefficient.Run(result, gtImage)

	fmt.Println("F1 Score:", f1_score)
	fmt.Println("Weighted F1 Score:", weighted_f1)
}
