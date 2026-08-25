go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\Breast Ultrasound\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\Breast Ultrasound\MASKS\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\vCells\BreastUltrasound.csv" `
    vcell `
    "count=25,1000,25" `
    "compactness=0.5,20,0.5"
