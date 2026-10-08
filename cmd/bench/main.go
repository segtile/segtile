package main

import (
	"encoding/csv"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	_ "golang.org/x/image/tiff"

	"github.com/struffel/annotation-tool/internal/cli"
	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/internal/utils"
	"github.com/struffel/annotation-tool/pkg/superpixel"
	_ "github.com/struffel/annotation-tool/pkg/superpixel/rss"
	_ "github.com/struffel/annotation-tool/pkg/superpixel/scale_adaptive"
	_ "github.com/struffel/annotation-tool/pkg/superpixel/slic"

	//_ "github.com/struffel/annotation-tool/pkg/superpixel/snic"
	_ "github.com/struffel/annotation-tool/pkg/superpixel/vcell"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	_ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/achievable_segmentation_accuracy"
	_ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/actual_superpixel_count"

	// _ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/average_boundary_distance"
	_ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/circularity"
	_ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/density_score"

	// _ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/density_score_legacy"
	_ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/dice_coefficient"
	_ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/labeling_calibration_score"

	//_ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/outer_boundary_pixel_count"
	_ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/regularity_score"
	//_ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/regularity_score_legacy"

	// _ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/selected_bordering_superpixel_count"
	// _ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/selected_superpixel_count"
	// _ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/total_boundary_pixel_count"
	_ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/distance_weighted_dice_coefficient"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

type benchmarkRun struct {
	imagePath  string
	parameters map[string]float64
}

type benchmarkResult struct {
	imagePath    string
	parameters   map[string]float64
	metricScores map[string]float64
}

func cloneParameterMap(params map[string]float64) map[string]float64 {
	cloned := make(map[string]float64, len(params))
	for key, value := range params {
		cloned[key] = value
	}
	return cloned
}

func buildRunParameterSets(defaultParameters map[string]float64, parameterRanges []cli.ParameterRange) ([]map[string]float64, error) {
	// Determine how many results there will be
	numResults := 1
	for _, pR := range parameterRanges {
		numResults *= len(pR.Values())
	}

	// Allocate results
	var runConfigurations []map[string]float64 = make([]map[string]float64, numResults)
	for i := range runConfigurations {
		runConfigurations[i] = cloneParameterMap(defaultParameters)
	}

	// Create runConfigurations, containing one parameter from each range
	step := 1
	for _, pR := range parameterRanges {
		rangeValues := pR.Values()
		for i := 0; i < numResults; i++ {
			valueIndex := (i / step) % len(rangeValues)
			runConfigurations[i][pR.Name] = rangeValues[valueIndex]
		}
		step *= len(rangeValues)
	}

	return runConfigurations, nil
}

func evaluateBenchmark(result superpixel.SpResult, img image.Image, groundTruthMask *image.Gray16, metricIDs []string, metricParams map[string]map[string]float64) (benchmarkResult, error) {
	metricScores := make(map[string]float64, len(metricIDs))
	for _, metricID := range metricIDs {
		score, err := superpixel_metrics.Run(metricID, result, img, groundTruthMask, metricParams[metricID])
		if err != nil {
			return benchmarkResult{}, fmt.Errorf("metric %s failed: %w", metricID, err)
		}
		metricScores[metricID] = score
	}

	return benchmarkResult{
		metricScores: metricScores,
	}, nil
}

func main() {

	// Show help
	if len(os.Args) < 5 {
		fmt.Println("Usage: bench <image_directory> <gt_mask_directory> <output_csv> <algorithm_name> <parameter1=start,end,step> ...")
		os.Exit(1)
	}

	// Read parameters
	algorithmName := os.Args[4]
	defaultParameters, err := cli.LoadDefaultSuperpixelParameters(algorithmName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Read image directory and GT mask directory
	images, err := io_utils.ListFiles(os.Args[1], true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing image files: %v\n", err)
		os.Exit(1)
	}

	masks, err := io_utils.ListFiles(os.Args[2], true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing mask files: %v\n", err)
		os.Exit(1)
	}

	if len(images) != len(masks) {
		fmt.Fprintf(os.Stderr, "Number of images and masks does not match: %d vs %d\n", len(images), len(masks))
		os.Exit(1)
	}

	// Shuffle the order of images and masks in the same way to avoid biasing results if the process is interrupted before completion
	utils.ShuffleInUnison(images, masks)

	// Build parameter ranges
	parameterRanges, err := cli.ParseParameterRanges(os.Args[5:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing parameter ranges: %v\n", err)
		os.Exit(1)
	}

	runParameters, err := buildRunParameterSets(defaultParameters, parameterRanges)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building run configurations: %v\n", err)
		os.Exit(1)
	}
	metricIDs := superpixel_metrics.ListSpMetrics()
	sort.Strings(metricIDs)
	metricParameters := make(map[string]map[string]float64, len(metricIDs))
	for _, metricID := range metricIDs {
		params, err := superpixel_metrics.GetDefaultSpMetricParameters(metricID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading metric defaults: %v\n", err)
			os.Exit(1)
		}
		metricParameters[metricID] = params
	}

	// Print configuration
	fmt.Printf("Benchmarking %s algorithm with %d image/mask pairs and %d configurations for each pair\n", os.Args[4], len(images), len(runParameters))

	// Prepare CSV writer
	outputFile := os.Args[3]
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating CSV file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	writer.Comma = ';'
	defer writer.Flush()

	// Establish a stable sorted order for parameters in the CSV output
	parameterKeys := make([]string, 0, len(runParameters[0]))
	for param := range runParameters[0] {
		parameterKeys = append(parameterKeys, param)
	}
	sort.Strings(parameterKeys)

	// Prepare the header
	csvHeader := []string{
		"image_name",
		"algorithm",
	}
	for _, param := range parameterKeys {
		csvHeader = append(csvHeader, param)
	}
	for _, metricID := range metricIDs {
		csvHeader = append(csvHeader, metricID)
	}

	writer.Write(csvHeader)
	writer.Flush()

	// Process each image and mask pair
	for index_img := range images {
		fmt.Fprintf(os.Stderr, "Processing image %d/%d: %s with mask %s\n", index_img+1, len(images), images[index_img], masks[index_img])

		// Load images
		img := io_utils.ToNRGBA(io_utils.LoadImage(images[index_img]))
		groundTruthMask := io_utils.ToGray16(io_utils.LoadImage(masks[index_img]))

		// Ensure that the image and mask have the same dimensions
		if img.Bounds() != groundTruthMask.Bounds() {
			fmt.Fprintf(os.Stderr, "Warning: Image and mask dimensions do not match for %s and %s. Skipping this pair.\n", images[index_img], masks[index_img])
			continue
		}

		// Test if ground truth mask contains at least one non-zero pixel
		if !superpixel_utils.HasNonZeroPixels(groundTruthMask) {
			fmt.Fprintf(os.Stderr, "Warning: Ground truth mask %s contains no non-zero pixels. Skipping this pair.\n", masks[index_img])
			continue
		}

		// Build list of benchmark run tasks
		var work []benchmarkRun
		for _, config := range runParameters {
			work = append(work, benchmarkRun{imagePath: images[index_img], parameters: config})
		}

		// Process work in parallel
		numWorkers := runtime.NumCPU() / 2
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

				for item := range workCh {
					// Perform the actual run
					spResult, err := superpixel.Run(algorithmName, img, item.parameters)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error running superpixel algorithm: %v\n", err)
						return
					}
					spResult.EnforceConnectivity()

					metrics, err := evaluateBenchmark(spResult, img, groundTruthMask, metricIDs, metricParameters)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error evaluating superpixel result: %v\n", err)
						return
					}

					// Print progress indicator
					fmt.Fprintf(os.Stderr, ".")

					// Send the result out
					resultCh <- benchmarkResult{
						imagePath:    images[index_img],
						parameters:   item.parameters,
						metricScores: metrics.metricScores,
					}
				}
			}()
		}

		wg.Wait()
		close(resultCh)
		fmt.Println()

		// Collect results
		results := make([]benchmarkResult, 0, len(work))
		for r := range resultCh {
			results = append(results, r)
		}

		// Write results to CSV
		newCsvLines := make([][]string, len(results))

		for i, r := range results {
			imageName := filepath.Base(r.imagePath)
			newCsvLines[i] = []string{
				fmt.Sprintf("%s", imageName),
				fmt.Sprintf("%s", algorithmName)}
			for _, param := range parameterKeys {
				newCsvLines[i] = append(newCsvLines[i], fmt.Sprintf("%v", r.parameters[param]))
			}
			for _, metricID := range metricIDs {
				newCsvLines[i] = append(newCsvLines[i], fmt.Sprintf("%.4f", r.metricScores[metricID]))
			}
		}

		writer.WriteAll(newCsvLines)
	}
}
