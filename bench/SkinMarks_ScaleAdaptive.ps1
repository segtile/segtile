go run ./cmd/bench `
    "S:\MasterThesisData\Datasets\SkinMarks\IMAGES\" `
    "S:\MasterThesisData\Datasets\SkinMarks\MASKS\" `
    "S:\MasterThesisData\BenchmarkResults\ScaleAdaptive\SkinMarks.csv" `
    scale_adaptive `
    "threshold=5,100,5" `
    "beta=0,1,0.05"
