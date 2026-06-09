go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\Brain MRI\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\Brain MRI\MASKS\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\BrainMRI.csv" `
    slic `
    "count=20,1000,20" `
    "compactness=0,50,5"
