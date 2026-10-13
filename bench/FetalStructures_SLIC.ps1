go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\Fetal Structures\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\Fetal Structures\MASKS\liver\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\FetalStructures.csv" `
    slic `
    "count=50,2500,50" `
    "compactness=0,70,5"
