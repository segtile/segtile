go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\Brain MRI\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\Brain MRI\MASKS\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\vCells\BrainMRI.csv" `
    vcell `
    "count=20,1000,20" `
    "compactness=1,30,1"
