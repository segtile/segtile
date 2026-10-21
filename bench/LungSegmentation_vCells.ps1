go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\Lung Segmentation\IMAGES_SMALL\" `
    "\\lennart\Sync\MasterThesisData\Datasets\Lung Segmentation\MASKS_SMALL\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\vCells\LungSegmentation.csv" `
    vcell `
    "count=50,1000,50" `
    "compactness=1,30,1"
