go run ./cmd/bench `
    "S:\MasterThesisData\Datasets\SkinMarks\IMAGES\" `
    "S:\MasterThesisData\Datasets\SkinMarks\MASKS\" `
    "S:\MasterThesisData\BenchmarkResults\SLIC\SkinMarks.csv" `
    slic `
    "count=20,1000,20" `
    "compactness=0,150,5"
