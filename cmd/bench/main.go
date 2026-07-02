package main

import (
	"encoding/csv"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	_ "golang.org/x/image/tiff"

	"github.com/struffel/annotation-tool/internal/cli"
	util "github.com/struffel/annotation-tool/internal/cli"
	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/internal/utils"
	"github.com/struffel/annotation-tool/pkg/superpixel"
)

type benchmarkRun struct {
	imagePath  string
	parameters map[superpixel.SpParameter]float64
}

type benchmarkResult struct {
	imagePath                  string
	actualSpCount              int
	parameters                 map[superpixel.SpParameter]float64
	f1Score                    float64
	averageBoundaryDistance    float64
	selectedSpCount            int
	selectedBorderingSpCount   int
	labelingCalibrationScore   float64
	densityCalibrationScore    float64
	regularityCalibrationScore float64
}

func buildRunParameterSets(parameterRanges []cli.ParameterRange) ([]map[superpixel.SpParameter]float64, error) {
	// Determine how many results there will be
	numResults := 1
	for _, pR := range parameterRanges {
		numResults *= len(pR.Values())
	}

	// Allocate results
	var runConfigurations []map[superpixel.SpParameter]float64 = make([]map[superpixel.SpParameter]float64, numResults)
	for i := range runConfigurations {
		runConfigurations[i] = make(map[superpixel.SpParameter]float64)
	}

	// Create runConfigurations, containing one parameter from each range
	step := 1
	for _, pR := range parameterRanges {
		rangeValues := pR.Values()
		for i := 0; i < numResults; i++ {
			valueIndex := (i / step) % len(rangeValues)
			runConfigurations[i][superpixel.SpParameter(pR.Name)] = rangeValues[valueIndex]
		}
		step *= len(rangeValues)
	}

	return runConfigurations, nil
}

func main() {

	// Show help
	if len(os.Args) < 5 {
		fmt.Println("Usage: bench <image_directory> <gt_mask_directory> <output_csv> <algorithm_name> <parameter1=start,end,step> ...")
		os.Exit(1)
	}

	// Read parameters
	algorithmName := os.Args[4]
	_, err := superpixel_factory.NewSuperpixelImplementationByName(algorithmName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Read image directory and GT mask directory
	images, err := io_utils.ListFiles(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing image files: %v\n", err)
		os.Exit(1)
	}

	masks, err := io_utils.ListFiles(os.Args[2])
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
	parameterRanges, err := util.ParseParameterRanges(os.Args[5:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing parameter ranges: %v\n", err)
		os.Exit(1)
	}

	runParameters, err := buildRunParameterSets(parameterRanges)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building run configurations: %v\n", err)
		os.Exit(1)
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
	parameterKeys := make([]superpixel.SpParameter, 0, len(runParameters[0]))
	for param := range runParameters[0] {
		parameterKeys = append(parameterKeys, param)
	}
	//sort.Slice(parameterKeys, func(i, j int) bool { return parameterKeys[i] < parameterKeys[j] })

	// Prepare the header
	csvHeader := []string{
		"ImageName",
		"Algorithm",
		"Actual Superpixel Count",
		"Selected Superpixel Count",
		"Selected Bordering Superpixel Count",
		"Average Boundary Distance",
		"F1 Score",
		"Density Calibration Score",
		"Regularity Calibration Score",
		"Labeling Calibration Score",
	}
	for _, param := range parameterKeys {
		csvHeader = append(csvHeader, param.Title())
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
		if !evaluation.HasNonZeroPixels(groundTruthMask) {
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

				// Set up superpixel algorithm
				sp, _ := superpixel_factory.NewSuperpixelImplementationByName(algorithmName)
				sp.SetImage(img)
				sp.SetRecommendedParameters()

				for item := range workCh {

					// Set parameters
					for param, value := range item.parameters {
						sp.SetParameter(param, value)
					}

					// Perform the actual run
					if err := sp.Run(); err != nil {
						fmt.Fprintf(os.Stderr, "Error running superpixel algorithm: %v\n", err)
						return
					}

					// Get the result
					spResult := sp.GetResult()

					labelingCalibrationScore,
						f1Score,
						densityCalibrationScore,
						regularityCalibrationScore,
						averageBoundaryDistance,
						selectedSpCount,
						selectedBorderingSpCount := evaluation.SuperpixelEvaluation(groundTruthMask, spResult, 5)

					// Print progress indicator
					fmt.Fprintf(os.Stderr, ".")

					// Send the result out
					resultCh <- benchmarkResult{
						imagePath:                images[index_img],
						parameters:               item.parameters,
						actualSpCount:            spResult.SpCount,
						selectedSpCount:          selectedSpCount,
						selectedBorderingSpCount: selectedBorderingSpCount,
						averageBoundaryDistance:  averageBoundaryDistance,

						f1Score:                    f1Score,
						densityCalibrationScore:    densityCalibrationScore,
						regularityCalibrationScore: regularityCalibrationScore,
						labelingCalibrationScore:   labelingCalibrationScore,
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
				fmt.Sprintf("%s", algorithmName),
				fmt.Sprintf("%d", r.actualSpCount),
				fmt.Sprintf("%d", r.selectedSpCount),
				fmt.Sprintf("%d", r.selectedBorderingSpCount),
				fmt.Sprintf("%.4f", r.averageBoundaryDistance),
				fmt.Sprintf("%.4f", r.f1Score),
				fmt.Sprintf("%.4f", r.densityCalibrationScore),
				fmt.Sprintf("%.4f", r.regularityCalibrationScore),
				fmt.Sprintf("%.4f", r.labelingCalibrationScore)}
			for _, param := range parameterKeys {
				newCsvLines[i] = append(newCsvLines[i], fmt.Sprintf("%v", r.parameters[param]))
			}
		}

		writer.WriteAll(newCsvLines)
	}
}
