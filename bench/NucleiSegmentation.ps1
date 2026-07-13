go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\CryoNuSeg\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\CryoNuSeg\MASK1a\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\NucleiSegmentation.csv" `
    slic `
    "count=100,10000,100" `
    "compactness=0,160,10"
