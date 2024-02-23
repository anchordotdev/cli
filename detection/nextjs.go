package detection

import (
	"github.com/anchordotdev/cli/anchorcli"
	"github.com/anchordotdev/cli/detection/package_managers"
)

var NextJS = &NextJSDetector{}

type NextJSDetector struct {
	FollowUpDetectors []Detector
}

func (d NextJSDetector) GetTitle() string { return "Next.js" }

func (d NextJSDetector) FollowUp() []Detector {
	return d.FollowUpDetectors
}

func (d NextJSDetector) Detect(dirFS FS) (Match, error) {
	_, err := dirFS.Stat("package.json")
	if err != nil {
		return Match{}, err
	}

	packageData, err := dirFS.ReadFile("package.json")
	if err != nil {
		return Match{}, err
	}

	packages, err := package_managers.ParsePackageJSON(packageData)
	if err != nil {
		return Match{}, err
	}

	if packages.HasDependency("next") {
		return Match{
			Detector:       d,
			Detected:       true,
			Confidence:     High,
			AnchorCategory: anchorcli.CategoryJavascript,
		}, nil
	}
	return Match{}, nil
}
