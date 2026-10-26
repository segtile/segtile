$IMG = "\\lennart\Sync\MasterThesisData\EvalDatasets\BreastUltrasound\IMG\"
$MASK = "\\lennart\Sync\MasterThesisData\EvalDatasets\BreastUltrasound\MASK\"
$CSV =  "BreastUltrasound.csv"

<#
go run ./cmd/bench `
    "$IMG" `
    "$MASK" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\$CSV" `
    slic `
    "count=25,750,25" `
    "compactness=0,100,5"


go run ./cmd/bench `
    "$IMG" `
    "$MASK" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\RootedSpanning\$CSV" `
    rss `
    "count=25,750,25" `
    "compactness=0,100,5"


go run ./cmd/bench `
    "$IMG" `
    "$MASK" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\vCells\$CSV" `
    vcell `
    "count=25,750,25" `
    "compactness=0,25,1.25"
#>
go run ./cmd/bench `
    "$IMG" `
    "$MASK" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\ScaleAdaptive\$CSV" `
    scale_adaptive `
    "threshold=10,100,10" `
    "beta=0.0,1,0.01"

