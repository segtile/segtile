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

	"github.com/struffel/annotation-tool/superpixel"
	"github.com/struffel/annotation-tool/superpixel_factory"
	"github.com/struffel/annotation-tool/util"
)

type benchmarkRun struct {
	imagePath  string
	parameters map[superpixel.SpParameter]float64
}

type benchmarkResult struct {
	imagePath        string
	actualSpCount    int
	parameters       map[superpixel.SpParameter]float64
	score            float64
	overlapHistogram []int
}

func buildRunParameterSets(parameterRanges []util.ParameterRange) ([]map[superpixel.SpParameter]float64, error) {
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
			parameter, err := superpixel.ParseSpParameter(pR.Name)
			if err != nil {
				err := fmt.Errorf("Error parsing superpixel parameter: %v", err)
				return nil, err
			}
			runConfigurations[i][parameter] = rangeValues[valueIndex]
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
	_, err := superpixel_factory.GetSuperpixelImplementationByName(algorithmName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Read image directory and GT mask directory
	images, err := util.ListFiles(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing image files: %v\n", err)
		os.Exit(1)
	}

	masks, err := util.ListFiles(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing mask files: %v\n", err)
		os.Exit(1)
	}

	if len(images) != len(masks) {
		fmt.Fprintf(os.Stderr, "Number of images and masks does not match: %d vs %d\n", len(images), len(masks))
		os.Exit(1)
	}

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

	// Prepare the header
	csvHeader := []string{"ImageName", "Algorithm", "Actual Superpixel Count", "Dice Score"}
	for param := range runParameters[0] {
		csvHeader = append(csvHeader, param.String())
	}
	for i := 0; i <= 100; i++ {
		csvHeader = append(csvHeader, fmt.Sprintf("OverlapPercentage_%d", i))
	}
	writer.Write(csvHeader)
	writer.Flush()

	// Process each image and mask pair
	for index_img := range images {
		fmt.Fprintf(os.Stderr, "Processing image %d/%d: %s with mask %s\n", index_img+1, len(images), images[index_img], masks[index_img])

		// Load images
		img := util.ToNRGBA(util.LoadImage(images[index_img]))
		groundTruthMask := util.ToGray16(util.LoadImage(masks[index_img]))

		// Build list of benchmark run tasks
		var work []benchmarkRun
		for _, config := range runParameters {
			work = append(work, benchmarkRun{imagePath: images[index_img], parameters: config})
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

					// Set parameters
					for param, value := range item.parameters {
						sp.SetParameter(param, value)
					}

					// Perform the actual run
					if err := sp.Run(); err != nil {
						fmt.Fprintf(os.Stderr, "Error running superpixel algorithm: %v\n", err)
						return
					}

					// Get the result and compare it to the baseline mask
					spResult := sp.GetResult()
					approximatedMask, overlapHistogram := spResult.ApproximateMask(groundTruthMask, 0.5)
					score, _ := util.DiceCoefficient(groundTruthMask, approximatedMask)

					// Print progress indicator
					fmt.Fprintf(os.Stderr, ".")

					// Send the result out
					resultCh <- benchmarkResult{
						imagePath:        images[index_img],
						parameters:       item.parameters,
						actualSpCount:    spResult.SpCount,
						score:            score,
						overlapHistogram: overlapHistogram}
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
			newCsvLines[i] = []string{fmt.Sprintf("%s", imageName), fmt.Sprintf("%s", algorithmName), fmt.Sprintf("%d", r.actualSpCount), fmt.Sprintf("%.4f", r.score)}
			for _, parameterValue := range r.parameters {
				newCsvLines[i] = append(newCsvLines[i], fmt.Sprintf("%v", parameterValue))
			}
			for _, v := range r.overlapHistogram {
				newCsvLines[i] = append(newCsvLines[i], fmt.Sprintf("%d", v))
			}
		}

		writer.WriteAll(newCsvLines)
	}
}
