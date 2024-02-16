package detection

import (
	"github.com/anchordotdev/cli/anchorcli"
)

// Ruby Frameworks
var RailsFiles []string = []string{"Gemfile", "Rakefile", "config.ru", "app", "config", "db", "lib", "public", "vendor"}
var SinatraFiles []string = []string{"Gemfile", "config.ru", "app.rb"}

// Python Frameworks
var DjangoFiles []string = []string{"requirements.txt", "manage.py"}
var FlaskFiles []string = []string{"requirements.txt", "app.py"}

var RailsDetector = &FileDetector{
	Title:             "rails",
	Paths:             RailsFiles,
	FollowUpDetectors: nil,
	AnchorCategory:    anchorcli.CategoryRuby,
}

var SinatraDetector = &FileDetector{
	Title:             "sinatra",
	Paths:             SinatraFiles,
	FollowUpDetectors: nil,
	AnchorCategory:    anchorcli.CategoryRuby,
	RequiredFiles:     []string{"app.rb"},
}

var DjangoDetector = &FileDetector{
	Title:             "django",
	Paths:             DjangoFiles,
	FollowUpDetectors: nil,
	AnchorCategory:    anchorcli.CategoryPython,
}
var FlaskDetector = &FileDetector{
	Title:             "flask",
	Paths:             FlaskFiles,
	FollowUpDetectors: nil,
	AnchorCategory:    anchorcli.CategoryPython,
}
