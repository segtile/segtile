go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\CryoNuSeg\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\CryoNuSeg\MASK1a\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\RootedSpanning\NucleiSegmentation.csv" `
    rss `
    "count=250,15000,250" `
    "compactness=0,5,0.25"