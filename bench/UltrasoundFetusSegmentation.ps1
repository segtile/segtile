go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\Fetal Structures\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\Fetal Structures\MASKS\liver" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\UltrasoundFetusSegmentation.csv" `
    slic `
    "count=20,1000,20" `
    "compactness=0,50,5"
