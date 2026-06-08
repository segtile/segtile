go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\Breast Ultrasound\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\Breast Ultrasound\MASKS\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\BreastUltrasound.csv" `
    slic `
    "count=20,1000,20" `
    "compactness=0,150,10"
