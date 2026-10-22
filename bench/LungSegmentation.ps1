$IMG = "\\lennart\Sync\MasterThesisData\EvalDatasets\LungSegmentation\IMG\"
$MASK = "\\lennart\Sync\MasterThesisData\EvalDatasets\LungSegmentation\MASK\"
$CSV =  "LungSegmentation.csv"


go run ./cmd/bench `
    "$IMG" `
    "$MASK" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\$CSV" `
    slic `
    "count=50,1000,50" `
    "compactness=0,100,5"


go run ./cmd/bench `
    "$IMG" `
    "$MASK" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\RootedSpanning\$CSV" `
    rss `
    "count=50,1000,50" `
    "compactness=0,50,2.5"


go run ./cmd/bench `
    "$IMG" `
    "$MASK" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\vCells\$CSV" `
    vcell `
    "count=50,1000,50" `
    "compactness=0,25,1.25"

go run ./cmd/bench `
    "$IMG" `
    "$MASK" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\ScaleAdaptive\$CSV" `
    scale_adaptive `
    "threshold=5,100,5" `
    "beta=0,1,0.05"

