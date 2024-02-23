package detection

import (
	"github.com/anchordotdev/cli/anchorcli"
)

var (

	// Python Frameworks

	Django = &FileDetector{
		Title:             "Django",
		Paths:             []string{"manage.py"},
		FollowUpDetectors: nil,
		AnchorCategory:    anchorcli.CategoryPython,
	}
	Flask = &FileDetector{
		Title:             "Flask",
		Paths:             []string{"app.py"},
		FollowUpDetectors: nil,
		AnchorCategory:    anchorcli.CategoryPython,
	}

	// Ruby Frameworks

	Rails = &FileDetector{
		Title: "Ruby on Rails",
		Paths: []string{
			"config.ru", "app", "config", "db", "lib", "public", "vendor",
		},
		FollowUpDetectors: nil,
		AnchorCategory:    anchorcli.CategoryRuby,
	}
	Sinatra = &FileDetector{
		Title: "Sinatra",
		Paths: []string{
			"app.rb",
		},
		FollowUpDetectors: nil,
		AnchorCategory:    anchorcli.CategoryRuby,
		RequiredFiles:     []string{"app.rb"},
	}
)
