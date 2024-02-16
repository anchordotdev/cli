package detection

import (
	"github.com/anchordotdev/cli/anchorcli"
)

// Confidence represents the confidence score
type Confidence int

// Different confidence levels
const (
	High   Confidence = 100
	Medium Confidence = 60
	Low    Confidence = 40
	None   Confidence = 0
)

// Confidence.String() returns the string representation of the confidence level
func (s Confidence) String() string {
	switch s {
	case High:
		return "High"
	case Medium:
		return "Medium"
	case Low:
		return "Low"
	case None:
		return "None"
	default:
		return "Unknown"
	}
}

var (
	DefaultDetectors = []Detector{
		RubyDetector,
		GoDetector,
		JavascriptDetector,
		PythonDetector,
	}

	DetectorsByFlag = map[string]Detector{
		"django":     DjangoDetector,
		"flask":      FlaskDetector,
		"go":         GoDetector,
		"javascript": JavascriptDetector,
		"python":     PythonDetector,
		"rails":      RailsDetector,
		"ruby":       RubyDetector,
		"sinatra":    SinatraDetector,
	}

	PositiveDetectionMessage = "%s project detected with confidence level %s!\n"
)

// Match holds the detection result, confidence, and follow-up detectors
type Match struct {
	Detector   Detector
	Detected   bool
	Confidence Confidence
	// MissingRequiredFiles represents a list of files that are required but missing.
	MissingRequiredFiles []string
	FollowUpDetectors    []Detector
	Details              string
	AnchorCategory       *anchorcli.Category
}

// Detector interface represents a project detector
type Detector interface {
	GetTitle() string
	Detect(directory string) (Match, error)
	FollowUp() []Detector
}

func Perform(detectors []Detector, dir string) (Results, error) {
	res := make(Results)
	for _, detector := range detectors {
		match, err := detector.Detect(dir)
		if err != nil {
			return nil, err
		}

		if !match.Detected {
			continue
		}

		res[match.Confidence] = append(res[match.Confidence], match)

		if followupResults, err := Perform(match.FollowUpDetectors, dir); err == nil {
			res.merge(followupResults)
		} else {
			return nil, err
		}
	}
	return res, nil
}

type Results map[Confidence][]Match

func (r Results) merge(other Results) {
	for confidence, matches := range other {
		for _, match := range matches {
			if !match.Detected {
				continue
			}
			// Merge the results, putting the new matches at the front of the list
			r[confidence] = append([]Match{match}, r[confidence]...)
		}
	}
}
