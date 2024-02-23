package detection

import (
	"encoding/json"
	"io/fs"
	"reflect"
	"slices"
	"testing"
	"testing/fstest"
	"time"

	"github.com/anchordotdev/cli/anchorcli"
)

func TestDetectorsDetect(t *testing.T) {
	tests := []struct {
		name string

		detector Detector
		fs       FS

		match Match
		err   error
	}{
		{
			name: "rails-happy-path",

			detector: Rails,
			fs: fstest.MapFS{
				"Gemfile":   emptyFile,
				"Rakefile":  emptyFile,
				"config.ru": emptyFile,
				"app":       emptyDir,
				"config":    emptyDir,
				"db":        emptyDir,
				"lib":       emptyDir,
				"public":    emptyDir,
				"vendor":    emptyDir,
			},

			match: Match{
				Detector:       Rails,
				Detected:       true,
				Confidence:     High,
				AnchorCategory: anchorcli.CategoryRuby,
			},
		},

		{
			name: "sinatra-happy-path",

			detector: Sinatra,
			fs: fstest.MapFS{
				"Gemfile":   emptyFile,
				"config.ru": emptyFile,
				"app.rb":    emptyFile,
			},

			match: Match{
				Detector:       Sinatra,
				Detected:       true,
				Confidence:     High,
				AnchorCategory: anchorcli.CategoryRuby,
			},
		},

		{
			name: "django-happy-path",

			detector: Django,
			fs: fstest.MapFS{
				"requirements.txt": emptyFile,
				"manage.py":        emptyFile,
			},

			match: Match{
				Detector:       Django,
				Detected:       true,
				Confidence:     High,
				AnchorCategory: anchorcli.CategoryPython,
			},
		},

		{
			name: "flask-happy-path",

			detector: Flask,
			fs: fstest.MapFS{
				"requirements.txt": emptyFile,
				"app.py":           emptyFile,
			},

			match: Match{
				Detector:       Flask,
				Detected:       true,
				Confidence:     High,
				AnchorCategory: anchorcli.CategoryPython,
			},
		},

		{
			name: "nextjs-happy-path",

			detector: NextJS,
			fs: fstest.MapFS{
				"package.json": jsonFragment{
					"dependencies": map[string]any{
						"next": "latest",
					},
				}.mapFile(0644),
				"pages": emptyDir,
			},

			match: Match{
				Detector:       NextJS,
				Detected:       true,
				Confidence:     High,
				AnchorCategory: anchorcli.CategoryJavascript,
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			match, err := test.detector.Detect(test.fs)
			if err != nil {
				if want, got := test.err, err; want != got {
					t.Fatalf("%s: want error %q, but got %q", test.detector.GetTitle(), want, got)
				}
			}

			if want, got := test.match.Detected, match.Detected; want != got {
				t.Errorf("%s: want match detection %t, got %t", test.detector.GetTitle(), want, got)
			}
			if want, got := test.match.Confidence, match.Confidence; want != got {
				t.Errorf("%s: want match confidence score %s, got %s", test.detector.GetTitle(), want, got)
			}
			if want, got := len(test.match.FollowUpDetectors), len(match.FollowUpDetectors); want != got {
				t.Errorf("%s: want %d follow-up detectors, got %d", test.detector.GetTitle(), want, got)
			}
			if want, got := test.match.FollowUpDetectors, match.FollowUpDetectors; !reflect.DeepEqual(want, got) {
				t.Errorf("%s: want %+v follow-up detectors, got %+v", test.detector.GetTitle(), want, got)
			}
			if want, got := test.match.AnchorCategory, match.AnchorCategory; want != got {
				t.Errorf("%s: want AnchorCategory %s, got %s", test.detector.GetTitle(), want, got)
			}
			if want, got := test.match.MissingRequiredFiles, match.MissingRequiredFiles; slices.Compare(want, got) != 0 {
				t.Errorf("%s: want missing required files %+v, got %+v", test.detector.GetTitle(), want, got)
			}
		})
	}
}

var (
	emptyFile = textFile("", 0644)

	emptyDir = &fstest.MapFile{
		Mode:    0755 | fs.ModeDir,
		ModTime: mtime,
	}

	mtime = time.Now()
)

func textFile(data string, mode fs.FileMode) *fstest.MapFile {
	return &fstest.MapFile{
		Data:    []byte(data),
		Mode:    mode,
		ModTime: mtime,
	}
}

func jsonFile(v map[string]any, mode fs.FileMode) *fstest.MapFile {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return textFile(string(data), mode)
}

type jsonFragment map[string]any

func (f jsonFragment) mapFile(mode fs.FileMode) *fstest.MapFile {
	return jsonFile((map[string]any)(f), mode)
}
