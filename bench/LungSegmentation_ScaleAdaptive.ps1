go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\Lung Segmentation\IMAGES_SMALL\" `
    "\\lennart\Sync\MasterThesisData\Datasets\Lung Segmentation\MASKS_SMALL\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\ScaleAdaptive\LungSegmentation.csv" `
    scale_adaptive `
    "threshold=5,100,5" `
    "beta=0,1,0.05"
