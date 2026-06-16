package wasserstein_sp

/*
Superpixel-based image segmentation using squared 2-Wasserstein distances.

This implements the SP model from:
  Huang, J., Alpers, A., Chen, K., Lei, N. (2026)
  Superpixel-Based Image Segmentation Using Squared 2-Wasserstein Distances.

The algorithm uses SLIC for the initial oversegmentation (standing in for
Power-SLIC). Adjacent regions are greedily merged based on a regularized
optimal-transport cost between their color histograms.
*/

import (
	"container/heap"
	"errors"
	"image"
	"image/color"
	"math"

	"github.com/lucasb-eyer/go-colorful"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel/slic"
)

// ---------------------------------------------------------------------------
// Priority queue for merge candidates (min-heap by kappa)
// ---------------------------------------------------------------------------

type mergeCandidate struct {
	i, j  int
	kappa float64
	cost  float64
	index int
}

type mergeQueue []*mergeCandidate

func (mq mergeQueue) Len() int           { return len(mq) }
func (mq mergeQueue) Less(a, b int) bool { return mq[a].kappa < mq[b].kappa }
func (mq mergeQueue) Swap(a, b int) {
	mq[a], mq[b] = mq[b], mq[a]
	mq[a].index = a
	mq[b].index = b
}
func (mq *mergeQueue) Push(x any) {
	item := x.(*mergeCandidate)
	item.index = len(*mq)
	*mq = append(*mq, item)
}
func (mq *mergeQueue) Pop() any {
	old := *mq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*mq = old[:n-1]
	return item
}

// ---------------------------------------------------------------------------
// Region bookkeeping
// ---------------------------------------------------------------------------

type region struct {
	active        bool
	pixelCount    int
	hist          []float64
	heterogeneity float64
	neighbors     map[int]struct{}
}

// ---------------------------------------------------------------------------
// Sinkhorn solver for entropic OT
// ---------------------------------------------------------------------------

type otSolver struct {
	k          int
	cost       [][]float64
	kernel     [][]float64
	epsilon    float64
	iterations int
}

func newOtSolver(centers [][3]float64, epsilon float64, iterations int) *otSolver {
	k := len(centers)
	cost := make([][]float64, k)
	maxCost := 0.0
	for i := 0; i < k; i++ {
		cost[i] = make([]float64, k)
		for j := 0; j < k; j++ {
			dl := centers[i][0] - centers[j][0]
			da := centers[i][1] - centers[j][1]
			db := centers[i][2] - centers[j][2]
			d := dl*dl + da*da + db*db
			cost[i][j] = d
			if d > maxCost {
				maxCost = d
			}
		}
	}
	if maxCost <= 0 {
		maxCost = 1
	}
	for i := 0; i < k; i++ {
		for j := 0; j < k; j++ {
			cost[i][j] = cost[i][j] / maxCost
		}
	}
	kernel := make([][]float64, k)
	for i := 0; i < k; i++ {
		kernel[i] = make([]float64, k)
		for j := 0; j < k; j++ {
			kernel[i][j] = math.Exp(-cost[i][j] / epsilon)
		}
	}
	return &otSolver{k: k, cost: cost, kernel: kernel, epsilon: epsilon, iterations: iterations}
}

func (s *otSolver) distance(a, b []float64) float64 {
	k := s.k
	u := make([]float64, k)
	v := make([]float64, k)
	for i := 0; i < k; i++ {
		u[i] = 1
		v[i] = 1
	}

	const tiny = 1e-12
	for iter := 0; iter < s.iterations; iter++ {
		for i := 0; i < k; i++ {
			den := 0.0
			for j := 0; j < k; j++ {
				den += s.kernel[i][j] * v[j]
			}
			if den < tiny {
				den = tiny
			}
			u[i] = a[i] / den
		}
		for j := 0; j < k; j++ {
			den := 0.0
			for i := 0; i < k; i++ {
				den += s.kernel[i][j] * u[i]
			}
			if den < tiny {
				den = tiny
			}
			v[j] = b[j] / den
		}
	}

	cost := 0.0
	for i := 0; i < k; i++ {
		for j := 0; j < k; j++ {
			cost += u[i] * s.kernel[i][j] * v[j] * s.cost[i][j]
		}
	}
	return cost
}

// ---------------------------------------------------------------------------
// Disjoint set for final labeling
// ---------------------------------------------------------------------------

type dsu struct {
	parent []int
}

func newDsu(n int) *dsu {
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	return &dsu{parent: p}
}

