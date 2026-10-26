$IMG = "\\lennart\Sync\MasterThesisData\EvalDatasets\SkinMarks\IMG\"
$MASK = "\\lennart\Sync\MasterThesisData\EvalDatasets\SkinMarks\MASK\"
$CSV =  "SkinMarks.csv"

<#
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
#>
go run ./cmd/bench `
    "$IMG" `
    "$MASK" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\ScaleAdaptive\$CSV" `
    scale_adaptive `
    "threshold=10,250,10" `
    "beta=0.5,1,0.01"

