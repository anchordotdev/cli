package detection

import (
	"flag"
	"slices"
	"testing"
	"testing/fstest"
)

var (
	_ = flag.Bool("prism-verbose", false, "ignored")
	_ = flag.Bool("prism-proxy", false, "ignored")
	_ = flag.Bool("update", false, "ignored")
)

func TestScore_String(t *testing.T) {
	testCases := []struct {
		confidence     Confidence
		expectedString string
	}{
		{High, "High"},
		{Medium, "Medium"},
		{Low, "Low"},
		{None, "None"},
		{Confidence(42), "Unknown"}, // Unknown confidence scores
	}

	for _, testCase := range testCases {
		t.Run(testCase.expectedString, func(t *testing.T) {
			actualString := testCase.confidence.String()
			if actualString != testCase.expectedString {
				t.Errorf("Expected string representation %s, but got %s", testCase.expectedString, actualString)
			}
		})
	}
}

func TestDefaultDetectors(t *testing.T) {
	// Verify that the default detectors are present
	if len(DefaultDetectors) < 1 {
		t.Errorf("Expected some default detectors, but got %d", len(DefaultDetectors))
	}

	fakeFS := fstest.MapFS{
		"Gemfile":          &fstest.MapFile{Data: []byte(""), Mode: 0644},
		"Gemfile.lock":     &fstest.MapFile{Data: []byte(""), Mode: 0644},
		"package.json":     &fstest.MapFile{Data: []byte(""), Mode: 0644},
		"requirements.txt": &fstest.MapFile{Data: []byte(""), Mode: 0644},
		"main.go":          &fstest.MapFile{Data: []byte(""), Mode: 0644},
		"index.js":         &fstest.MapFile{Data: []byte(""), Mode: 0644},
		"app.py":           &fstest.MapFile{Data: []byte(""), Mode: 0644},
	}

	for _, detector := range DefaultDetectors {
		t.Run(detector.GetTitle(), func(t *testing.T) {
			// Assume all detectors are FileDetectors right now
			det := detector.(*FileDetector)

			match, err := det.Detect(fakeFS)
			if err != nil {
				t.Fatal(err)
			}

			if !match.Detected {
				t.Errorf("Expected detection result to be true, but got false")
			}

			if !slices.Contains([]Confidence{High, Medium, Low}, match.Confidence) {
				t.Errorf("Expected confidence to be High, Medium or Low, but got %s", match.Confidence)
			}
		})
	}
}
