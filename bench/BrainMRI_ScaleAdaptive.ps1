go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\Brain MRI\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\Brain MRI\MASKS\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\ScaleAdaptive\BrainMRI.csv" `
    scale_adaptive `
    "threshold=5,100,5" `
    "beta=0,1,0.05"
