go run ./cmd/bench `
    "S:\MasterThesisData\Datasets\Breast Ultrasound\IMAGES\" `
    "S:\MasterThesisData\Datasets\Breast Ultrasound\MASKS\" `
    "S:\MasterThesisData\BenchmarkResults\ScaleAdaptive\BreastUltrasound.csv" `
    scale_adaptive `
    "threshold=5,100,5" `
    "beta=0,1,0.05"
