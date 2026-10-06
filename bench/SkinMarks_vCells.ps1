go run ./cmd/bench `
    "S:\MasterThesisData\Datasets\SkinMarks\IMAGES\" `
    "S:\MasterThesisData\Datasets\SkinMarks\MASKS\" `
    "S:\MasterThesisData\BenchmarkResults\vCells\SkinMarks.csv" `
    vcell `
    "count=20,1000,20" `
    "compactness=0,40,2"
