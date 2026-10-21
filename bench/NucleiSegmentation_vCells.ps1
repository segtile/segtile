go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\CryoNuSeg\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\CryoNuSeg\MASK1a\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\vCells\NucleiSegmentation.csv" `
    vcell `
    "count=250,15000,250" `
    "compactness=0,20,2"
