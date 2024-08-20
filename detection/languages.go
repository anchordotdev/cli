package detection

import (
	"github.com/anchordotdev/cli/anchorcli"
)

var (
	Custom = &FileDetector{
		Title:             "Custom",
		Paths:             []string{},
		FollowUpDetectors: nil,
		AnchorCategory:    anchorcli.CategoryCustom,
	}

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
		Paths:             []string{"Pipfile", "Pipfile.lock", "requirements.txt"},
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
