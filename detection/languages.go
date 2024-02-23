package detection

import (
	"github.com/anchordotdev/cli/anchorcli"
)

var (
	Go = &FileDetector{
		Title:             "Go",
		Paths:             []string{"main.go", "go.mod", "go.sum"},
		FollowUpDetectors: nil,
		AnchorCategory:    anchorcli.CategoryGo,
	}

	Javascript = &FileDetector{
		Title:             "JavaScript",
		Paths:             []string{"package.json", "index.js", "app.js"},
		FollowUpDetectors: []Detector{NextJS},
		AnchorCategory:    anchorcli.CategoryJavascript,
	}

	Python = &FileDetector{
		Title:             "Python",
		Paths:             []string{"requirements.txt"},
		FollowUpDetectors: []Detector{Django, Flask},
		AnchorCategory:    anchorcli.CategoryPython,
	}

	Ruby = &FileDetector{
		Title:             "Ruby",
		Paths:             []string{"Gemfile", "Gemfile.lock", "Rakefile"},
		FollowUpDetectors: []Detector{Rails, Sinatra},
		AnchorCategory:    anchorcli.CategoryRuby,
	}
)
