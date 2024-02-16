package detection

import (
	"github.com/anchordotdev/cli/anchorcli"
)

// RubyDetector is a detector for Ruby projects.
var RubyDetector = &FileDetector{
	Title:             "ruby",
	Paths:             []string{"Gemfile", "Gemfile.lock", "Rakefile"},
	FollowUpDetectors: []Detector{RailsDetector, SinatraDetector},
	AnchorCategory:    anchorcli.CategoryRuby,
}

// GoDetector is a Go detector
var GoDetector = &FileDetector{
	Title:             "go",
	Paths:             []string{"main.go", "go.mod", "go.sum"},
	FollowUpDetectors: nil,
	AnchorCategory:    anchorcli.CategoryGo,
}

// JavascriptDetector is a JavaScript detector
var JavascriptDetector = &FileDetector{
	Title:             "javascript",
	Paths:             []string{"package.json", "index.js", "app.js"},
	FollowUpDetectors: nil,
	AnchorCategory:    anchorcli.CategoryJavascript,
}

// PythonDetector is a Python detector with Django and Flask follow-up detectors
var PythonDetector = &FileDetector{
	Title:             "python",
	Paths:             []string{"requirements.txt"},
	FollowUpDetectors: []Detector{DjangoDetector, FlaskDetector},
	AnchorCategory:    anchorcli.CategoryPython,
}
