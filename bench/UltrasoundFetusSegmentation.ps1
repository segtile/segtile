go run ./cmd/bench `
    "\\lennart\Sync\MasterThesisData\Datasets\Fetal Abdominal Structures Segmentation Dataset Using Ultrasonic Images\IMAGES\" `
    "\\lennart\Sync\MasterThesisData\Datasets\Fetal Abdominal Structures Segmentation Dataset Using Ultrasonic Images\MASKS\" `
    "\\lennart\Sync\MasterThesisData\BenchmarkResults\SLIC\UltrasoundFetusSegmentation.csv" `
    slic `
    "count=20,1000,20" `
    "compactness=0,50,5"