func (d *dsu) find(x int) int {
	if d.parent[x] != x {
		d.parent[x] = d.find(d.parent[x])
	}
	return d.parent[x]
}

func (d *dsu) union(into, from int) {
	d.parent[from] = into
}

// ---------------------------------------------------------------------------
// SpWasserstein
// ---------------------------------------------------------------------------

type SpWasserstein struct {
	superpixel.SpAlgorithmData

	initialSuperpixels                    int
	targetRegions                         int
	compactness                           float64
	slicIterations                        int
	initialPerturbationNeighborhoodRadius int
	histogramBins                         int
	sinkhornEpsilon                       float64
	sinkhornIterations                    int

	labCache [][3]float64
}

func (sp *SpWasserstein) SetRecommendedParameters() {
	sp.initialSuperpixels = 2000
	sp.targetRegions = 200
	sp.compactness = 10
	sp.slicIterations = 8
	sp.initialPerturbationNeighborhoodRadius = 1
	sp.histogramBins = 15
	sp.sinkhornEpsilon = 0.05
	sp.sinkhornIterations = 30
}

func (sp *SpWasserstein) GetSupportedParameters() []superpixel.SpParameter {
	return []superpixel.SpParameter{
		superpixel.Count,
		superpixel.TargetRegionCount,
		superpixel.Compactness,
		superpixel.Iterations,
		superpixel.InitialPerturbationNeighborhoodRadius,
		superpixel.HistogramBins,
		superpixel.SinkhornEpsilon,
		superpixel.SinkhornIterations,
	}
}

func (sp *SpWasserstein) GetParameter(parameter superpixel.SpParameter) (float64, error) {
	switch parameter {
	case superpixel.Count:
		return float64(sp.initialSuperpixels), nil
	case superpixel.TargetRegionCount:
		return float64(sp.targetRegions), nil
	case superpixel.Compactness:
		return sp.compactness, nil
	case superpixel.Iterations:
		return float64(sp.slicIterations), nil
	case superpixel.InitialPerturbationNeighborhoodRadius:
		return float64(sp.initialPerturbationNeighborhoodRadius), nil
	case superpixel.HistogramBins:
		return float64(sp.histogramBins), nil
	case superpixel.SinkhornEpsilon:
		return sp.sinkhornEpsilon, nil
	case superpixel.SinkhornIterations:
		return float64(sp.sinkhornIterations), nil
	default:
		return 0, errors.New("Wasserstein SP does not have this parameter.")
	}
}

func (sp *SpWasserstein) SetParameter(parameter superpixel.SpParameter, value float64) error {
	switch parameter {
	case superpixel.Count:
		if value < 1 {
			return errors.New("Initial superpixel count must be at least 1.")
		}
		sp.initialSuperpixels = int(value)
		return nil
	case superpixel.TargetRegionCount:
		if value < 1 {
			return errors.New("Target region count must be at least 1.")
		}
		sp.targetRegions = int(value)
		return nil
	case superpixel.Compactness:
		if value < 0 {
			return errors.New("Compactness can not be negative.")
		}
		sp.compactness = value
		return nil
	case superpixel.Iterations:
		if value < 1 {
			return errors.New("Iterations must be at least 1.")
		}
		sp.slicIterations = int(value)
		return nil
	case superpixel.InitialPerturbationNeighborhoodRadius:
		if value < 0 {
			return errors.New("Initial perturbation neighborhood radius can not be negative.")
		}
		sp.initialPerturbationNeighborhoodRadius = int(value)
		return nil
	case superpixel.HistogramBins:
		if value < 2 {
			return errors.New("Histogram bins must be at least 2.")
		}
		sp.histogramBins = int(value)
		return nil
	case superpixel.SinkhornEpsilon:
		if value <= 0 {
			return errors.New("Sinkhorn epsilon must be positive.")
		}
		sp.sinkhornEpsilon = value
		return nil
	case superpixel.SinkhornIterations:
		if value < 1 {
			return errors.New("Sinkhorn iterations must be at least 1.")
		}
		sp.sinkhornIterations = int(value)
		return nil
	default:
		return errors.New("Wasserstein SP does not support this parameter.")
	}
}

func (sp *SpWasserstein) SetImage(img image.Image) {
	sp.Img = img
	sp.Result = superpixel.SpResult{}
	sp.labCache = nil
}

