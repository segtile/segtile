go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\CryoNuSeg\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\CryoNuSeg\MASK1a\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\ScaleAdaptive\NucleiSegmentation.csv" `
    scale_adaptive `
    "threshold=5,100,5" `
    "beta=0,1,0.05"
