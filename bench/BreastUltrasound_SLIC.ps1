go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\Breast Ultrasound\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\Breast Ultrasound\MASKS\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\BreastUltrasound.csv" `
    slic `
    "count=100,1500,50" `
    "compactness=0,100,5"