func (sp *SpWasserstein) buildLabCache() {
	width := sp.Img.Bounds().Dx()
	height := sp.Img.Bounds().Dy()
	sp.labCache = make([][3]float64, width*height)
	for y := range height {
		for x := range width {
			c, _ := colorful.MakeColor(sp.Img.At(x, y))
			l, a, b := c.Lab()
			l *= 100
			a *= 100
			b *= 100
			sp.labCache[y*width+x] = [3]float64{l, a, b}
		}
	}
}

func computeRepresentativeColors(labCache [][3]float64, k int) [][3]float64 {
	if k < 1 {
		k = 1
	}
	total := len(labCache)
	maxSamples := 10000
	step := 1
	if total > maxSamples {
		step = int(math.Ceil(float64(total) / float64(maxSamples)))
	}
	samples := make([][3]float64, 0, total/step+1)
	for i := 0; i < total; i += step {
		samples = append(samples, labCache[i])
	}
	if len(samples) == 0 {
		return make([][3]float64, k)
	}

	centers := make([][3]float64, k)
	for i := 0; i < k; i++ {
		centers[i] = samples[(i*len(samples))/k]
	}

	maxIter := 10
	for iter := 0; iter < maxIter; iter++ {
		sums := make([][3]float64, k)
		counts := make([]int, k)
		for _, s := range samples {
			nearest := 0
			minDist := math.Inf(1)
			for c := 0; c < k; c++ {
				dl := s[0] - centers[c][0]
				da := s[1] - centers[c][1]
				db := s[2] - centers[c][2]
				d := dl*dl + da*da + db*db
				if d < minDist {
					minDist = d
					nearest = c
				}
			}
			sums[nearest][0] += s[0]
			sums[nearest][1] += s[1]
			sums[nearest][2] += s[2]
			counts[nearest]++
		}

		shift := 0.0
		for c := 0; c < k; c++ {
			if counts[c] == 0 {
				centers[c] = samples[(c+iter)%len(samples)]
				continue
			}
			newCenter := [3]float64{
				sums[c][0] / float64(counts[c]),
				sums[c][1] / float64(counts[c]),
				sums[c][2] / float64(counts[c]),
			}
			dl := newCenter[0] - centers[c][0]
			da := newCenter[1] - centers[c][1]
			db := newCenter[2] - centers[c][2]
			shift += dl*dl + da*da + db*db
			centers[c] = newCenter
		}
		if shift < 1e-4 {
			break
		}
	}

	return centers
}

func (sp *SpWasserstein) initialSlicPartition() (*superpixel.SpResult, error) {
	impl := &slic.SpSlic{}
	impl.SetImage(sp.Img)
	impl.SetParameter(superpixel.Count, float64(sp.initialSuperpixels))
	impl.SetParameter(superpixel.Compactness, sp.compactness)
	impl.SetParameter(superpixel.Iterations, float64(sp.slicIterations))
	impl.SetParameter(superpixel.InitialPerturbationNeighborhoodRadius, float64(sp.initialPerturbationNeighborhoodRadius))
	if err := impl.Run(); err != nil {
		return nil, err
	}
	return impl.GetResult(), nil
}

