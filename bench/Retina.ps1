go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\Retina\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\Retina\MASKS\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\Retina.csv" `
    slic `
    "count=200,10000,200" `
    "compactness=0,50,5"
