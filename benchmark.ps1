$ImageDirectory = 'S:\Google-Drive\Studium\Master\Masterarbeit\Datasets\Brain MRI\IMAGES\'
$MaskDirectory = 'S:\Google-Drive\Studium\Master\Masterarbeit\Datasets\Brain MRI\MASKS\'

Get-ChildItem -Path $ImageDirectory | ForEach-Object {
	$ImagePath = $_.FullName
	$MaskPath = Join-Path -Path $MaskDirectory -ChildPath ($_.BaseName + "_mask.tif")
	"Processing $ImagePath with mask $MaskPath"
	go run ./cmd/bench $ImagePath $MaskPath | Out-File -Encoding UTF8 -Append -FilePath benchmark.csv
}