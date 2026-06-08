go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\Lung Segmentation\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\Lung Segmentation\MASKS\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\LungSegmentation.csv" `
    slic `
    "count=50,2000,50" `
    "compactness=0,50,5"
