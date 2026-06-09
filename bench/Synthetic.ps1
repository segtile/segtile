go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\Synthetic\IMAGES_NOISE_A\" `
    "\\lennart\Sync\MasterThesisData\Datasets\Synthetic\MASKS\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\Synthetic.csv" `
    slic `
    "count=20,1000,20" `
    "compactness=0,50,5"
