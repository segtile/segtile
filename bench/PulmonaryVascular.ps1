go run ./cmd/bench `
    "S:\MasterThesisData\Datasets\PulmonaryVascular\IMAGES\" `
    "S:\MasterThesisData\Datasets\PulmonaryVascular\MASKS\" `
    "S:\MasterThesisData\BenchmarkResults\SLIC\PulmonaryVascular.csv" `
    slic `
    "count=100,5000,100" `
    "compactness=0,100,10"
