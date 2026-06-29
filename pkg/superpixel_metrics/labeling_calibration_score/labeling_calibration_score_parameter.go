package labeling_calibration_score

import "fmt"

type LabelingCalibrationScoreParameter string

const (
	MaxIslands LabelingCalibrationScoreParameter = "max_islands"
)

func (this LabelingCalibrationScoreParameter) String() string {
	return string(this)
}

// Titles
func (this LabelingCalibrationScoreParameter) Title() string {
	switch this {
	case MaxIslands:
		return "Max Islands"
	default:
		return fmt.Sprintf("Unnamed Labeling Calibration Score Parameter (%s)", this)
	}
}

// Descriptions
func (this LabelingCalibrationScoreParameter) Description() string {
	switch this {
	case MaxIslands:
		return "The maximum number of islands to consider for evaluation."
	default:
		return fmt.Sprintf("Parameter without description. (%s)", this)
	}
}
