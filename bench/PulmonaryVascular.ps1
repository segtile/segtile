go run ./cmd/bench `
    "S:\MasterThesisData\Datasets\PulmonaryVascular\IMAGES\" `
    "S:\MasterThesisData\Datasets\PulmonaryVascular\MASKS\" `
    "S:\MasterThesisData\BenchmarkResults\SLIC\PulmonaryVascular.csv" `
    slic `
    "count=20,1000,20" `
    "compactness=0,50,5"
