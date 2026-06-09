go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\CryoNuSeg\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\CryoNuSeg\MASK1a\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\NucleiSegmentation.csv" `
    slic `
    "count=250,20000,250" `
    "compactness=0,50,10"
