go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\Retina\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\Retina\MASKS\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\Retina.csv" `
    slic `
    "count=20,1000,20" `
    "compactness=0,50,5"
