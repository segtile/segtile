package density_score

import "fmt"

type DensityScoreParameter string

const (
	MaxIslands DensityScoreParameter = "max_islands"
)

func (this DensityScoreParameter) String() string {
	return string(this)
}

// Titles
func (this DensityScoreParameter) Title() string {
	switch this {
	case MaxIslands:
		return "Max Islands"
	default:
		return fmt.Sprintf("Unnamed Density Score Parameter (%s)", this)
	}
}

// Descriptions
func (this DensityScoreParameter) Description() string {
	switch this {
	case MaxIslands:
		return "The maximum number of islands to consider for evaluation."
	default:
		return fmt.Sprintf("Parameter without description. (%s)", this)
	}
}
