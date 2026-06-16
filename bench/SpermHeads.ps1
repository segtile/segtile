go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\SpermHeads\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\SpermHeads\MASKS\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\SpermHeads.csv" `
    slic `
    "count=20,1000,20" `
    "compactness=0,50,5"
