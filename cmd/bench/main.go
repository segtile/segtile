package main

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"runtime"
	"sort"
	"sync"

	_ "golang.org/x/image/tiff"

	"github.com/struffel/annotation-tool/superpixel/slic"
	"github.com/struffel/annotation-tool/util"
)

type benchmarkRun struct {
	compactness float64
	spCount     int
}

type benchmarkResult struct {
	spCount          int
	compactness      float64
	score            float64
	overlapHistogram []int
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: bench <image_path> <gt_mask_path>")
		return
	}

	// Load images
	img := util.ToNRGBA(util.LoadImage(os.Args[1]))
	groundTruthMask := util.ToGray16(util.LoadImage(os.Args[2]))

	//fmt.Fprintln(os.Stderr, "Loaded image and mask.")

	// Build work queue
	var work []benchmarkRun
	for compactness := 0.0; compactness <= 30.0; compactness += 2 {
		for spCount := 25; spCount <= 2000; spCount += 50 {
			work = append(work, benchmarkRun{compactness, spCount})
		}
	}

	numWorkers := runtime.NumCPU()
	workCh := make(chan benchmarkRun, len(work))
	for _, w := range work {
		workCh <- w
	}
	close(workCh)

	resultCh := make(chan benchmarkResult, len(work))

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sp := slic.SpSlic{}
			sp.SetImage(img)
			sp.SetDefaultParameters()

			for item := range workCh {
				sp.SetCompactness(item.compactness)
				sp.SetNumSuperpixels(item.spCount)

				if err := sp.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "Error running superpixel algorithm: %v\n", err)
					return
				}

				spResult := sp.GetResult()
				approximatedMask, overlapHistogram := spResult.ApproximateMask(groundTruthMask, 0.5)
				score, _ := util.DiceCoefficient(groundTruthMask, approximatedMask)

				//fmt.Fprintf(os.Stderr, "%d,%.4f,%.4f\n", item.spCount, item.compactness, score)

				resultCh <- benchmarkResult{item.spCount, item.compactness, score, overlapHistogram}
			}
		}()
	}

	wg.Wait()
	close(resultCh)

	results := make([]benchmarkResult, 0, len(work))
	for r := range resultCh {
		results = append(results, r)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].compactness != results[j].compactness {
			return results[i].compactness < results[j].compactness
		}
		return results[i].spCount < results[j].spCount
	})

	//fmt.Println("Superpixels;Compactness;DiceCoefficient")
	for _, r := range results {
		fmt.Printf("%d;%.1f;%.4f;", r.spCount, r.compactness, r.score)
		for _, v := range r.overlapHistogram {
			fmt.Printf("%d;", v)
		}
		fmt.Println()
	}
}
