go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\Lung Segmentation\IMAGES_SMALL\" `
    "\\lennart\Sync\MasterThesisData\Datasets\Lung Segmentation\MASKS_SMALL\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\LungSegmentation.csv" `
    slic `
    "count=50,1000,50" `
    "compactness=0,100,5"