func (sp *SpWasserstein) Run() error {
	if sp.Img == nil {
		return errors.New("Image not set.")
	}
	if sp.targetRegions < 1 {
		return errors.New("Target region count must be at least 1.")
	}

	// Initial superpixels via SLIC
	initialResult, err := sp.initialSlicPartition()
	if err != nil {
		return err
	}

	width := sp.Img.Bounds().Dx()
	height := sp.Img.Bounds().Dy()
	total := width * height

	labelMapInt := make([]int, total)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			labelMapInt[y*width+x] = int(initialResult.LabelMap.Gray16At(x, y).Y)
		}
	}
	m := initialResult.SpCount
	if sp.targetRegions > m {
		return errors.New("Target region count can not exceed initial superpixel count.")
	}

	// LAB cache
	sp.buildLabCache()

	// Representative colors for histograms
	centers := computeRepresentativeColors(sp.labCache, sp.histogramBins)
	ot := newOtSolver(centers, sp.sinkhornEpsilon, sp.sinkhornIterations)

	// Bin index for each pixel
	binIndex := make([]int, total)
	for idx, lab := range sp.labCache {
		minIdx := 0
		minDist := math.Inf(1)
		for c := 0; c < len(centers); c++ {
			dl := lab[0] - centers[c][0]
			da := lab[1] - centers[c][1]
			db := lab[2] - centers[c][2]
			d := dl*dl + da*da + db*db
			if d < minDist {
				minDist = d
				minIdx = c
			}
		}
		binIndex[idx] = minIdx
	}

	// Region histograms
	regions := make([]region, m)
	for i := 0; i < m; i++ {
		regions[i] = region{
			active:    true,
			hist:      make([]float64, sp.histogramBins),
			neighbors: make(map[int]struct{}),
		}
	}

	for idx := 0; idx < total; idx++ {
		lbl := labelMapInt[idx]
		regions[lbl].pixelCount++
		regions[lbl].hist[binIndex[idx]]++
	}
	for i := 0; i < m; i++ {
		if regions[i].pixelCount == 0 {
			continue
		}
		count := float64(regions[i].pixelCount)
		for b := 0; b < sp.histogramBins; b++ {
			regions[i].hist[b] /= count
		}
	}

	// Build adjacency
	addEdge := func(a, b int) {
		if a == b {
			return
		}
		regions[a].neighbors[b] = struct{}{}
		regions[b].neighbors[a] = struct{}{}
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			lbl := labelMapInt[idx]
			if x+1 < width {
				right := labelMapInt[idx+1]
				if right != lbl {
					addEdge(lbl, right)
				}
			}
			if y+1 < height {
				down := labelMapInt[idx+width]
				if down != lbl {
					addEdge(lbl, down)
				}
			}
		}
	}

	// Merge loop ---------------------------------------------------------
	pq := &mergeQueue{}
	heap.Init(pq)

	for i := 0; i < m; i++ {
		for j := range regions[i].neighbors {
			if j <= i {
				continue
			}
			cost := ot.distance(regions[i].hist, regions[j].hist)
			kappa := cost - regions[i].heterogeneity - regions[j].heterogeneity
			heap.Push(pq, &mergeCandidate{i: i, j: j, kappa: kappa, cost: cost})
		}
	}

	dsu := newDsu(m)
	activeCount := m

	for activeCount > sp.targetRegions && pq.Len() > 0 {
		item := heap.Pop(pq).(*mergeCandidate)
		i := item.i
		j := item.j
		if !regions[i].active || !regions[j].active {
			continue
		}
		if _, ok := regions[i].neighbors[j]; !ok {
			continue
		}
		currentCost := ot.distance(regions[i].hist, regions[j].hist)
		currentKappa := currentCost - regions[i].heterogeneity - regions[j].heterogeneity
		if currentKappa > item.kappa+1e-9 {
			heap.Push(pq, &mergeCandidate{i: i, j: j, kappa: currentKappa, cost: currentCost})
			continue
		}

		// Merge j into i
		regions[i].heterogeneity = currentCost
		countI := regions[i].pixelCount
		countJ := regions[j].pixelCount
		regions[i].pixelCount = countI + countJ
		if regions[i].pixelCount > 0 {
			fCountI := float64(countI)
			fCountJ := float64(countJ)
			for b := 0; b < sp.histogramBins; b++ {
				regions[i].hist[b] = (regions[i].hist[b]*fCountI + regions[j].hist[b]*fCountJ) / (fCountI + fCountJ)
			}
		}
		regions[j].active = false
		dsu.union(i, j)
		activeCount--

		delete(regions[i].neighbors, j)
		for k := range regions[j].neighbors {
			if !regions[k].active || k == i {
				delete(regions[k].neighbors, j)
				continue
			}
			delete(regions[k].neighbors, j)
			regions[k].neighbors[i] = struct{}{}
			regions[i].neighbors[k] = struct{}{}
		}
		regions[j].neighbors = nil

		for k := range regions[i].neighbors {
			if !regions[k].active {
				delete(regions[i].neighbors, k)
				continue
			}
			cost := ot.distance(regions[i].hist, regions[k].hist)
			kappa := cost - regions[i].heterogeneity - regions[k].heterogeneity
			heap.Push(pq, &mergeCandidate{i: i, j: k, kappa: kappa, cost: cost})
		}
	}

	// Build final label map ----------------------------------------------
	labelMap := image.NewGray16(image.Rect(0, 0, width, height))
	rootToLabel := make(map[int]int)
	nextLabel := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			root := dsu.find(labelMapInt[idx])
			label, ok := rootToLabel[root]
			if !ok {
				label = nextLabel
				rootToLabel[root] = label
				nextLabel++
			}
			labelMap.SetGray16(x, y, color.Gray16{Y: uint16(label)})
		}
	}

	sp.Result = superpixel.SpResult{LabelMap: labelMap, SpCount: nextLabel}
	return nil
}
