go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\Lung Segmentation\IMAGES_SMALL\" `
    "\\lennart\Sync\MasterThesisData\Datasets\Lung Segmentation\MASKS_SMALL\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\RootedSpanning\LungSegmentation.csv" `
    rss `
    "count=25,1000,25" `
    "compactness=0,50,5"
