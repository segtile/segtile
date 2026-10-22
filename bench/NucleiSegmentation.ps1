$IMG = "\\lennart\Sync\MasterThesisData\EvalDatasets\NucleiSegmentation\IMG\"
$MASK = "\\lennart\Sync\MasterThesisData\EvalDatasets\NucleiSegmentation\MASK\"
$CSV =  "NucleiSegmentation.csv"


go run ./cmd/bench `
    "$IMG" `
    "$MASK" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\$CSV" `
    slic `
    "count=250,10000,250" `
    "compactness=0,100,5"


go run ./cmd/bench `
    "$IMG" `
    "$MASK" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\RootedSpanning\$CSV" `
    rss `
    "count=250,10000,250" `
    "compactness=0,50,2.5"


go run ./cmd/bench `
    "$IMG" `
    "$MASK" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\vCells\$CSV" `
    vcell `
    "count=250,10000,250" `
    "compactness=0,25,1.25"

go run ./cmd/bench `
    "$IMG" `
    "$MASK" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\ScaleAdaptive\$CSV" `
    scale_adaptive `
    "threshold=5,100,5" `
    "beta=0,1,0.05"

