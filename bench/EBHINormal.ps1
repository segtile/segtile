go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\EBHI\Normal\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\EBHI\Normal\MASKS\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\EBHINormal.csv" `
    slic `
    "count=50,2000,50" `
    "compactness=0,200,20"
