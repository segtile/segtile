package main

import (
	"encoding/csv"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"runtime"
	"sync"

	_ "golang.org/x/image/tiff"

	"github.com/struffel/annotation-tool/superpixel"
	"github.com/struffel/annotation-tool/superpixel_factory"
	"github.com/struffel/annotation-tool/util"
)

type benchmarkRun struct {
	threshold float64
	beta      float64
}

type benchmarkResult struct {
	imageName        string
	threshold        float64
	beta             float64
	score            float64
	overlapHistogram []int
}

func main() {

	// Show help
	if len(os.Args) < 4 {
		fmt.Println("Usage: bench <image_directory> <gt_mask_directory> <output_csv>")
		return
	}

	// Read image directory and GT mask directory
	images, err := util.ListFiles(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing image files: %v\n", err)
		return
	}

	masks, err := util.ListFiles(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing mask files: %v\n", err)
		return
	}

	if len(images) != len(masks) {
		fmt.Fprintf(os.Stderr, "Number of images and masks do not match: %d vs %d\n", len(images), len(masks))
		os.Exit(1)
	}

	// Prepare CSV writer
	outputFile := os.Args[3]
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating CSV file: %v\n", err)
		return
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	writer.Comma = ';'
	defer writer.Flush()

	for index_img := range images {
		fmt.Fprintf(os.Stderr, "Processing image %d/%d: %s with mask %s\n", index_img+1, len(images), images[index_img], masks[index_img])

		// Load images
		img := util.ToNRGBA(util.LoadImage(images[index_img]))
		groundTruthMask := util.ToGray16(util.LoadImage(masks[index_img]))

		// Configuration
		const algorithmName = "scale_adaptive"

		var work []benchmarkRun
		for threshold := 0.0; threshold <= 10.0; threshold += 0.1 {
			for beta := 0.0; beta <= 2.0; beta += 0.25 {
				work = append(work, benchmarkRun{threshold, beta})
			}
		}

		// Process work in parallel
		numWorkers := runtime.NumCPU()
		workCh := make(chan benchmarkRun, len(work))
		for _, w := range work {
			workCh <- w
		}
		close(workCh)

		resultCh := make(chan benchmarkResult, len(work))

		var wg sync.WaitGroup
		for index_worker := 0; index_worker < numWorkers; index_worker++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Set up superpixel algorithm
				sp, _ := superpixel_factory.GetSuperpixelImplementationByName(algorithmName)
				sp.SetImage(img)
				sp.SetRecommendedParameters()

				for item := range workCh {
					sp.SetParameter(superpixel.Threshold, item.threshold)
					sp.SetParameter(superpixel.Beta, item.beta)

					if err := sp.Run(); err != nil {
						fmt.Fprintf(os.Stderr, "Error running superpixel algorithm: %v\n", err)
						return
					}

					spResult := sp.GetResult()
					approximatedMask, overlapHistogram := spResult.ApproximateMask(groundTruthMask, 0.5)
					score, _ := util.DiceCoefficient(groundTruthMask, approximatedMask)

					fmt.Fprintf(os.Stderr, ".")

					resultCh <- benchmarkResult{imageName: images[index_img], threshold: item.threshold, beta: item.beta, score: score, overlapHistogram: overlapHistogram}
				}
			}()
		}

		wg.Wait()
		close(resultCh)
		fmt.Println()

		results := make([]benchmarkResult, 0, len(work))
		for r := range resultCh {
			results = append(results, r)
		}

		newCsvLines := make([][]string, len(results))
		for i, r := range results {
			newCsvLines[i] = []string{fmt.Sprintf("%s", r.imageName), fmt.Sprintf("%.1f", r.threshold), fmt.Sprintf("%.2f", r.beta), fmt.Sprintf("%.4f", r.score)}
			for _, v := range r.overlapHistogram {
				newCsvLines[i] = append(newCsvLines[i], fmt.Sprintf("%d", v))
			}
		}
		writer.WriteAll(newCsvLines)
	}
}
