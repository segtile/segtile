package f1_match

import "fmt"

type F1MatchParameter string

const (
	DesiredF1Score F1MatchParameter = "desired_f1_score"
)

func (this F1MatchParameter) String() string {
	return string(this)
}

// Titles
func (this F1MatchParameter) Title() string {
	switch this {
	case DesiredF1Score:
		return "Desired F1 Score"
	default:
		return fmt.Sprintf("Unnamed F1 Match Parameter (%s)", this)
	}
}

// Descriptions
func (this F1MatchParameter) Description() string {
	switch this {
	case DesiredF1Score:
		return "The desired F1 score."
	default:
		return fmt.Sprintf("Parameter without description. (%s)", this)
	}
}
