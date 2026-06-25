package regularity_score

import "fmt"

type RegularityScoreParameter string

const (
	MaxIslands RegularityScoreParameter = "max_islands"
)

func (this RegularityScoreParameter) String() string {
	return string(this)
}

// Titles
func (this RegularityScoreParameter) Title() string {
	switch this {
	case MaxIslands:
		return "Max Islands"
	default:
		return fmt.Sprintf("Unnamed Regularity Score Parameter (%s)", this)
	}
}

// Descriptions
func (this RegularityScoreParameter) Description() string {
	switch this {
	case MaxIslands:
		return "The maximum number of islands to consider for evaluation."
	default:
		return fmt.Sprintf("Parameter without description. (%s)", this)
	}
}
