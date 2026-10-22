$IMG = "\\lennart\Sync\MasterThesisData\EvalDatasets\BrainMRI\IMG\"
$MASK = "\\lennart\Sync\MasterThesisData\EvalDatasets\BrainMRI\MASK\"


go run ./cmd/bench `
    "$IMG" `
    "$MASK" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\BrainMRI.csv" `
    slic `
    "count=50,1000,50" `
    "compactness=0,100,5"


go run ./cmd/bench `
    "$IMG" `
    "$MASK" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\RootedSpanning\BrainMRI.csv" `
    rss `
    "count=50,1000,50" `
    "compactness=0,50,2.5"


go run ./cmd/bench `
    "$IMG" `
    "$MASK" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\vCells\BrainMRI.csv" `
    vcell `
    "count=50,1000,50" `
    "compactness=0,25,1.25"

go run ./cmd/bench `
    "$IMG" `
    "$MASK" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\ScaleAdaptive\BrainMRI.csv" `
    scale_adaptive `
    "threshold=5,100,5" `
    "beta=0,1,0.05"

